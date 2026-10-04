// Push bank soal Matematika SD (30 soal) ke VPS lewat API admin, sekaligus
// membuat paket CBT + ujian + seluruh soal dalam satu transaksi.
//
// Pemakaian (token/akun via environment):
//   $env:TKA_ADMIN_EMAIL="admin@..." ; $env:TKA_ADMIN_PASSWORD="..."
//   node tools/bank-soal/push-bank-soal-matematika-sd.mjs
//
// Variabel opsional:
//   TKA_API_BASE   default https://siap-siap.my.id/api/v1
//   TKA_CBT_TOKEN  default MATH6SD
//   TKA_KODE       default TKA-MATH6SD
//   TKA_FORCE=1    tetap buat walau kode sudah ada
//
// Catatan: rahasia hanya dibaca dari environment, tidak ditulis ke file/log.

import { QUESTIONS } from "./generate-bank-soal-matematika-sd.mjs";

const BASE = (process.env.TKA_API_BASE || "https://siap-siap.my.id/api/v1").replace(/\/$/, "");
const EMAIL = process.env.TKA_ADMIN_EMAIL;
const PASSWORD = process.env.TKA_ADMIN_PASSWORD;
const CBT_TOKEN = process.env.TKA_CBT_TOKEN || "MATH6SD";
const KODE = process.env.TKA_KODE || "TKA-MATH6SD";
const FORCE = process.env.TKA_FORCE === "1";

if (!EMAIL || !PASSWORD) {
  console.error("Set TKA_ADMIN_EMAIL dan TKA_ADMIN_PASSWORD lebih dahulu.");
  process.exit(1);
}

async function api(path, { method = "GET", body, token } = {}) {
  const headers = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!res.ok) {
    const detail = data && typeof data === "object" ? JSON.stringify(data) : String(data);
    throw new Error(`${method} ${path} -> HTTP ${res.status}: ${detail}`);
  }
  return data;
}

function pick(items, name) {
  const found = items.find((item) => String(item.nama || "").toLowerCase() === name.toLowerCase());
  if (!found) {
    throw new Error(`Master "${name}" tidak ditemukan. Tersedia: ${items.map((i) => i.nama).join(", ")}`);
  }
  return found.id;
}

const questionPayload = QUESTIONS.map((item) => ({
  subject_name: "Matematika",
  chapter_name: item.chapter ?? "",
  content_text: item.text,
  question_type: item.t,
  presentation_type: "single",
  group_code: "",
  stimulus_text: "",
  question_image_url: "",
  stimulus_image_url: "",
  category_labels: item.labels ?? [],
  options: item.opts
    ? Object.keys(item.opts).map((key) => ({ key, content: item.opts[key], image_url: "" }))
    : [],
  correct_answer: item.ans,
  score_weight: 1,
  explanation_text: item.exp,
  status: "active",
}));

async function main() {
  console.log(`login sebagai ${EMAIL} ...`);
  const login = await api("/auth/login", { method: "POST", body: { email: EMAIL, password: PASSWORD } });
  if (login.requires_2fa) {
    throw new Error("Akun memerlukan 2FA. Selesaikan login dengan kode TOTP dan pakai token dari browser.");
  }
  const token = login.access_token;
  if (!token) throw new Error("Tidak menerima access_token.");
  console.log(`login OK (role=${login.user?.role})`);

  const [mapels, kategoris, kelas] = await Promise.all([
    api("/admin/master/mapel", { token }),
    api("/admin/master/kategori", { token }),
    api("/admin/master/kelas", { token }),
  ]);
  const mapelId = pick(mapels.items, "Matematika");
  const kategoriId = pick(kategoris.items, "Tryout");
  const kelasId = pick(kelas.items, "Kelas 6 SD");
  console.log(`master: mapel=${mapelId} kategori=${kategoriId} kelas=${kelasId}`);

  if (!FORCE) {
    const existing = await api(`/admin/packages?q=${encodeURIComponent(KODE)}&per_page=100`, { token });
    const dup = (existing.items || []).find((item) => String(item.kode || "").toUpperCase() === KODE.toUpperCase());
    if (dup) {
      console.log(`Paket dengan kode ${KODE} sudah ada (id=${dup.id}, status=${dup.status}). Set TKA_FORCE=1 untuk tetap membuat.`);
      return;
    }
  }

  const bundle = {
    package: {
      title: "Tryout Matematika SD - Kelas 6",
      kode: KODE,
      description: "Bank soal contoh Matematika jenjang SD: 30 soal (pilihan ganda, PG majemuk, benar/salah, esai) lengkap dengan pembahasan.",
      price: 0,
      validity_days: 30,
      status: "active",
      jenjang: "SD",
      exam_type: "cbt",
      cbt_token: CBT_TOKEN,
      start_date: "",
      end_date: "",
      kategori_id: kategoriId,
      kelas_id: kelasId,
    },
    exam: {
      package_id: "",
      title: "Tryout Matematika SD",
      mapel_id: mapelId,
      tahun_ajaran_id: "",
      duration_minutes: 60,
      total_questions: questionPayload.length,
      passing_score: 60,
      status: "active",
      shuffle_questions: true,
      shuffle_options: true,
    },
    questions: questionPayload,
  };

  console.log(`membuat paket + ujian + ${questionPayload.length} soal ...`);
  const created = await api("/admin/packages/bundle", { method: "POST", token, body: bundle });
  const pkg = created.package;
  console.log(`paket dibuat: id=${pkg.id} kode=${pkg.kode} status=${pkg.status} token=${pkg.cbt_token}`);

  const exams = await api(`/admin/exams?package_id=${pkg.id}&per_page=50`, { token });
  const exam = (exams.items || [])[0];
  if (exam) {
    console.log(`ujian dibuat: id=${exam.id} mapel_id=${exam.mapel_id}`);
    await api(`/admin/cbt-settings/${exam.id}/shuffle`, {
      method: "PATCH",
      token,
      body: { shuffle_questions: true, shuffle_options: true },
    });
    console.log("pengacakan soal+opsi diaktifkan.");
  }

  const lookup = await api(`/cbt/lookup?token=${encodeURIComponent(CBT_TOKEN)}`, { token });
  const reachable = (lookup.data || []).some((item) => item.id === pkg.id);
  console.log(`terjangkau lewat token CBT: ${reachable ? "YA" : "TIDAK"} (paket CBT sengaja tidak tampil di katalog publik).`);
  console.log("selesai.");
}

await main();
