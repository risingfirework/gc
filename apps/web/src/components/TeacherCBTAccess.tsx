"use client";

import { ChangeEvent, FormEvent, useEffect, useState } from "react";
import { api, CBTAccessStatus, TeacherCBTPayment } from "@/services/api";
import DataImage from "./DataImage";

const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
const dateLabel = (value?: string) => (value ? new Date(value).toLocaleDateString("id-ID", { day: "2-digit", month: "long", year: "numeric" }) : "—");
const paymentStatusLabel: Record<string, string> = { pending: "Menunggu verifikasi", approved: "Disetujui", rejected: "Ditolak", cancelled: "Dibatalkan" };

export function TeacherCBTAccessBanner({ access, onOpen }: { access: CBTAccessStatus; onOpen: () => void }) {
  if (access.settings.access_mode === "free") return null;
  const trialRemaining = Math.max(0, access.settings.free_quota - access.entitlement.free_quota_used);
  const tone = access.can_create_cbt ? (access.reason === "trial_available" ? "trial" : "ok") : access.reason === "pending_review" ? "pending" : "locked";
  return <section className={`cbt-access-banner ${tone}`}>
    <div className="cbt-access-banner-icon" aria-hidden="true">{access.can_create_cbt ? "✓" : access.reason === "pending_review" ? "…" : "!"}</div>
    <div className="cbt-access-banner-body">
      <strong>{access.can_create_cbt ? (access.reason === "trial_available" ? `Jatah gratis CBT tersedia (${trialRemaining} paket)` : "Akses CBT aktif") : "Akses CBT perlu diaktifkan"}</strong>
      <small>{access.message}</small>
      {access.can_create_cbt && access.entitlement.active_until && <small>Berlaku sampai {dateLabel(access.entitlement.active_until)}.</small>}
    </div>
    {!access.can_create_cbt && <button className="button" type="button" onClick={onOpen}>{access.reason === "pending_review" ? "Lihat status" : "Aktifkan akses"}</button>}
    {access.can_create_cbt && access.reason === "trial_available" && <span className="cbt-access-pill">Paket ke-{access.entitlement.free_quota_used + 1} dari {access.settings.free_quota} gratis</span>}
  </section>;
}

export function TeacherCBTAccessModal({ access, open, onClose, onSubmitted }: { access: CBTAccessStatus; open: boolean; onClose: () => void; onSubmitted: () => void }) {
  const [proof, setProof] = useState("");
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [history, setHistory] = useState<TeacherCBTPayment[]>([]);

  useEffect(() => {
    if (!open) return;
    setError(""); setProof("");
    api.teacherCBTPayments().then(setHistory).catch(() => { /* riwayat bersifat informatif */ });
  }, [open]);

  if (!open) return null;

  function pickProof(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError("");
    if (!file.type.startsWith("image/")) { setError("Bukti pembayaran harus berupa gambar."); return; }
    if (file.size > 4 * 1024 * 1024) { setError("Ukuran gambar maksimal 4 MB."); return; }
    const reader = new FileReader();
    reader.onload = () => setProof(String(reader.result ?? ""));
    reader.onerror = () => setError("Gambar gagal dibaca, coba lagi.");
    reader.readAsDataURL(file);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proof || saving) return;
    setSaving(true); setError("");
    try {
      await api.submitTeacherCBTPayment({ proof_image: proof, note: note.trim() });
      const items = await api.teacherCBTPayments();
      setHistory(items);
      setProof(""); setNote("");
      onSubmitted();
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Pembayaran gagal dikirim."); }
    finally { setSaving(false); }
  }

  const pending = history.find((item) => item.status === "pending") ?? access.pending_payment;
  const requiresPayment = access.settings.access_mode !== "free";

  return <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) onClose(); }}>
    <div className="card checkout-modal cbt-access-modal" onMouseDown={(event) => event.stopPropagation()}>
      <button type="button" className="modal-close" disabled={saving} onClick={onClose}>×</button>
      <p className="eyebrow">Akses fitur CBT</p>
      <h2>{pending ? "Menunggu verifikasi admin" : access.can_create_cbt ? "Akses CBT Anda" : "Aktifkan akses CBT"}</h2>
      <p className="muted">{access.message}</p>

      {requiresPayment && <div className="cbt-tariff-card">
        <div><small>Tarif akses</small><strong>{money.format(access.settings.price_amount)}</strong></div>
        <div><small>Masa berlaku</small><strong>{access.settings.validity_days > 0 ? `${access.settings.validity_days} hari` : "Tanpa batas"}</strong></div>
        {access.settings.free_quota > 0 && <div><small>Jatah gratis</small><strong>{Math.max(0, access.settings.free_quota - access.entitlement.free_quota_used)} paket tersisa</strong></div>}
      </div>}

      {access.settings.payment_instructions && <div className="cbt-instructions"><strong>Instruksi pembayaran</strong><p>{access.settings.payment_instructions}</p></div>}
      {access.settings.contact_info && <p className="muted">Konfirmasi ke admin: {access.settings.contact_info}</p>}

      {pending ? <div className="cbt-pending-note"><span aria-hidden="true">⏳</span><p><strong>Pembayaran sedang diperiksa</strong><small>Admin akan memverifikasi bukti transfer Anda. Status akan diperbarui otomatis.</small></p></div>
        : requiresPayment && !access.can_create_cbt ? <form className="cbt-payment-form" onSubmit={submit}>
          <label className="gate-file-browser"><span>Upload bukti transfer</span><input type="file" accept="image/*" onChange={pickProof} required /><small>Lampirkan screenshot atau foto bukti pembayaran (maks. 4 MB).</small></label>
          {proof && <div className="gate-proof-preview"><DataImage src={proof} alt="Pratinjau bukti pembayaran" width={480} height={360} /></div>}
          <label><span>Catatan untuk admin <em>Opsional</em></span><textarea rows={2} value={note} maxLength={500} onChange={(event) => setNote(event.target.value)} placeholder="Contoh: transfer atas nama sekolah." /></label>
          {error && <p className="error">{error}</p>}
          <button className="button full" disabled={saving || !proof}>{saving ? "Mengirim..." : "Kirim bukti pembayaran"}</button>
        </form>
        : <div className="cbt-access-active"><span aria-hidden="true">✓</span><p><strong>Akses CBT aktif</strong><small>Anda dapat membuat paket CBT baru sesuai kebijakan yang berlaku.</small></p></div>}

      {history.length > 0 && <div className="cbt-payment-history">
        <strong>Riwayat pembayaran</strong>
        {history.map((item) => <div className="cbt-payment-row" key={item.id}>
          <div><strong>{money.format(item.amount)}</strong><small>{dateLabel(item.created_at)}</small></div>
          <span className={`request-status ${item.status}`}>{paymentStatusLabel[item.status] ?? item.status}</span>
          {item.rejection_reason && <small className="cbt-payment-reason">{item.rejection_reason}</small>}
        </div>)}
      </div>}
    </div>
  </div>;
}
