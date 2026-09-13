"use client";

import { useState } from "react";
import { CBTParticipant, CBTPublishSetting } from "@/services/api";
import ConfirmModal from "./ConfirmModal";

export default function CBTSettingsPanel({ items, saving, onToggle, loadParticipants }: { items: CBTPublishSetting[]; saving: boolean; onToggle: (item: CBTPublishSetting, publish: boolean) => Promise<void> | void; loadParticipants: (examID: string) => Promise<CBTParticipant[]> }) {
  const [confirmItem, setConfirmItem] = useState<{ item: CBTPublishSetting; publish: boolean } | null>(null);
  const [busy, setBusy] = useState(false);
  const [selectedExam, setSelectedExam] = useState<CBTPublishSetting | null>(null);
  const [participants, setParticipants] = useState<CBTParticipant[]>([]);
  const [participantsLoading, setParticipantsLoading] = useState(false);
  const [participantsError, setParticipantsError] = useState("");

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
          : <div className="cbt-settings-table-wrap"><table className="transactions-table cbt-settings-table"><thead><tr><th>Ujian</th><th>Pembuat</th><th>Peserta</th><th>Pembahasan</th><th></th></tr></thead><tbody>
            {items.map((item) => <tr key={item.exam_id} className={selectedExam?.exam_id === item.exam_id ? "active-row" : ""}>
              <td><button type="button" className="cbt-settings-exam-link" onClick={() => void openParticipants(item)}>{item.exam_title}</button><small>{item.package_title} · {item.package_kode} · {item.jenjang} · {item.total_questions} soal · {item.duration_minutes} menit · Syarat lulus {item.passing_score.toFixed(0)}</small></td>
              <td>{item.publisher_email || "Platform"}</td>
              <td>{item.participated} siswa</td>
              <td><span className={`status-pill ${item.publish_pembahasan ? "paid" : "pending"}`}>{item.publish_pembahasan ? "Dipublish" : "Tidak dipublish"}</span></td>
              <td><div className="user-row-actions"><button type="button" className="table-action" onClick={() => void openParticipants(item)}>{selectedExam?.exam_id === item.exam_id ? "Tutup peserta" : "Lihat peserta"}</button>{(item.participated > 0) && <button type="button" className="button small-btn" disabled={saving} onClick={() => setConfirmItem({ item, publish: !item.publish_pembahasan })}>{item.publish_pembahasan ? "Tarik pembahasan" : "Publish pembahasan"}</button>}</div></td>
            </tr>)}
          </tbody></table></div>}
      </div>
      {selectedExam && <div className="card cbt-participants-card">
        <div className="section-heading"><div><p className="eyebrow">Peserta ujian</p><h3>{selectedExam.exam_title}</h3><p className="muted">Siswa yang sedang mengerjakan atau sudah menyelesaikan ujian CBT ini.</p></div><button type="button" className="button secondary small-btn" onClick={() => { setSelectedExam(null); setParticipants([]); }}>Tutup</button></div>
        {participantsError && <p className="error">{participantsError}</p>}
        {participantsLoading ? <p className="empty-state">Memuat peserta...</p> : participants.length === 0 ? <p className="empty-state">Belum ada siswa yang mengikuti ujian ini.</p> : <div className="transactions-table-wrap"><table className="transactions-table"><thead><tr><th>Nama</th><th>Email</th><th>Jenjang</th><th>Soal</th><th>Status</th><th>Waktu</th></tr></thead><tbody>{participants.map((participant) => {
          const ongoing = participant.status === "ongoing";
          const startedStr = participant.started_at ? new Date(participant.started_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          const finishedStr = participant.finished_at ? new Date(participant.finished_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—";
          return <tr key={participant.user_exam_id} className={ongoing ? "active-row" : ""}><td>{participant.name || "—"}</td><td>{participant.email}</td><td>{participant.school_level || "—"}</td><td>{ongoing ? `Soal ${participant.current_question}/${participant.total_questions}` : `${participant.total_questions} soal`}</td><td>{ongoing ? <span className="status-pill ongoing">Sedang mengerjakan</span> : <span>{<span className="status-pill paid">Selesai</span>} <small className="cbt-hint">· {participant.passed ? "Lulus" : "Belum lulus"}</small></span>}</td><td>{ongoing ? <span className="cbt-hint">Mulai {startedStr}</span> : finishedStr}</td></tr>;
        })}</tbody></table></div>}
      </div>}
      {confirmItem && <ConfirmModal title={confirmItem.publish ? "Publish pembahasan?" : "Tarik pembahasan?"} message={confirmItem.publish ? `Siswa yang sudah mengerjakan "${confirmItem.item.exam_title}" akan dapat melihat analitik dan pembahasan hasilnya.` : `Siswa yang sudah mengerjakan "${confirmItem.item.exam_title}" tidak akan bisa melihat analitik hasilnya lagi.`} busy={busy} confirmLabel={confirmItem.publish ? "Ya, publish" : "Ya, tarik"} onClose={() => !busy && setConfirmItem(null)} onConfirm={() => void runToggle()} />}
    </>
  );
}