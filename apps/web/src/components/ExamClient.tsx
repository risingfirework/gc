"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import Link from "next/link";
import Image from "next/image";
import { APIError, api, ExamStartResponse, SiteSettings, SubmitExamResponse } from "@/services/api";
import { AntiCheatEvent, useAntiCheat } from "@/hooks/useAntiCheat";
import { useCBTAutoSave } from "@/hooks/useCBTAutoSave";
import { useOnlineStatus } from "@/hooks/useOnlineStatus";
import BrandLogo from "./BrandLogo";
import RichText from "@/components/RichText";

type StoredExamState = { answers: Record<string, string>; doubtful: string[] };

export default function ExamClient({ examID, initialToken }: { examID: string; initialToken?: string }) {
  const [settings, setSettings] = useState<SiteSettings>();
  const [exam, setExam] = useState<ExamStartResponse>();
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [doubtful, setDoubtful] = useState<Set<string>>(new Set());
  const [index, setIndex] = useState(0);
  const [remainingSeconds, setRemainingSeconds] = useState(0);
  const [clockOffset, setClockOffset] = useState(0);
  const [result, setResult] = useState<SubmitExamResponse>();
  const [error, setError] = useState("");
  const [starting, setStarting] = useState(true);
  const [tokenGate, setTokenGate] = useState(false);
  const [tokenInvalid, setTokenInvalid] = useState("");
  const [cbtToken, setCbtToken] = useState("");
  const [attemptedToken, setAttemptedToken] = useState(initialToken ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [confirmSubmit, setConfirmSubmit] = useState(false);
  const [violations, setViolations] = useState<AntiCheatEvent[]>([]);
  const online = useOnlineStatus();
  const submissionStarted = useRef(false);
  const { queueAnswer, flushAll, state: autoSaveState } = useCBTAutoSave(exam?.user_exam_id ?? null);

  useEffect(() => { api.siteSettings().then(setSettings).catch(() => undefined); }, []);
  useEffect(() => {
    let active = true;
    const start = api.startExam(examID, attemptedToken);
    start.then((payload) => {
      if (!active) return;
      const offset = Date.parse(payload.server_time) - Date.now();
      setClockOffset(offset);
      setExam(payload);
      setRemainingSeconds(payload.remaining_seconds);
      try {
        const stored = localStorage.getItem(`tka:cbt:${payload.user_exam_id}`);
        if (stored) {
          const parsed = JSON.parse(stored) as StoredExamState;
          setAnswers(parsed.answers ?? {});
          setDoubtful(new Set(parsed.doubtful ?? []));
        }
      } catch { localStorage.removeItem(`tka:cbt:${payload.user_exam_id}`); }
    }).catch((reason: unknown) => {
      if (!active) return;
      const message = reason instanceof Error ? reason.message : "";
      if (message.includes("memerlukan kode rahasia")) {
        setTokenGate(true);
        setTokenInvalid("");
      } else if (message.includes("salah")) {
        setTokenGate(true);
        setTokenInvalid(message);
      } else {
        setTokenGate(false);
        setError(message || "Ujian tidak dapat dimuat.");
      }
    }).finally(() => { if (active) setStarting(false); });
    return () => { active = false; };
  }, [examID, attemptedToken]);

  useEffect(() => {
    if (!exam) return;
    localStorage.setItem(`tka:cbt:${exam.user_exam_id}`, JSON.stringify({ answers, doubtful: [...doubtful] } satisfies StoredExamState));
  }, [answers, doubtful, exam]);

  const submitAttempt = useCallback(async (automatic = false) => {
    if (!exam || submissionStarted.current) return;
    submissionStarted.current = true;
    setSubmitting(true);
    setError("");
    try {
      await flushAll();
      const submitted = await api.submitExam(exam.user_exam_id);
      localStorage.removeItem(`tka:cbt:${exam.user_exam_id}`);
      setResult(submitted);
    } catch (reason) {
      submissionStarted.current = false;
      const message = reason instanceof APIError && reason.networkError
        ? "Jawaban tersimpan di perangkat. Sambungkan internet lalu coba submit kembali."
        : reason instanceof Error ? reason.message : "Ujian gagal disubmit.";
      setError(automatic ? `Waktu habis. ${message}` : message);
    } finally { setSubmitting(false); }
  }, [exam, flushAll]);

  useEffect(() => {
    if (!exam || result) return;
    const update = () => {
      const seconds = Math.max(0, Math.ceil((Date.parse(exam.ends_at) - (Date.now() + clockOffset)) / 1000));
      setRemainingSeconds(seconds);
      if (seconds === 0) void submitAttempt(true);
    };
    update();
    const timer = window.setInterval(update, 250);
    window.addEventListener("focus", update);
    document.addEventListener("visibilitychange", update);
    return () => { window.clearInterval(timer); window.removeEventListener("focus", update); document.removeEventListener("visibilitychange", update); };
  }, [clockOffset, exam, result, submitAttempt]);

  const reportViolation = useCallback((event: AntiCheatEvent) => {
    setViolations((current) => [...current.slice(-49), event]);
  }, []);
  useAntiCheat(Boolean(exam && !result), reportViolation);

  const question = exam?.questions[index];
  const formattedTime = useMemo(() => {
    const hours = Math.floor(remainingSeconds / 3600);
    const minutes = Math.floor((remainingSeconds % 3600) / 60);
    const seconds = remainingSeconds % 60;
    return hours > 0 ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}` : `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }, [remainingSeconds]);

  function selectAnswer(questionID: string, selectedOption: string) {
    setAnswers((current) => ({ ...current, [questionID]: selectedOption }));
    queueAnswer(questionID, selectedOption);
  }

  function toggleMultipleAnswer(questionID: string, key: string) {
    const selected = new Set((answers[questionID] ?? "").split(",").filter(Boolean));
    if (selected.has(key)) selected.delete(key); else selected.add(key);
    if (selected.size > 0) selectAnswer(questionID, [...selected].sort().join(","));
  }

  function selectCategoryAnswer(questionID: string, statementKey: string, category: string) {
    const selected = Object.fromEntries((answers[questionID] ?? "").split(";").map((part) => part.split("=", 2)).filter((part) => part.length === 2));
    selected[statementKey] = category;
    selectAnswer(questionID, Object.keys(selected).sort().map((key) => `${key}=${selected[key]}`).join(";"));
  }

  function toggleDoubtful(questionID: string) {
    setDoubtful((current) => {
      const next = new Set(current);
      if (next.has(questionID)) next.delete(questionID); else next.add(questionID);
      return next;
    });
  }

  if (result) return <main className="auth"><section className="card result-card"><p className="eyebrow">Hasil ujian</p><div className="result-score">{result.total_score.toFixed(2)}</div><h2 className={result.passed ? "passed" : "failed"}>{result.passed ? "LULUS" : "TIDAK LULUS"}</h2><p>Passing score: {result.passing_score}</p><div className="actions"><Link className="button" href={`/dashboard/results/${result.user_exam_id}`}>Lihat analitik</Link><Link className="button secondary" href="/dashboard">Dashboard</Link></div></section></main>;
  if (tokenGate && !exam) return <main className="auth"><section className="card exam-token-card"><p className="eyebrow">Ujian CBT</p><h2>Masukkan kode rahasia (token)</h2><p className="auth-intro">Masukkan token CBT yang diberikan oleh pengawas atau guru Anda untuk memulai ujian.</p>{tokenInvalid && <p className="error">{tokenInvalid}</p>}<form onSubmit={(event) => { event.preventDefault(); setTokenInvalid(""); setError(""); setStarting(true); setAttemptedToken(cbtToken.trim().toUpperCase()); }}><input className="exam-token-input" autoFocus value={cbtToken} onChange={(event) => setCbtToken(event.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ""))} maxLength={8} placeholder="cth: 9K4PT2" required /><button className="button full" type="submit" disabled={starting}>{starting ? "Memeriksa..." : "Mulai ujian"}</button></form><Link className="button secondary exam-token-back" href="/dashboard">Kembali ke dashboard</Link></section></main>;
  if (!exam && error) return <main className="auth"><section className="card"><p className="error">{error}</p><Link className="button secondary" href="/dashboard">Kembali</Link></section></main>;
  if (!exam) return <main className="auth"><div className="exam-loader"/><p>Menyiapkan ujian dan menyinkronkan waktu...</p></main>;
  if (!question) return <main className="auth"><p className="error">Soal tidak tersedia.</p></main>;

  const isAnswered = (item: typeof exam.questions[number]) => {
    const answer = answers[item.id] ?? "";
    if (item.question_type === "multiple_choice") return answer.split(",").filter(Boolean).length >= 2;
    if (item.question_type === "category") return answer.split(";").filter(Boolean).length === item.options.length;
    if (item.question_type === "essay") return answer.trim() !== "";
    return answer !== "";
  };
  const answeredCount = exam.questions.filter(isAnswered).length;
  return <main className="cbt-page">
    {!online && <div className="connection-banner offline">Offline — jawaban ditahan di perangkat dan dikirim otomatis saat tersambung.</div>}
    {online && autoSaveState === "saved" && <div className="connection-banner saved">Jawaban tersimpan</div>}
    <div className="container">
      <header className="exam-header">
        <div><div className="brand"><BrandLogo logoDataURL={settings?.logo_data_url}/>{settings?.platform_name??""}</div><p>{exam.title}</p></div>
        <div className="timer-box"><small>Sisa waktu server</small><strong>{formattedTime}</strong></div>
      </header>
      {error && <div className="error exam-error">{error}<button onClick={() => void submitAttempt(false)} disabled={!online || submitting}>Coba submit lagi</button></div>}
      <div className="exam-shell">
        <section className="card question-card">
          <div className="question-meta"><span>{question.subject_name}</span><span>Soal {index + 1} dari {exam.questions.length}</span></div>
          {question.presentation_type === "group" && <div className="group-stimulus"><small>Stimulus {question.group_code}</small><RichText content={question.stimulus_text}/><QuestionImage src={question.stimulus_image_url} alt="Gambar stimulus soal"/></div>}
          <div className="question-content"><RichText content={question.content_text}/><QuestionImage src={question.question_image_url} alt="Gambar pertanyaan"/></div>
          {question.question_type === "single_choice" && <div className="answer-list">{question.options.map((option) => <button type="button" className={`option ${answers[question.id] === option.key ? "selected" : ""}`} onClick={() => selectAnswer(question.id, option.key)} key={option.key}><span className="option-key">{option.key}</span><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pilihan ${option.key}`}/></span></button>)}</div>}
          {question.question_type === "essay" && <><p className="answer-instruction">Tuliskan jawaban uraian Anda.</p><textarea className="essay-answer" value={answers[question.id] ?? ""} onChange={(event) => selectAnswer(question.id, event.target.value)} placeholder="Ketik jawaban esai Anda di sini..." rows={8}/></>}
          {question.question_type === "multiple_choice" && <><p className="answer-instruction">Pilih semua jawaban yang benar.</p><div className="answer-list">{question.options.map((option) => { const selected = (answers[question.id] ?? "").split(",").includes(option.key); return <button type="button" role="checkbox" aria-checked={selected} className={`option ${selected ? "selected" : ""}`} onClick={() => toggleMultipleAnswer(question.id, option.key)} key={option.key}><span className="option-key">{selected ? "✓" : option.key}</span><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pilihan ${option.key}`}/></span></button>; })}</div></>}
          {question.question_type === "category" && <><p className="answer-instruction">Tentukan kategori untuk setiap pernyataan.</p><div className="category-answer-table">{question.options.map((option) => { const selected = Object.fromEntries((answers[question.id] ?? "").split(";").map((part) => part.split("=", 2)).filter((part) => part.length === 2))[option.key]; return <div className="category-answer-row" key={option.key}><div><b>{option.key}</b><span><RichText content={option.content}/><QuestionImage src={option.image_url} alt={`Gambar pernyataan ${option.key}`}/></span></div><div>{question.category_labels.map((label) => <button type="button" className={selected === label ? "selected" : ""} onClick={() => selectCategoryAnswer(question.id, option.key, label)} key={label}>{label}</button>)}</div></div>; })}</div></>}
          <footer className="exam-actions">
            <button className="button secondary" disabled={index === 0} onClick={() => setIndex((value) => value - 1)}>Sebelumnya</button>
            <button className={`doubt-button ${doubtful.has(question.id) ? "active" : ""}`} onClick={() => toggleDoubtful(question.id)}>⚑ {doubtful.has(question.id) ? "Batalkan ragu" : "Ragu-ragu"}</button>
            {index < exam.questions.length - 1
              ? <button className="button" onClick={() => setIndex((value) => value + 1)}>Selanjutnya</button>
              : <button className="button submit-button" disabled={submitting || confirmSubmit} onClick={() => setConfirmSubmit(true)}>{submitting ? "Mengirim..." : "Submit ujian"}</button>}
          </footer>
        </section>
        <aside className="card question-panel">
          <div className="panel-summary"><strong>{answeredCount}/{exam.questions.length}</strong><span>soal dijawab</span></div>
          <div className="numbers">{exam.questions.map((item, itemIndex) => {
            const status = doubtful.has(item.id) ? "doubtful" : isAnswered(item) ? "done" : "empty";
            return <button type="button" aria-label={`Soal ${itemIndex + 1}, ${status}`} className={`number ${status} ${itemIndex === index ? "current" : ""}`} onClick={() => setIndex(itemIndex)} key={item.id}>{itemIndex + 1}</button>;
          })}</div>
          <div className="legend"><span><i className="done"/>Dijawab</span><span><i className="doubtful"/>Ragu-ragu</span><span><i/>Belum</span></div>
          <p className="security-note">Mode ujian aktif · {violations.length} aktivitas keluar halaman tercatat</p>
        </aside>
      </div>
    </div>
    {confirmSubmit && createPortal(<div className="verify-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !submitting) setConfirmSubmit(false); }}><section className="verify-modal" role="dialog" aria-modal="true" aria-labelledby="submit-exam-title" onMouseDown={(event) => event.stopPropagation()}><button type="button" className="verify-modal-close" disabled={submitting} onClick={() => setConfirmSubmit(false)}>×</button><div className="verify-modal-icon verify-modal-icon--success" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 13l4 4L19 7"/></svg></div><p className="eyebrow">Akhiri ujian</p><h2 id="submit-exam-title">Kumpulkan ujian sekarang?</h2><p className="muted">{answeredCount === exam.questions.length ? `Semua ${answeredCount} soal sudah terjawab. Setelah dikumpulkan, jawaban tidak dapat diubah.` : `${answeredCount} dari ${exam.questions.length} soal terjawab${exam.questions.length - answeredCount > 0 ? `, ${exam.questions.length - answeredCount} soal masih kosong` : ""}. Setelah dikumpulkan, jawaban tidak dapat diubah.`}</p><div className="verify-modal-actions"><button type="button" className="verify-cancel" disabled={submitting} onClick={() => setConfirmSubmit(false)}>Batal</button><button type="button" className="verify-confirm-approve" disabled={submitting} onClick={() => { setConfirmSubmit(false); void submitAttempt(false); }}>{submitting ? "Mengirim..." : "Ya, kumpulkan"}</button></div></section></div>, document.body)}
  </main>;
}

function QuestionImage({ src, alt }: { src?: string; alt: string }) {
  return src ? <Image className="question-media" src={src} alt={alt} width={800} height={500} unoptimized/> : null;
}
