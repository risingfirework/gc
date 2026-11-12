"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { api, CBTAccessSettings, TeacherCBTAccessRow, TeacherCBTPayment } from "@/services/api";
import DataImage from "./DataImage";

const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
const dateTime = (value?: string) => (value ? new Date(value).toLocaleString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }) : "—");
const entitlementLabel: Record<string, string> = {
  trial_available: "Jatah gratis tersedia",
  trial_used: "Jatah gratis terpakai",
  active: "Aktif",
  expired: "Kedaluwarsa",
  pending: "Menunggu verifikasi",
  blocked: "Diblokir",
};
const paymentStatusLabel: Record<string, string> = { pending: "Menunggu verifikasi", approved: "Disetujui", rejected: "Ditolak", cancelled: "Dibatalkan" };
type Tab = "kebijakan" | "pembayaran" | "akses";

export default function CBTAccessAdminPanel() {
  const [tab, setTab] = useState<Tab>("kebijakan");
  const [settings, setSettings] = useState<CBTAccessSettings | null>(null);
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");
  const [saving, setSaving] = useState("");

  const [payments, setPayments] = useState<TeacherCBTPayment[]>([]);
  const [paymentPage, setPaymentPage] = useState(1);
  const [paymentTotal, setPaymentTotal] = useState(0);
  const [paymentStatus, setPaymentStatus] = useState<"pending" | "approved" | "rejected" | "all">("pending");
  const [paymentFilter, setPaymentFilter] = useState<TeacherCBTPayment | null>(null);

  const [teachers, setTeachers] = useState<TeacherCBTAccessRow[]>([]);
  const [teacherPage, setTeacherPage] = useState(1);
  const [teacherTotal, setTeacherTotal] = useState(0);
  const [teacherStatus, setTeacherStatus] = useState("all");
  const [teacherQuery, setTeacherQuery] = useState("");
  const [grantTarget, setGrantTarget] = useState<TeacherCBTAccessRow | null>(null);

  const loadSettings = useCallback(() => {
    api.adminCBTAccessSettings().then(setSettings).catch((reason) => setError(reason instanceof Error ? reason.message : "Pengaturan akses CBT gagal dimuat."));
  }, []);
  useEffect(() => { loadSettings(); }, [loadSettings]);
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(""), 3500); return () => window.clearTimeout(timer); }, [toast]);

  useEffect(() => {
    if (tab !== "pembayaran") return;
    api.adminCBTPayments(paymentPage, 25, paymentStatus === "all" ? undefined : paymentStatus)
      .then((result) => { setPayments(result.items); setPaymentTotal(result.total); })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "Pembayaran akses CBT gagal dimuat."));
  }, [tab, paymentPage, paymentStatus]);

  useEffect(() => {
    if (tab !== "akses") return;
    api.adminTeacherCBTAccess(teacherPage, 25, { status: teacherStatus === "all" ? undefined : teacherStatus, q: teacherQuery.trim() || undefined })
      .then((result) => { setTeachers(result.items); setTeacherTotal(result.total); })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "Daftar akses guru gagal dimuat."));
  }, [tab, teacherPage, teacherStatus, teacherQuery]);

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const payload: Omit<CBTAccessSettings, "updated_at"> = {
      access_mode: String(form.get("access_mode")) as CBTAccessSettings["access_mode"],
      price_amount: Number(form.get("price_amount")),
      currency: "IDR",
      validity_days: Number(form.get("validity_days")),
      free_quota: Number(form.get("free_quota")),
      free_quota_reset: String(form.get("free_quota_reset")) as CBTAccessSettings["free_quota_reset"],
      payment_mode: String(form.get("payment_mode")) as CBTAccessSettings["payment_mode"],
      payment_instructions: String(form.get("payment_instructions")),
      contact_info: String(form.get("contact_info")),
      lock_scope: String(form.get("lock_scope")) as CBTAccessSettings["lock_scope"],
      allow_manage_existing: form.get("allow_manage_existing") === "on",
      grace_days: Number(form.get("grace_days")),
      require_verified_teacher: form.get("require_verified_teacher") === "on",
    };
    setSaving("settings"); setError("");
    try {
      const updated = await api.updateAdminCBTAccessSettings(payload);
      setSettings(updated);
      setToast("Kebijakan akses CBT berhasil disimpan.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Kebijakan gagal disimpan."); }
    finally { setSaving(""); }
  }

  async function approve(payment: TeacherCBTPayment) {
    setSaving(payment.id); setError("");
    try {
      await api.adminApproveCBTPayment(payment.id, { note: "" });
      setPayments((current) => current.map((item) => item.id === payment.id ? { ...item, status: "approved" } : item));
      setToast(`Pembayaran ${payment.user_email ?? ""} disetujui. Akses CBT guru diaktifkan.`);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Pembayaran gagal disetujui."); }
    finally { setSaving(""); }
  }

  async function reject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!paymentFilter) return;
    const reason = String(new FormData(event.currentTarget).get("reason") ?? "").trim();
    if (!reason) return;
    setSaving(paymentFilter.id); setError("");
    try {
      await api.adminRejectCBTPayment(paymentFilter.id, { reason });
      setPayments((current) => current.map((item) => item.id === paymentFilter.id ? { ...item, status: "rejected", rejection_reason: reason } : item));
      setToast("Pembayaran akses CBT ditolak.");
      setPaymentFilter(null);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Pembayaran gagal ditolak."); }
    finally { setSaving(""); }
  }

  async function revoke(row: TeacherCBTAccessRow) {
    const reason = window.prompt(`Alasan mencabut akses CBT ${row.name || row.email}?`, "Pelanggaran ketentuan");
    if (!reason?.trim()) return;
    setSaving(row.user_id); setError("");
    try {
      await api.adminRevokeCBTAccess(row.user_id, reason.trim());
      setTeachers((current) => current.map((item) => item.user_id === row.user_id ? { ...item, can_create_cbt: false, entitlement: { ...item.entitlement, status: "blocked" } } : item));
      setToast("Akses CBT guru dicabut.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Akses gagal dicabut."); }
    finally { setSaving(""); }
  }

  return <section className="admin-view cbt-access-admin">
    <div className="admin-view-header">
      <div>
        <p className="eyebrow">Akses fitur CBT guru</p>
        <h2>Kebijakan &amp; tarif CBT</h2>
        <p className="muted">Atur mode akses, tarif, dan verifikasi pembayaran fitur CBT untuk guru. Perubahan kebijakan tidak mengubah data historis atau masa aktif yang sudah berjalan.</p>
      </div>
    </div>
    {error && <p className="error">{error}</p>}
    <div className="content-subtabs cbt-access-tabs">
      <button type="button" className={tab === "kebijakan" ? "active" : ""} onClick={() => setTab("kebijakan")}>Kebijakan &amp; Tarif</button>
      <button type="button" className={tab === "pembayaran" ? "active" : ""} onClick={() => setTab("pembayaran")}>Pembayaran</button>
      <button type="button" className={tab === "akses" ? "active" : ""} onClick={() => setTab("akses")}>Akses Guru</button>
    </div>

    {tab === "kebijakan" && settings && <form className="card cbt-access-form" onSubmit={saveSettings}>
      <div className="cbt-access-grid">
        <label><span>Mode akses</span><select name="access_mode" defaultValue={settings.access_mode}><option value="free">Gratis untuk semua guru</option><option value="trial_then_paid">Paket gratis sekali, lalu berbayar</option><option value="paid">Berbayar</option></select><small>Gratis = tanpa gate. Paket gratis sekali = guru mendapat jatah gratis sebelum wajib berlangganan.</small></label>
        <label><span>Tarif per guru</span><div className="finance-number"><i>Rp</i><input name="price_amount" type="number" min="0" step="1000" defaultValue={settings.price_amount} required /></div><small>Nominal yang harus dibayar guru untuk mengaktifkan akses CBT.</small></label>
        <label><span>Masa berlaku akses (hari)</span><input name="validity_days" type="number" min="0" max="3650" defaultValue={settings.validity_days} required /><small>Isi 0 untuk akses tanpa batas waktu.</small></label>
        <label><span>Jatah paket gratis</span><input name="free_quota" type="number" min="0" max="100" defaultValue={settings.free_quota} required /><small>Jumlah paket CBT yang boleh dibuat sebelum akses berbayar diperlukan.</small></label>
        <label><span>Reset jatah gratis</span><select name="free_quota_reset" defaultValue={settings.free_quota_reset}><option value="never">Tidak pernah</option><option value="yearly" disabled>Tahunan (belum tersedia)</option></select><small>Reset otomatis belum tersedia pada versi ini.</small></label>
        <label><span>Metode pembayaran</span><select name="payment_mode" defaultValue={settings.payment_mode}><option value="manual">Verifikasi manual (transfer)</option><option value="gateway" disabled>Payment gateway (belum tersedia)</option></select><small>Versi ini memakai verifikasi manual oleh admin.</small></label>
        <label><span>Cakupan penguncian</span><select name="lock_scope" defaultValue={settings.lock_scope}><option value="create_new_cbt">Hanya pembuatan CBT baru</option><option value="all_cbt_features" disabled>Seluruh fitur CBT (belum tersedia)</option></select><small>Guru tetap dapat mengelola CBT yang sudah ada saat akses habis.</small></label>
        <label><span>Masa tenggang (hari)</span><input name="grace_days" type="number" min="0" max="90" defaultValue={settings.grace_days} required /></label>
      </div>
      <div className="cbt-access-checks">
        <label className="check-row"><input type="checkbox" name="allow_manage_existing" defaultChecked={settings.allow_manage_existing} /><span><strong>Izinkan kelola CBT lama</strong><small>Guru tetap dapat mengelola paket, ujian, soal, dan peserta CBT yang sudah ada saat akses habis.</small></span></label>
        <label className="check-row"><input type="checkbox" name="require_verified_teacher" defaultChecked={settings.require_verified_teacher} /><span><strong>Wajib guru terverifikasi</strong><small>Hanya guru dengan verifikasi disetujui yang dapat membuat paket CBT.</small></span></label>
      </div>
      <label><span>Instruksi pembayaran</span><textarea name="payment_instructions" rows={4} defaultValue={settings.payment_instructions} placeholder="Contoh: Transfer ke BCA 1234567890 a.n. Yayasan TKA Juara. Cantumkan email guru pada berita transfer." /></label>
      <label><span>Kontak admin</span><input name="contact_info" defaultValue={settings.contact_info} placeholder="Contoh: WhatsApp 0812-3456-7890 atau admin@siap-siap.my.id" /></label>
      <div className="cbt-access-form-footer">
        <p className="muted">Terakhir diperbarui: {dateTime(settings.updated_at)}</p>
        <button className="button" disabled={saving === "settings"}>{saving === "settings" ? "Menyimpan..." : "Simpan kebijakan"}</button>
      </div>
    </form>}

    {tab === "pembayaran" && <>
      <div className="content-subtabs cbt-access-tabs">
        {(["pending", "approved", "rejected", "all"] as const).map((status) => <button key={status} type="button" className={paymentStatus === status ? "active" : ""} onClick={() => { setPaymentPage(1); setPaymentStatus(status); }}>{status === "all" ? "Semua" : paymentStatusLabel[status]}</button>)}
      </div>
      <div className="card admin-table-card">
        <div className="ranking-table-wrap"><table className="ranking-table"><thead><tr><th>Guru</th><th>Nominal</th><th>Bukti</th><th>Status</th><th>Diajukan</th><th>Aksi</th></tr></thead><tbody>
          {payments.length ? payments.map((item) => <tr key={item.id}>
            <td><strong>{item.user_name || item.user_email}</strong><small>{item.user_email}</small>{item.note && <small>{item.note}</small>}</td>
            <td><strong>{money.format(item.amount)}</strong></td>
            <td>{item.proof_image ? <div className="cbt-proof-thumb"><DataImage src={item.proof_image} alt="Bukti transfer" width={128} height={96} /></div> : "—"}</td>
            <td><span className={`request-status ${item.status}`}>{paymentStatusLabel[item.status] ?? item.status}</span>{item.rejection_reason && <small>{item.rejection_reason}</small>}</td>
            <td>{dateTime(item.created_at)}</td>
            <td>{item.status === "pending" ? <div className="cbt-row-actions"><button className="table-action" disabled={saving === item.id} onClick={() => void approve(item)}>Setujui</button><button className="danger-action" disabled={saving === item.id} onClick={() => setPaymentFilter(item)}>Tolak</button></div> : item.period_end ? <small>Aktif s.d. {dateTime(item.period_end)}</small> : "—"}</td>
          </tr>) : <tr><td colSpan={6} className="empty-state">Belum ada pembayaran akses CBT.</td></tr>}
        </tbody></table></div>
        <Pager page={paymentPage} total={paymentTotal} onChange={setPaymentPage} />
      </div>
    </>}

    {tab === "akses" && <>
      <div className="cbt-access-toolbar">
        <input type="search" placeholder="Cari nama atau email guru..." value={teacherQuery} onChange={(event) => { setTeacherQuery(event.target.value); setTeacherPage(1); }} />
        <select value={teacherStatus} onChange={(event) => { setTeacherStatus(event.target.value); setTeacherPage(1); }}>
          <option value="all">Semua status</option>
          <option value="trial_available">Jatah gratis tersedia</option>
          <option value="trial_used">Jatah gratis terpakai</option>
          <option value="active">Aktif</option>
          <option value="expired">Kedaluwarsa</option>
          <option value="blocked">Diblokir</option>
        </select>
      </div>
      <div className="card admin-table-card">
        <div className="ranking-table-wrap"><table className="ranking-table"><thead><tr><th>Guru</th><th>Verifikasi</th><th>Status akses</th><th>Masa aktif</th><th>Jatah gratis</th><th>Aksi</th></tr></thead><tbody>
          {teachers.length ? teachers.map((item) => <tr key={item.user_id}>
            <td><strong>{item.name || item.email}</strong><small>{item.email}</small></td>
            <td>{item.verification_status === "approved" ? <span className="request-status approved">Disetujui</span> : <span className="request-status pending">{item.verification_status || "Belum"}</span>}</td>
            <td><span className={`request-status ${item.entitlement.status}`}>{entitlementLabel[item.entitlement.status] ?? item.entitlement.status}</span><small>{item.message}</small>{item.has_pending_payment && <small>Ada pembayaran menunggu verifikasi.</small>}</td>
            <td>{item.entitlement.active_until ? dateTime(item.entitlement.active_until) : item.entitlement.status === "active" ? "Tanpa batas" : "—"}</td>
            <td>{item.entitlement.free_quota_used} terpakai</td>
            <td><div className="cbt-row-actions"><button className="table-action" disabled={saving === item.user_id} onClick={() => setGrantTarget(item)}>{item.can_create_cbt ? "Perpanjang" : "Beri akses"}</button>{item.entitlement.status !== "blocked" && <button className="danger-action" disabled={saving === item.user_id} onClick={() => void revoke(item)}>Cabut</button>}</div></td>
          </tr>) : <tr><td colSpan={6} className="empty-state">Belum ada data guru.</td></tr>}
        </tbody></table></div>
        <Pager page={teacherPage} total={teacherTotal} onChange={setTeacherPage} />
      </div>
    </>}

    {paymentFilter && <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget) setPaymentFilter(null); }}>
      <form className="card checkout-modal" onSubmit={reject} onMouseDown={(event) => event.stopPropagation()}>
        <button type="button" className="modal-close" onClick={() => setPaymentFilter(null)}>×</button>
        <p className="eyebrow">Tolak pembayaran</p>
        <h2>{paymentFilter.user_name || paymentFilter.user_email}</h2>
        <p className="muted">Sampaikan alasan agar guru dapat memperbaiki bukti pembayaran.</p>
        <label><span>Alasan penolakan</span><textarea name="reason" rows={3} required autoFocus placeholder="Contoh: nominal transfer tidak sesuai." /></label>
        <button className="button full" disabled={saving === paymentFilter.id}>{saving === paymentFilter.id ? "Menyimpan..." : "Tolak pembayaran"}</button>
      </form>
    </div>}

    {grantTarget && <GrantModal row={grantTarget} defaultValidity={settings?.validity_days ?? 365} onClose={() => setGrantTarget(null)} onDone={() => { setGrantTarget(null); setToast("Akses CBT guru berhasil diaktifkan."); api.adminTeacherCBTAccess(teacherPage, 25, { status: teacherStatus === "all" ? undefined : teacherStatus, q: teacherQuery.trim() || undefined }).then((result) => { setTeachers(result.items); setTeacherTotal(result.total); }).catch(() => { /* data akan dimuat ulang saat tab dibuka */ }); }} />}

    {toast && <div className="success-toast" role="status"><span>✓</span><p>{toast}</p><button type="button" aria-label="Tutup notifikasi" onClick={() => setToast("")}>×</button></div>}
  </section>;
}

function Pager({ page, total, onChange }: { page: number; total: number; onChange: (page: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / 25));
  if (pages <= 1) return null;
  return <div className="cbt-access-pager"><button type="button" className="button secondary" disabled={page <= 1} onClick={() => onChange(page - 1)}>← Sebelumnya</button><span>Halaman {page} dari {pages}</span><button type="button" className="button secondary" disabled={page >= pages} onClick={() => onChange(page + 1)}>Berikutnya →</button></div>;
}

function GrantModal({ row, defaultValidity, onClose, onDone }: { row: TeacherCBTAccessRow; defaultValidity: number; onClose: () => void; onDone: () => void }) {
  const [validity, setValidity] = useState(String(defaultValidity));
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true); setError("");
    try { await api.adminGrantCBTAccess(row.user_id, Number(validity), reason.trim()); onDone(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Akses gagal diaktifkan."); }
    finally { setSaving(false); }
  }
  return <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) onClose(); }}>
    <form className="card checkout-modal" onSubmit={submit} onMouseDown={(event) => event.stopPropagation()}>
      <button type="button" className="modal-close" disabled={saving} onClick={onClose}>×</button>
      <p className="eyebrow">Aktifkan akses CBT</p>
      <h2>{row.name || row.email}</h2>
      <p className="muted">Akses akan langsung aktif tanpa pembayaran. Tindakan ini tercatat pada audit log.</p>
      <label><span>Masa berlaku (hari)</span><input type="number" min="0" max="3650" value={validity} onChange={(event) => setValidity(event.target.value)} required /><small>Isi 0 untuk akses tanpa batas waktu.</small></label>
      <label><span>Catatan / alasan</span><textarea rows={3} value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Contoh: kompensasi guru berprestasi." /></label>
      {error && <p className="error">{error}</p>}
      <button className="button full" disabled={saving}>{saving ? "Menyimpan..." : "Aktifkan akses"}</button>
    </form>
  </div>;
}
