import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080/api/v1').replace(/\/$/, '');
const EXAM_ID = __ENV.EXAM_ID || '30000000-0000-0000-0000-000000000001';
const MAX_VUS = Number.parseInt(__ENV.MAX_VUS || '20000', 10);
const ANSWER_INTERVAL_SECONDS = Number.parseInt(__ENV.ANSWER_INTERVAL_SECONDS || '5', 10);
const SESSION_DURATION_SECONDS = Number.parseInt(__ENV.SESSION_DURATION_SECONDS || '1200', 10);

if (!Number.isInteger(MAX_VUS) || MAX_VUS < 1 || MAX_VUS > 20000) {
  throw new Error('MAX_VUS harus berada pada rentang 1..20000; seed menyediakan 20.000 akun.');
}

export const options = {
  scenarios: {
    concurrent_exam_takers: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || '10m', target: MAX_VUS },
        { duration: __ENV.STEADY || '20m', target: MAX_VUS },
        { duration: __ENV.RAMP_DOWN || '2m', target: 0 },
      ],
      gracefulRampDown: '5m',
    },
  },
  thresholds: {
    http_req_duration: [{ threshold: 'p(95)<150', abortOnFail: false }],
    http_req_failed: [{ threshold: 'rate<0.001', abortOnFail: true, delayAbortEval: '1m' }],
    checks: ['rate>0.999'],
  },
  userAgent: 'tka-k6-load-test/1.0',
  discardResponseBodies: false,
};

const jsonHeaders = { 'Content-Type': 'application/json', Accept: 'application/json' };

function parseJSON(response, operation) {
  try {
    return response.json();
  } catch (_) {
    fail(`${operation}: respons bukan JSON (status=${response.status})`);
  }
}

export default function () {
  // Each seeded account may only own one attempt for this exam. Keep a VU idle
  // after its first full journey instead of creating an invalid second attempt.
  if (exec.vu.iterationInScenario > 0) {
    sleep(60);
    return;
  }

  const accountNumber = exec.vu.idInTest;
  const login = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: `loadtest+${accountNumber}@tka.local`,
    password: 'TkaLoad123!',
  }), { headers: jsonHeaders, tags: { name: 'POST /auth/login' }, timeout: '10s' });

  if (!check(login, { 'login 200': (response) => response.status === 200 })) return;
  const loginBody = parseJSON(login, 'login');
  const token = loginBody.access_token;
  if (!token) fail('login: access_token kosong');
  const auth = { ...jsonHeaders, Authorization: `Bearer ${token}` };

  const start = http.get(`${BASE_URL}/exams/${EXAM_ID}/start`, {
    headers: auth,
    tags: { name: 'GET /exams/:id/start' },
    timeout: '10s',
  });
  if (!check(start, { 'start exam 200': (response) => response.status === 200 })) return;
  const exam = parseJSON(start, 'start exam');
  if (!exam.user_exam_id || !Array.isArray(exam.questions) || exam.questions.length === 0) {
    fail('start exam: payload sesi/soal tidak lengkap');
  }

  const sessionStartedAt = Date.now();
  let answerIndex = 0;
  while ((Date.now() - sessionStartedAt) / 1000 < SESSION_DURATION_SECONDS) {
    const question = exam.questions[answerIndex % exam.questions.length];
    sleep(ANSWER_INTERVAL_SECONDS);
    const options = question.options || [];
    const selected = options[(accountNumber + Number.parseInt(question.id.slice(-2), 16)) % options.length]?.key;
    if (!selected) fail(`soal ${question.id} tidak memiliki opsi`);
    const save = http.post(`${BASE_URL}/cbt/answers/sync`, JSON.stringify({
      exam_id: exam.user_exam_id,
      question_id: question.id,
      selected_option: selected,
    }), { headers: auth, tags: { name: 'POST /cbt/answers/sync' }, timeout: '10s' });
    if (!check(save, { 'autosave 200': (response) => response.status === 200 })) return;
    answerIndex += 1;
  }

  const submit = http.post(`${BASE_URL}/exams/${exam.user_exam_id}/submit`, '{}', {
    headers: auth,
    tags: { name: 'POST /exams/:id/submit' },
    timeout: '30s',
  });
  check(submit, {
    'submit 200': (response) => response.status === 200,
    'submitted status': (response) => {
      try { return response.json('status') === 'submitted'; } catch (_) { return false; }
    },
  });
}
