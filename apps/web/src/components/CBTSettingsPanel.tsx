"use client";

import { useState } from "react";
import { CBTParticipant, CBTPublishSetting } from "@/services/api";
import ConfirmModal from "./ConfirmModal";

export default function CBTSettingsPanel({ items, saving, onToggle, onShuffle, onScreenLock, loadParticipants, unlockParticipant, resetParticipant }: { items: CBTPublishSetting[]; saving: boolean; onToggle: (item: CBTPublishSetting, publish: boolean) => Promise<void> | void; onShuffle: (item: CBTPublishSetting, shuffleQuestions: boolean, shuffleOptions: boolean) => Promise<void> | void; onScreenLock: (item: CBTPublishSetting, screenLockEnabled: boolean, screenLockSeconds: number) => Promise<void> | void; loadParticipants: (examID: string) => Promise<CBTParticipant[]>; unlockParticipant?: (examID: string, userExamID: string) => Promise<void>; resetParticipant?: (examID: string, userExamID: string) => Promise<void> }) {
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
          : <div className="cbt-settings-table-wrap"><table className="transactions-table cbt-settings-table"><thead><tr><th>Ujian</th><th>Pembuat</th><th>Peserta</th><th>Pengacakan</th><th>Blokir layar</th><th>Pembahasan</th><th></th></tr></thead><tbody>
            {items.map((item) => <tr key={item.exam_id} className={selectedExam?.exam_id === item.exam_id ? "active-row" : ""}>
              <td><button type="button" className="cbt-settings-exam-link" onClick={() => void openParticipants(item)}>{item.exam_title}</button><small>{item.package_title} · {item.package_kode} · {item.jenjang} · {item.total_questions} soal · {item.duration_minutes} menit · Syarat lulus {item.passing_score.toFixed(0)}</small></td>
              <td>{item.publisher_email || "Platform"}</td>
              <td>{item.participated} siswa</td>
              <td><ShuffleSettingsEditor item={item} saving={saving} onShuffle={onShuffle}/></td>
              <td><ScreenLockSettingsEditor item={item} saving={saving} onScreenLock={onScreenLock}/></td>
              <td><span className={`status-pill ${item.publish_pembahasan ? "paid" : "pending"}`}>{item.publish_pembahasan ? "Dipublish" : "Tidak dipublish"}</span></td>
              <td><div className="user-row-actions"><button type="button" className="table-action" onClick={() => void openParticipants(item)}>{selectedExam?.exam_id === item.exam_id ? "Tutup peserta" : "Lihat peserta"}</button>{(item.participated > 0) && <button type="button" className="button small-btn" disabled={saving} onClick={() => setConfirmItem({ item, publish: !item.publish_pembahasan })}>{item.publish_pembahasan ? "Tarik pembahasan" : "Publish pembahasan"}</button>}</div></td>
            </tr>)}
          </tbody></table></div>}
      </div>
      {selectedExam && <div className="card cbt-participants-card">
        <div className="section-heading"><div><p className="eyebrow">Peserta ujian</p><h3>{selectedExam.exam_title}</h3><p className="muted">Siswa yang sedang mengerjakan atau sudah menyelesaikan ujian CBT ini.</p></div><button type="button" className="button secondary small-btn" onClick={() => { setSelectedExam(null); setParticipants([]); }}>Tutup</button></div>
        {participantsError && <p className="error">{participantsError}</p>}
        {unlockError && <p className="error">{unlockError}</p>}
        {resetError && <p className="error">{resetError}</p>}
        {participantsLoading ? <p className="empty-state">Memuat peserta...</p> : participants.length === 0 ? <p className="empty-state">Belum ada siswa yang mengikuti ujian ini.</p> : <div className="transactions-table-wrap"><table className="transactions-table"><thead><tr><th>Nama</th><th>Email</th><th>Jenjang</th><th>Soal</th><th>Status</th><th>Blokir layar</th><th>Waktu</th></tr></thead><tbody>{participants.map((participant) => {
          const ongoing = participant.status === "ongoing";
          const locked = ongoing && Boolean(participant.screen_locked);
          const startedStr = participant.started_at ? new Date(participant.started_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          const finishedStr = participant.finished_at ? new Date(participant.finished_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          const lockUntilStr = participant.lock_until ? new Date(participant.lock_until).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", second: "2-digit" }) : "";
          const violations = participant.lock_count ?? 0;
          return <tr key={participant.user_exam_id} className={ongoing ? "active-row" : ""}><td>{participant.name || "—"}</td><td>{participant.email}</td><td>{participant.school_level || "—"}</td><td>{ongoing ? `Soal ${participant.current_question}/${participant.total_questions}` : `${participant.total_questions} soal`}</td><td>{ongoing ? <span className="status-pill ongoing">Sedang mengerjakan</span> : <span>{<span className="status-pill paid">Selesai</span>} <small className="cbt-hint">· {participant.passed ? "Lulus" : "Belum lulus"}</small></span>}</td><td>{!ongoing ? <span className="cbt-hint">—</span> : <div className="user-row-actions">{locked ? <span className="status-pill rejected">Terkunci</span> : <span className="cbt-hint">Normal</span>}{unlockParticipant && locked && <button type="button" className="table-action" disabled={unlocking === participant.user_exam_id} onClick={() => void runUnlock(participant)}>{unlocking === participant.user_exam_id ? "Membuka..." : "Buka blokir"}</button>}{resetParticipant && violations > 0 && <button type="button" className="danger-action" disabled={resetting === participant.user_exam_id} onClick={() => void runReset(participant)}>{resetting === participant.user_exam_id ? "Mereset..." : "Reset blokir"}</button>}<small className="cbt-hint">{violations}×{lockUntilStr ? ` · sampai ${lockUntilStr}` : ""}</small></div>}</td><td>{ongoing ? <span className="cbt-hint">Mulai {startedStr}</span> : finishedStr}</td></tr>;
        })}</tbody></table></div>}
      </div>}
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
