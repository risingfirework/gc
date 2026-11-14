"use client";

import { useState, useEffect } from "react";
import { CBTParticipant, CBTPublishSetting, ExamResult } from "@/services/api";
import ConfirmModal from "./ConfirmModal";

export default function CBTSettingsPanel({ items, saving, onToggle, onShuffle, onScreenLock, onScoreRelease, loadParticipants, unlockParticipant, resetParticipant, pauseParticipant, resumeParticipant, resetParticipantDevice, loadParticipantResult }: { items: CBTPublishSetting[]; saving: boolean; onToggle: (item: CBTPublishSetting, publish: boolean) => Promise<void> | void; onShuffle: (item: CBTPublishSetting, shuffleQuestions: boolean, shuffleOptions: boolean) => Promise<void> | void; onScreenLock: (item: CBTPublishSetting, screenLockEnabled: boolean, screenLockSeconds: number) => Promise<void> | void; onScoreRelease: (item: CBTPublishSetting, scoreRelease: string) => Promise<void> | void; loadParticipants: (examID: string) => Promise<CBTParticipant[]>; unlockParticipant?: (examID: string, userExamID: string) => Promise<void>; resetParticipant?: (examID: string, userExamID: string) => Promise<void>; pauseParticipant?: (examID: string, userExamID: string) => Promise<void>; resumeParticipant?: (examID: string, userExamID: string) => Promise<void>; resetParticipantDevice?: (examID: string, userExamID: string) => Promise<void>; loadParticipantResult?: (examID: string, userExamID: string) => Promise<ExamResult> }) {
  const [confirmItem, setConfirmItem] = useState<{ item: CBTPublishSetting; publish: boolean } | null>(null);
  const [busy, setBusy] = useState(false);
  const [selectedExam, setSelectedExam] = useState<CBTPublishSetting | null>(null);
  const [participants, setParticipants] = useState<CBTParticipant[]>([]);
  const [participantsLoading, setParticipantsLoading] = useState(false);
  const [participantsError, setParticipantsError] = useState("");
  const [unlocking, setUnlocking] = useState<string | null>(null);
  const [unlockError, setUnlockError] = useState("");
  const [resetting, setResetting] = useState<string | null>(null);
  const [resetError, setResetError] = useState("");
  const [pausing, setPausing] = useState<string | null>(null);
  const [resultModal, setResultModal] = useState<ExamResult | null>(null);
  const [resultLoading, setResultLoading] = useState<string | null>(null);
  const [settingsItem, setSettingsItem] = useState<CBTPublishSetting | null>(null);

  const openParticipants = async (item: CBTPublishSetting) => {
    if (selectedExam?.exam_id === item.exam_id) {
      setSelectedExam(null);
      setParticipants([]);
      return;
    }
    setSelectedExam(item);
    setParticipants([]);
    setParticipantsError("");
    setParticipantsLoading(true);
    try {
      setParticipants(await loadParticipants(item.exam_id));
    } catch (reason) {
      setParticipantsError(reason instanceof Error ? reason.message : "Gagal memuat daftar peserta.");
    } finally {
      setParticipantsLoading(false);
    }
  };

  useEffect(() => {
    if (!selectedExam) return;
    let active = true;
    const id = window.setInterval(async () => {
      if (!active) return;
      try {
        const fresh = await loadParticipants(selectedExam.exam_id);
        if (active) setParticipants(fresh);
      } catch { /* ignore */ }
    }, 8000);
    return () => { active = false; window.clearInterval(id); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedExam?.exam_id, loadParticipants]);

  const runUnlock = async (participant: CBTParticipant) => {
    if (!selectedExam || !unlockParticipant || unlocking) return;
    setUnlocking(participant.user_exam_id);
    setUnlockError("");
    try {
      await unlockParticipant(selectedExam.exam_id, participant.user_exam_id);
      setParticipants((current) => current.map((item) => item.user_exam_id === participant.user_exam_id ? { ...item, screen_locked: false, lock_until: undefined } : item));
    } catch (reason) {
      setUnlockError(reason instanceof Error ? reason.message : "Gagal membuka blokir.");
    } finally {
      setUnlocking(null);
    }
  };

  const runReset = async (participant: CBTParticipant) => {
    if (!selectedExam || !resetParticipant || resetting) return;
    setResetting(participant.user_exam_id);
    setResetError("");
    try {
      await resetParticipant(selectedExam.exam_id, participant.user_exam_id);
      setParticipants((current) => current.map((item) => item.user_exam_id === participant.user_exam_id ? { ...item, screen_locked: false, lock_until: undefined, lock_count: 0, last_lock_at: undefined } : item));
    } catch (reason) {
      setResetError(reason instanceof Error ? reason.message : "Gagal mereset blokir.");
    } finally {
      setResetting(null);
    }
  };

  const runPauseToggle = async (participant: CBTParticipant) => {
    if (!selectedExam || pausing) return;
    setPausing(participant.user_exam_id);
    try {
      if (participant.paused) {
        if (resumeParticipant) await resumeParticipant(selectedExam.exam_id, participant.user_exam_id);
        setParticipants((current) => current.map((item) => item.user_exam_id === participant.user_exam_id ? { ...item, paused: false, paused_at: undefined } : item));
      } else {
        if (pauseParticipant) await pauseParticipant(selectedExam.exam_id, participant.user_exam_id);
        setParticipants((current) => current.map((item) => item.user_exam_id === participant.user_exam_id ? { ...item, paused: true, paused_at: new Date().toISOString() } : item));
      }
    } catch {
      // ignore
    } finally {
      setPausing(null);
    }
  };

  const runResetDevice = async (participant: CBTParticipant) => {
    if (!selectedExam || !resetParticipantDevice) return;
    setPausing(participant.user_exam_id);
    try {
      await resetParticipantDevice(selectedExam.exam_id, participant.user_exam_id);
    } catch {
      // ignore
    } finally {
      setPausing(null);
    }
  };

  const runViewResult = async (participant: CBTParticipant) => {
    if (!selectedExam || !loadParticipantResult) return;
    setResultLoading(participant.user_exam_id);
    try {
      const r = await loadParticipantResult(selectedExam.exam_id, participant.user_exam_id);
      setResultModal(r);
    } catch {
      // ignore
    } finally {
      setResultLoading(null);
    }
  };

  const runToggle = async () => {
    if (!confirmItem) return;
    const pending = confirmItem;
    setBusy(true);
    try {
      await onToggle(pending.item, pending.publish);
      setConfirmItem(null);
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="card cbt-settings-card">
        {items.length === 0
          ? <p className="empty-state">Belum ada paket ujian CBT.</p>
          : <div className="cbt-settings-table-wrap"><table className="transactions-table cbt-settings-table"><thead><tr><th>Ujian</th><th>Peserta</th><th>Pembahasan</th><th></th></tr></thead><tbody>
            {items.map((item) => <tr key={item.exam_id} className={selectedExam?.exam_id === item.exam_id ? "active-row" : ""}>
              <td><button type="button" className="cbt-settings-exam-link" onClick={() => void openParticipants(item)}>{item.exam_title}</button><small>{item.package_title} · {item.package_kode} · {item.jenjang} · {item.total_questions} soal · {item.duration_minutes} menit</small></td>
              <td>{item.participated} siswa</td>
              <td><span className={`status-pill ${item.publish_pembahasan ? "paid" : "pending"}`}>{item.publish_pembahasan ? "Dipublish" : "Belum"}</span></td>
              <td><div className="user-row-actions"><button type="button" className="table-action" onClick={() => void openParticipants(item)}>{selectedExam?.exam_id === item.exam_id ? "Tutup" : "Peserta"}</button><button type="button" className="table-action" onClick={() => setSettingsItem(item)}>Kelola</button>{(item.participated > 0) && <button type="button" className="button small-btn" disabled={saving} onClick={() => setConfirmItem({ item, publish: !item.publish_pembahasan })}>{item.publish_pembahasan ? "Tarik" : "Publish"}</button>}</div></td>
            </tr>)}
          </tbody></table></div>}
      </div>
      {selectedExam && <div className="card cbt-participants-card">
        <div className="section-heading"><div><p className="eyebrow">Peserta ujian</p><h3>{selectedExam.exam_title}</h3><p className="muted">Siswa yang sedang mengerjakan atau sudah menyelesaikan ujian CBT ini.</p></div><div className="user-row-actions"><button type="button" className="button secondary small-btn" onClick={() => { const rows = [["Nama","Email","Jenjang","Status","Nilai","Lulus"]]; participants.forEach((p) => rows.push([p.name||"", p.email, p.school_level||"", p.status, String(p.total_score), p.passed?"Lulus":"Belum"])); const csv = rows.map((r) => r.map((c) => `"${String(c).replace(/"/g,'""')}"`).join(",")).join("\n"); const blob = new Blob([csv], { type: "text/csv" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = `peserta-${selectedExam.exam_id}.csv`; a.style.display = "none"; document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url); }}>Export CSV</button><button type="button" className="button secondary small-btn" onClick={() => { setSelectedExam(null); setParticipants([]); }}>Tutup</button></div></div>
        {participantsError && <p className="error">{participantsError}</p>}
        {unlockError && <p className="error">{unlockError}</p>}
        {resetError && <p className="error">{resetError}</p>}
        {participantsLoading ? <p className="empty-state">Memuat peserta...</p> : participants.length === 0 ? <p className="empty-state">Belum ada siswa yang mengikuti ujian ini.</p> : <div className="transactions-table-wrap"><table className="transactions-table"><thead><tr><th>Nama</th><th>Status</th><th>Soal</th><th>Aksi</th></tr></thead><tbody>{participants.map((participant) => {
          const ongoing = participant.status === "ongoing";
          const locked = ongoing && Boolean(participant.screen_locked);
          const startedStr = participant.started_at ? new Date(participant.started_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          const finishedStr = participant.finished_at ? new Date(participant.finished_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          const violations = participant.lock_count ?? 0;
          return <tr key={participant.user_exam_id} className={ongoing ? "active-row" : ""}>
            <td><b>{participant.name || "—"}</b><br/><small className="cbt-hint">{participant.email}</small></td>
            <td>{ongoing ? <span className="status-pill ongoing">Mengerjakan{participant.paused ? " · Dijeda" : ""}</span> : <span className="status-pill paid">{participant.passed ? "Lulus" : "Selesai"}</span>}</td>
            <td>{ongoing ? `Soal ${participant.current_question}/${participant.total_questions}` : `${participant.total_questions} soal`}</td>
            <td><div className="user-row-actions">
              {ongoing && (pauseParticipant || resumeParticipant) && <button type="button" className="table-action" disabled={pausing === participant.user_exam_id} onClick={() => void runPauseToggle(participant)}>{pausing === participant.user_exam_id ? "..." : participant.paused ? "Lanjutkan" : "Jeda"}</button>}
              {resetParticipantDevice && <button type="button" className="table-action" onClick={() => void runResetDevice(participant)}>Reset perangkat</button>}
              {unlockParticipant && locked && <button type="button" className="table-action" disabled={unlocking === participant.user_exam_id} onClick={() => void runUnlock(participant)}>{unlocking === participant.user_exam_id ? "..." : "Buka blokir"}</button>}
              {resetParticipant && violations > 0 && <button type="button" className="danger-action" disabled={resetting === participant.user_exam_id} onClick={() => void runReset(participant)}>{resetting === participant.user_exam_id ? "..." : "Reset blokir"}</button>}
              {!ongoing && loadParticipantResult && <button type="button" className="table-action" disabled={resultLoading === participant.user_exam_id} onClick={() => void runViewResult(participant)}>{resultLoading === participant.user_exam_id ? "Memuat..." : "Lihat hasil"}</button>}
            </div><small className="cbt-hint">{ongoing ? `Mulai ${startedStr}` : finishedStr}{violations > 0 ? ` · ${violations}× pelanggaran` : ""}</small></td>
          </tr>;
        })}</tbody></table></div>}
      </div>}
      {settingsItem && <div className="verify-modal-backdrop" role="presentation"><section className="verify-modal" role="dialog" aria-modal="true" style={{maxWidth:420}}><button type="button" className="verify-modal-close" onClick={() => setSettingsItem(null)}>×</button><p className="eyebrow">Pengaturan ujian</p><h2>{settingsItem.exam_title}</h2><div style={{display:"grid",gap:16,marginTop:12}}><ShuffleSettingsEditor item={settingsItem} saving={saving} onShuffle={onShuffle}/><ScreenLockSettingsEditor item={settingsItem} saving={saving} onScreenLock={onScreenLock}/><ScoreReleaseSettingsEditor item={settingsItem} saving={saving} onScoreRelease={onScoreRelease}/></div><div className="verify-modal-actions"><button type="button" className="verify-cancel" onClick={() => setSettingsItem(null)}>Tutup</button></div></section></div>}
      {resultModal && <div className="verify-modal-backdrop" role="presentation"><section className="verify-modal" role="dialog" aria-modal="true" style={{maxWidth:520}}><button type="button" className="verify-modal-close" onClick={() => setResultModal(null)}>×</button><p className="eyebrow">Hasil peserta</p><h2>{resultModal.title}</h2><p className="muted">Skor: {resultModal.total_score.toFixed(2)} · lulus {resultModal.passing_score} — {resultModal.passed ? "LULUS" : "TIDAK LULUS"}</p><p className="muted">{resultModal.correct_answers} benar · {resultModal.wrong_answers} salah · {resultModal.unanswered} kosong</p><div className="analytics-grid" style={{margin:"12px 0"}}>{resultModal.subjects.map((s) => <div className="subject-score" key={s.subject_name}><b>{s.subject_name}</b><span>{s.score.toFixed(2)}%</span><small>{s.correct_answers} benar · {s.wrong_answers} salah · {s.unanswered} kosong</small></div>)}</div>
<div style={{maxHeight:"50vh",overflow:"auto",margin:"12px 0",textAlign:"left"}}>{resultModal.review.map((q,idx) => <div key={q.question_id} style={{borderTop:"1px solid var(--border)",padding:"8px 0"}}><p><b>Soal {idx+1}.</b> {q.content_text}</p><p><small>Jawaban siswa: <span className={q.is_correct ? "answer-correct" : "answer-wrong"}>{q.selected_option || "—"}</span> · Kunci: <span className="answer-correct">{q.correct_answer}</span> · <span className={q.is_correct ? "answer-correct" : "answer-wrong"}>{q.is_correct ? "Benar" : "Salah"}</span></small></p>{q.explanation_text && <p><small>Pembahasan: {q.explanation_text}</small></p>}</div>)}</div>
<div className="verify-modal-actions"><button type="button" className="button" onClick={() => window.print()}>Cetak</button><button type="button" className="verify-cancel" onClick={() => setResultModal(null)}>Tutup</button></div></section></div>}
      {resultModal && <div className="question-bank-print"><header><p>HASIL UJIAN</p><h1>{resultModal.title}</h1><div><span>Skor: {resultModal.total_score.toFixed(2)}</span><span>Lulus: {resultModal.passing_score}</span><span>{resultModal.passed ? "LULUS" : "TIDAK LULUS"}</span></div></header><div className="print-student"><span>{resultModal.correct_answers} benar · {resultModal.wrong_answers} salah · {resultModal.unanswered} kosong</span></div>{resultModal.review.map((q,idx) => <div key={q.question_id} className="print-question"><span className="print-question-number">{idx+1}</span><div><p>{q.content_text}</p><small>Jawaban siswa: <span className={q.is_correct ? "answer-correct" : "answer-wrong"}>{q.selected_option || "—"}</span> | Kunci: <span className="answer-correct">{q.correct_answer}</span> | <span className={q.is_correct ? "answer-correct" : "answer-wrong"}>{q.is_correct ? "Benar" : "Salah"}</span></small>{q.explanation_text && <p><small>Pembahasan: {q.explanation_text}</small></p>}</div></div>)}</div>}
      {confirmItem && <ConfirmModal title={confirmItem.publish ? "Publish pembahasan?" : "Tarik pembahasan?"} message={confirmItem.publish ? `Siswa yang sudah mengerjakan "${confirmItem.item.exam_title}" akan dapat melihat analitik dan pembahasan hasilnya.` : `Siswa yang sudah mengerjakan "${confirmItem.item.exam_title}" tidak akan bisa melihat analitik hasilnya lagi.`} busy={busy} confirmLabel={confirmItem.publish ? "Ya, publish" : "Ya, tarik"} onClose={() => !busy && setConfirmItem(null)} onConfirm={() => void runToggle()} />}
    </>
  );
}

function ShuffleSettingsEditor({ item, saving, onShuffle }: { item: CBTPublishSetting; saving: boolean; onShuffle: (item: CBTPublishSetting, shuffleQuestions: boolean, shuffleOptions: boolean) => Promise<void> | void }) {
  const [shuffleQuestions, setShuffleQuestions] = useState(item.shuffle_questions);
  const [shuffleOptions, setShuffleOptions] = useState(item.shuffle_options);
  const [localBusy, setLocalBusy] = useState(false);
  const changed = shuffleQuestions !== item.shuffle_questions || shuffleOptions !== item.shuffle_options;
  const busy = saving || localBusy;
  const save = async () => {
    setLocalBusy(true);
    try {
      await onShuffle(item, shuffleQuestions, shuffleOptions);
    } finally {
      setLocalBusy(false);
    }
  };
  return <div className="cbt-shuffle-cell">
    <label className="checkbox-label"><input type="checkbox" disabled={busy} checked={shuffleQuestions} onChange={(event) => setShuffleQuestions(event.target.checked)} /> <span>Acak soal</span></label>
    <label className="checkbox-label"><input type="checkbox" disabled={busy} checked={shuffleOptions} onChange={(event) => setShuffleOptions(event.target.checked)} /> <span>Acak opsi</span></label>
    <button type="button" className="button small-btn" disabled={busy || !changed} onClick={() => void save()}>{busy ? "Menyimpan..." : "Simpan acak"}</button>
  </div>;
}

function ScreenLockSettingsEditor({ item, saving, onScreenLock }: { item: CBTPublishSetting; saving: boolean; onScreenLock: (item: CBTPublishSetting, screenLockEnabled: boolean, screenLockSeconds: number) => Promise<void> | void }) {
  const [enabled, setEnabled] = useState(item.screen_lock_enabled);
  const [seconds, setSeconds] = useState(item.screen_lock_seconds);
  const [localBusy, setLocalBusy] = useState(false);
  const changed = enabled !== item.screen_lock_enabled || seconds !== item.screen_lock_seconds;
  const busy = saving || localBusy;
  const save = async () => {
    setLocalBusy(true);
    try {
      await onScreenLock(item, enabled, seconds);
    } finally {
      setLocalBusy(false);
    }
  };
  return <div className="cbt-shuffle-cell">
    <label className="checkbox-label"><input type="checkbox" disabled={busy} checked={enabled} onChange={(event) => setEnabled(event.target.checked)} /> <span>Aktifkan blokir</span></label>
    <label className="cbt-hint" htmlFor={`screen-lock-seconds-${item.exam_id}`}>Durasi kunci (detik)</label>
    <input id={`screen-lock-seconds-${item.exam_id}`} type="number" min={1} max={300} className="cbt-lock-seconds" disabled={busy || !enabled} value={seconds} onChange={(event) => setSeconds(Math.max(1, Math.min(300, Number(event.target.value) || 1)))} />
    <button type="button" className="button small-btn" disabled={busy || !changed || seconds < 1 || seconds > 300} onClick={() => void save()}>{busy ? "Menyimpan..." : "Simpan blokir"}</button>
  </div>;
}

function ScoreReleaseSettingsEditor({ item, saving, onScoreRelease }: { item: CBTPublishSetting; saving: boolean; onScoreRelease: (item: CBTPublishSetting, scoreRelease: string) => Promise<void> | void }) {
  const [release, setRelease] = useState(item.score_release || "after_finish");
  const [localBusy, setLocalBusy] = useState(false);
  const changed = release !== (item.score_release || "after_finish");
  const busy = saving || localBusy;
  const save = async () => {
    setLocalBusy(true);
    try {
      await onScoreRelease(item, release);
    } finally {
      setLocalBusy(false);
    }
  };
  return <div className="cbt-shuffle-cell">
    <label className="cbt-hint" htmlFor={`score-release-${item.exam_id}`}>Waktu rilis</label>
    <select id={`score-release-${item.exam_id}`} className="cbt-score-release" disabled={busy} value={release} onChange={(event) => setRelease(event.target.value)}>
      <option value="after_finish">Setelah selesai</option>
      <option value="with_pembahasan">Bareng pembahasan</option>
    </select>
    <button type="button" className="button small-btn" disabled={busy || !changed} onClick={() => void save()}>{busy ? "Menyimpan..." : "Simpan rilis"}</button>
  </div>;
}