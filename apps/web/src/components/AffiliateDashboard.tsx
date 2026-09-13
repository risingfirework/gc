"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AffiliateDashboardData, api, SiteSettings, User } from "@/services/api";
import ProfileEditor from "./ProfileEditor";
import LogoutButton from "./LogoutButton";
import NotificationBell from "./NotificationBell";
import BrandLogo from "./BrandLogo";
import PaginationControls, { usePageSlice } from "./PaginationControls";

type AffiliateView = "ringkasan" | "rujukan" | "pencairan" | "profil";
const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });

export default function AffiliateDashboard({ user, onLogout, loggingOut, onUserUpdate }: { user: User; onLogout: () => void; loggingOut: boolean; onUserUpdate?: (u: User) => void }) {
  const [view, setView] = useState<AffiliateView>("ringkasan");
  const [data, setData] = useState<AffiliateDashboardData | null>(null);
  const [siteSettings, setSiteSettings] = useState<SiteSettings | null>(null);
  const [error, setError] = useState("");
  const [commissionMessage, setCommissionMessage] = useState("");
  const [saving, setSaving] = useState("");
  const [payoutModal, setPayoutModal] = useState(false);
  const referralsPage = usePageSlice(data?.referrals ?? [], 25);
  const commissionsPage = usePageSlice(data?.commissions ?? [], 25);

  const reload = useCallback(async () => {
    setData(await api.affiliateDashboard());
  }, []);
  useEffect(() => {
    Promise.all([api.affiliateDashboard(), api.siteSettings()])
      .then(([dashboard, settings]) => { setData(dashboard); setSiteSettings(settings); })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "Data affiliate gagal dimuat."));
  }, []);

  async function savePayoutAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving("payout-account"); setError(""); setCommissionMessage("");
    try {
      await api.updateAffiliatePayoutAccount({ method: String(form.get("method")) as "bank_transfer" | "e_wallet", provider: String(form.get("provider")), account_number: String(form.get("account_number")), account_holder_name: String(form.get("account_holder_name")), phone: String(form.get("phone")) });
      await reload();
      setCommissionMessage("Data pencairan berhasil disimpan dan siap digunakan oleh admin finance.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Data pencairan gagal disimpan."); }
    finally { setSaving(""); }
  }

  async function requestPayout(amount: number) {
    setSaving("payout-request"); setError(""); setCommissionMessage("");
    try { const item = await api.createAffiliatePayoutRequest(amount); await reload(); setCommissionMessage(`Pengajuan ${money.format(item.amount)} berhasil dikirim dan masuk antrean pemeriksaan admin.`); setPayoutModal(false); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengajuan pencairan gagal dikirim."); }
    finally { setSaving(""); }
  }
  async function cancelPayoutRequest(id: string) {
    if (!window.confirm("Batalkan pengajuan pencairan ini? Saldo akan kembali tersedia.")) return;
    setSaving("payout-cancel"); setError("");
    try { await api.cancelAffiliatePayoutRequest(id); await reload(); setCommissionMessage("Pengajuan berhasil dibatalkan."); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengajuan tidak dapat dibatalkan."); }
    finally { setSaving(""); }
  }

  const menus: Array<[AffiliateView, string]> = [["ringkasan", "Ringkasan"], ["rujukan", "Daftar Rujukan"], ["pencairan", "Pencairan Dana"], ["profil", "Profil"]];
  const referrals = data?.referrals ?? [];
  const converted = referrals.filter((item) => item.first_purchase_at);

  return <main><div className="admin-layout">
    <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url} /><span>{siteSettings?.platform_name ?? ""}<small>Panel Affiliate</small></span></Link><nav>{menus.map(([key, label]) => <button key={key} className={view === key ? "active" : ""} onClick={() => setView(key)}>{label}</button>)}</nav><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout} /></aside>
    <section className="admin-main">
      <header className="admin-topbar"><div><p className="eyebrow">Panel affiliate</p><h1>{menus.find(([key]) => key === view)?.[1]}</h1></div><span className="topbar-actions"><NotificationBell /><button className="admin-identity" onClick={() => setView("profil")}><span>{(user.name?.trim() || user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim() || "Nama belum dilengkapi"}</strong><small>{user.email} · Affiliate</small></div></button></span></header>
      {error && <p className="error">{error}</p>}{!data && !error && <div className="card">Memuat data affiliate...</div>}

      {data && view === "ringkasan" && <div className="admin-view">
        <section className="card affiliate-hero">
          <div><p className="eyebrow">Kode rujukanmu</p><h2>{data.referral_code || "Belum tersedia"}</h2><p className="muted">Bagikan link berikut kepada siswa untuk mulai mengumpulkan komisi rujukan.</p>{data.referral_link && <div className="affiliate-link-row"><input readOnly value={data.referral_link} onFocus={(event) => event.currentTarget.select()} /><button className="button secondary" onClick={() => { void navigator.clipboard.writeText(data.referral_link).then(() => setCommissionMessage("Link rujukan disalin ke clipboard.")); }}>Salin link</button></div>}</div>
          <div className="affiliate-code-box"><span>Kode</span><strong>{data.referral_code || "—"}</strong></div>
        </section>
        {commissionMessage && <p className="finance-success">{commissionMessage}</p>}
        <div className="admin-stats">
          <Stat label="Total rujukan" value={referrals.length} />
          <Stat label="Sudah membeli" value={converted.length} />
          <Stat label="Komisi rujukan" value={data.commission_summary.sales_bonus} money />
          <Stat label="Dapat dicairkan" value={data.commission_summary.available} money />
          <Stat label="Sudah dibayar" value={data.commission_summary.paid} money />
        </div>
        <div className="fees-recap"><FeeCard label="Masih ditahan" value={data.commission_summary.held} /><FeeCard label="Komisi rujukan" value={data.commission_summary.sales_bonus} /></div>
        <section className="admin-view-header"><div><p className="eyebrow">Konversi terbaru</p><h2>Rujukan yang sudah membeli</h2></div></section>
        <AdminTable headers={["Email", "Nama", "Belanja pertama", "Komisi"]}>{referralsPage.total ? referralsPage.sliced.map((item) => <tr key={item.user_id}><td>{item.email}</td><td>{item.name || "—"}</td><td>{item.first_purchase_at ? new Date(item.first_purchase_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" }) : "—"}</td><td><strong>{money.format(item.commission_amount ?? 0)}</strong></td></tr>) : <tr><td colSpan={4} className="empty-state">Belum ada rujukan yang membeli paket. Bagikan linkmu untuk memulai.</td></tr>}</AdminTable><PaginationControls page={referralsPage.page} perPage={referralsPage.perPage} total={referralsPage.total} onPrev={referralsPage.prev} onNext={referralsPage.next}/>
      </div>}

      {data && view === "rujukan" && <section className="admin-view teacher-commission-view">
        <div className="admin-view-header"><div><p className="eyebrow">Siswa yang kamu perkenalkan</p><h2>Daftar rujukan</h2><p className="muted">Atribusi first-touch: komisi hanya untuk transaksi pertama setiap siswa yang mendaftar lewat kode.</p></div></div>
        <AdminTable headers={["Didaftarkan", "Email", "Nama", "Pembelian terkonfirmasi", "Komisi"]}>{referrals.length ? referrals.map((item) => <tr key={item.user_id}><td>{new Date(item.referred_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</td><td>{item.email}</td><td>{item.name || "—"}</td><td>{item.first_purchase_at ? new Date(item.first_purchase_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" }) : "Belum membeli"}</td><td>{item.commission_id ? <strong>{money.format(item.commission_amount ?? 0)}</strong> : "—"}</td></tr>) : <tr><td colSpan={5} className="empty-state">Belum ada pengguna mendaftar lewat kodemu.</td></tr>}</AdminTable>
        <div className="admin-view-header"><div><h2>Rincian komisi</h2><p className="muted">Bonus rujukan {data.finance_settings.affiliate_rate_percent}% dari jatah komisi platform.</p></div><button className="button" onClick={() => setView("pencairan")}>Ajukan pencairan</button></div>
        <AdminTable headers={["Tanggal", "Paket", "Sumber", "Nominal", "Status"]}>{commissionsPage.total ? commissionsPage.sliced.map((item) => <tr key={item.id}><td>{new Date(item.created_at).toLocaleDateString("id-ID")}</td><td>{item.package_title}{item.invoice_number && <small>{item.invoice_number}</small>}</td><td>{item.kind === "referral_bonus" ? `Bonus rujukan ${item.rate_percent}%` : "Koreksi refund"}</td><td><strong>{money.format(item.amount)}</strong></td><td><PayoutStatus status={item.status} /></td></tr>) : <tr><td colSpan={5} className="empty-state">Belum ada komisi tercatat.</td></tr>}</AdminTable><PaginationControls page={commissionsPage.page} perPage={commissionsPage.perPage} total={commissionsPage.total} onPrev={commissionsPage.prev} onNext={commissionsPage.next}/>
      </section>}

      {data && view === "pencairan" && <section className="admin-view teacher-commission-view">
        <div className="admin-view-header"><div><p className="eyebrow">Pencairan dana</p><h2>Panel pencairan</h2><p className="muted">Komisi rujukan dicairkan lewat mekanisme yang sama dengan honorarium guru.</p></div></div>
        {commissionMessage && <p className="finance-success">{commissionMessage}</p>}
        <div className="card withdrawal-hero"><div><span>Saldo dapat dicairkan</span><strong>{money.format(data.commission_summary.available)}</strong><small>Minimum pencairan {money.format(data.finance_settings.minimum_payout)}</small></div>
          <div className="withdrawal-balance-grid"><div><span>Komisi rujukan</span><strong>{money.format(data.commission_summary.sales_bonus)}</strong></div><div><span>Masih ditahan</span><strong>{money.format(data.commission_summary.held)}</strong></div><div><span>Sudah dibayar</span><strong>{money.format(data.commission_summary.paid)}</strong></div></div>
          <div className="withdrawal-actions"><button className="button" disabled={saving !== "" || !data.payout_account.updated_at || data.commission_summary.available < data.finance_settings.minimum_payout || data.payout_requests.some((item) => item.status === "submitted" || item.status === "approved")} onClick={() => setPayoutModal(true)}>{data.payout_requests.some((item) => item.status === "submitted" || item.status === "approved") ? "Pengajuan sedang diproses" : "Ajukan pencairan"}</button></div></div>
        <form key={data.payout_account.updated_at ?? "new"} className="card payout-account-card" onSubmit={savePayoutAccount}>
          <div className="payout-account-heading"><div className="payout-account-icon" aria-hidden="true">Rp</div><div><p className="eyebrow">Tujuan pencairan</p><h2>Data rekening penerima</h2><p className="muted">Pastikan data sesuai dengan rekening aktif agar proses transfer tidak tertunda.</p></div><span className={`payout-readiness ${data.payout_account.updated_at ? "ready" : "pending"}`}>{data.payout_account.updated_at ? "Data lengkap" : "Belum dilengkapi"}</span></div>
          <div className="payout-form-grid">
            <label><span>Metode pencairan</span><select name="method" defaultValue={data.payout_account.method || "bank_transfer"} required><option value="bank_transfer">Transfer bank</option><option value="e_wallet">Dompet digital (e-wallet)</option></select></label>
            <label><span>Bank atau penyedia e-wallet</span><input name="provider" defaultValue={data.payout_account.provider} list="payout-providers-aff" placeholder="Contoh: BCA atau DANA" maxLength={80} autoComplete="organization" required /><datalist id="payout-providers-aff"><option value="BCA" /><option value="Bank Mandiri" /><option value="BNI" /><option value="BRI" /><option value="DANA" /><option value="GoPay" /><option value="OVO" /><option value="ShopeePay" /></datalist></label>
            <label><span>Nomor rekening atau akun</span><input name="account_number" defaultValue={data.payout_account.account_number} inputMode="numeric" pattern="[0-9]{6,30}" minLength={6} maxLength={30} placeholder="Masukkan nomor tanpa spasi" autoComplete="off" required /><small>Gunakan angka saja, 6–30 digit.</small></label>
            <label><span>Nama pemilik rekening</span><input name="account_holder_name" defaultValue={data.payout_account.account_holder_name || user.name} maxLength={120} placeholder="Sesuai buku tabungan atau akun" autoComplete="name" required /></label>
            <label><span>Nomor WhatsApp aktif <em>Opsional</em></span><input name="phone" defaultValue={data.payout_account.phone || user.phone} inputMode="tel" pattern="(\+62|62|0)[0-9]{8,13}" maxLength={16} placeholder="Contoh: 081234567890" autoComplete="tel" /><small>Digunakan jika admin perlu konfirmasi pencairan.</small></label>
          </div>
          <div className="payout-form-footer"><div className="payout-security"><span aria-hidden="true">✓</span><p><strong>Khusus administrasi pencairan</strong><small>Informasi ini hanya dapat diakses oleh affiliate terkait dan admin finance.</small></p></div><button className="button" disabled={saving === "payout-account"}>{saving === "payout-account" ? "Menyimpan..." : data.payout_account.updated_at ? "Perbarui data pencairan" : "Simpan data pencairan"}</button></div>
        </form>
        <div className="admin-view-header"><div><h2>Status pengajuan</h2><p className="muted">Pantau proses pemeriksaan, persetujuan, hingga bukti transfer.</p></div></div>
        <AdminTable headers={["Tanggal", "Tujuan", "Nominal", "Status", "Bukti / Catatan / Aksi"]}>{data.payout_requests.length ? data.payout_requests.map((item) => <tr key={item.id}><td>{new Date(item.submitted_at).toLocaleString("id-ID")}</td><td>{item.provider}<small>•••• {item.account_number.slice(-4)}</small></td><td><strong>{money.format(item.amount)}</strong></td><td><PayoutStatus status={item.status} /></td><td>{item.status === "submitted" || item.status === "approved" ? <button className="danger-action" disabled={saving !== ""} onClick={() => void cancelPayoutRequest(item.id)}>Batalkan</button> : item.status === "paid" && item.proof_url ? <a className="proof-link" href={item.proof_url} target="_blank" rel="noreferrer">Lihat bukti transfer</a> : item.transfer_reference || item.admin_note || "—"}</td></tr>) : <tr><td colSpan={5} className="empty-state">Belum ada pengajuan pencairan.</td></tr>}</AdminTable>
      </section>}

      {view === "profil" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div><ProfileEditor user={user} onUpdated={onUserUpdate} /></section>}
    </section>
    {payoutModal && data && <PayoutRequestModal available={data.commission_summary.available} minimum={data.finance_settings.minimum_payout} saving={saving === "payout-request"} onClose={() => setPayoutModal(false)} onSubmit={(amount) => void requestPayout(amount)} />}
  </div></main>;
}

function Stat({ label, value, money: isMoney }: { label: string; value: number; money?: boolean }) { return <article className="card admin-stat"><span>{label}</span><strong>{isMoney ? money.format(value) : value}</strong></article>; }
function FeeCard({ label, value }: { label: string; value: number }) { return <article className="card finance-metric"><span>{label}</span><strong>{money.format(value)}</strong></article>; }
function PayoutStatus({ status }: { status: string }) { const labels: Record<string, string> = { pending: "Ditahan", available: "Tersedia", submitted: "Pengajuan", approved: "Ditinjau admin", paid: "Berhasil ditransfer", rejected: "Ditolak", cancelled: "Dibatalkan" }; return <span className={`request-status ${status}`}>{labels[status] ?? status}</span>; }
function AdminTable({ headers, children }: { headers: string[]; children: React.ReactNode }) { return <div className="card admin-table-card"><div className="ranking-table-wrap"><table className="ranking-table"><thead><tr>{headers.map((header) => <th key={header}>{header}</th>)}</tr></thead><tbody>{children}</tbody></table></div></div>; }
function PayoutRequestModal({ available, minimum, saving, onClose, onSubmit }: { available: number; minimum: number; saving: boolean; onClose: () => void; onSubmit: (amount: number) => void }) {
  const [amount, setAmount] = useState("");
  const numeric = Number(amount.replace(/[^\d]/g, "") || 0);
  const valid = numeric > 0 && numeric >= minimum && numeric <= available;
  return <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) onClose(); }}><form className="card checkout-modal payout-request-modal" onSubmit={(event) => { event.preventDefault(); if (valid) onSubmit(numeric); }} onMouseDown={(event) => event.stopPropagation()}><button type="button" className="modal-close" disabled={saving} onClick={onClose}>×</button><p className="eyebrow">Pengajuan pencairan</p><h2>Berapa nominal yang ingin dicairkan?</h2><p className="muted">Nominal ini dikunci dari saldo dapat dicairkan hingga proses selesai atau dibatalkan.</p>
    <div className="checkout-total"><span>Saldo dapat dicairkan</span><strong>{money.format(available)}</strong></div>
    <label><span>Nominal penarikan (Rp)</span><div className="finance-number"><i>Rp</i><input name="amount" inputMode="numeric" placeholder={`Contoh: ${available}`} value={amount} onChange={(event) => setAmount(event.target.value)} autoFocus /></div></label>
    {numeric > 0 && !valid && <p className="form-helper">{numeric < minimum ? `Nominal minimal ${money.format(minimum)}.` : `Nominal tidak boleh melebihi saldo dapat dicairkan ${money.format(available)}.`}</p>}
    <button className="button full" type="submit" disabled={saving || !valid}>{saving ? "Mengirim..." : "Ajukan pencairan"}</button>
    <button type="button" className="danger-action payout-reject" disabled={saving} onClick={onClose}>Kembali</button>
  </form></div>;
}