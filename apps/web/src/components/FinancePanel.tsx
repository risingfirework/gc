"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { api, FinanceDashboardData, FinanceSettings, TeacherWithdrawalRequest } from "@/services/api";

const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
const kindLabel = { upload_fee: "Honorarium upload", sales_bonus: "Bonus penjualan", refund_reversal: "Koreksi refund" } as const;
const commissionStatus = { pending: "Ditahan", available: "Dapat dicairkan", paid: "Sudah dibayar", cancelled: "Dibatalkan" } as const;
const requestStatus = { submitted: "Menunggu pemeriksaan", approved: "Disetujui", rejected: "Ditolak", cancelled: "Dibatalkan guru", paid: "Berhasil dibayar" } as const;

export default function FinancePanel() {
  const [data, setData] = useState<FinanceDashboardData>();
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [requestTab, setRequestTab] = useState<"queue"|"history">("queue");
  const [reviewTarget, setReviewTarget] = useState<TeacherWithdrawalRequest|null>(null);
  const load = useCallback(() => api.adminFinance().then(setData).catch((reason) => setError(reason instanceof Error ? reason.message : "Data finance gagal dimuat.")), []);
  useEffect(() => { void load(); }, [load]);

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError(""); setMessage("");
    const form = new FormData(event.currentTarget);
    const input: FinanceSettings = { ...data!.settings, teacher_upload_fee:Number(form.get("upload_fee")), teacher_sales_bonus_percent:Number(form.get("sales_bonus")), commission_hold_days:Number(form.get("hold_days")), default_discount_percent:Number(form.get("discount")), minimum_payout:Number(form.get("minimum_payout")), payout_cycle:String(form.get("cycle")) as FinanceSettings["payout_cycle"], auto_payout:false };
    try { await api.updateFinanceSettings(input); await load(); setMessage("Kebijakan komisi berhasil disimpan."); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengaturan gagal disimpan."); }
    finally { setSaving(false); }
  }

  async function review(event:FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!reviewTarget) return;
    const form = new FormData(event.currentTarget);
    const submitter = (event.nativeEvent as SubmitEvent).submitter as HTMLButtonElement|null;
    const status = (submitter?.value || form.get("status")) as "approved"|"rejected"|"paid";
    const note = String(form.get("note")??"").trim();
    if (status==="rejected"&&!note) { setError("Tuliskan alasan penolakan agar guru mengetahui perbaikannya."); return; }
    setSaving(true); setError(""); setMessage("");
    try { await api.reviewPayoutRequest(reviewTarget.id,{status,note,reference:String(form.get("reference")??"").trim()}); await load(); setMessage(`Pengajuan ${reviewTarget.teacher_email} berhasil diperbarui: ${requestStatus[status]}.`); setReviewTarget(null); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengajuan gagal diproses."); }
    finally { setSaving(false); }
  }

  if (!data && !error) return <div className="card finance-loading">Memuat lingkungan finance...</div>;
  if (!data) return <p className="error">{error}</p>;
  const queue = data.payout_requests.filter(item=>item.status==="submitted"||item.status==="approved");
  const requests = requestTab==="queue" ? queue : data.payout_requests.filter(item=>item.status!=="submitted"&&item.status!=="approved");
  return <section className="admin-view finance-view">
    <div className="finance-heading"><div><p className="eyebrow">Finance guru · Versi B</p><h2>Honorarium, bonus, dan pencairan</h2><p className="muted">Seluruh pengajuan terlacak dari pemeriksaan sampai dana diterima guru.</p></div><span className="finance-updated">Diperbarui {data.settings.updated_at ? new Date(data.settings.updated_at).toLocaleString("id-ID") : "—"}</span></div>
    {error&&<p className="error">{error}</p>}{message&&<p className="finance-success">{message}</p>}
    <div className="finance-metrics"><Metric label="Pendapatan dibayar" value={money.format(data.overview.gross_revenue)} hint={`${data.overview.paid_transactions} transaksi berhasil`} tone="blue"/><Metric label="Honorarium upload" value={money.format(data.commission_summary.upload_fees)} hint={`${money.format(data.settings.teacher_upload_fee)} per paket/mapel`} tone="green"/><Metric label="Bonus penjualan" value={money.format(data.commission_summary.sales_bonus)} hint={`${data.settings.teacher_sales_bonus_percent}% pembayaran aktual`} tone="purple"/><Metric label="Antrean pencairan" value={String(queue.length)} hint="pengajuan perlu ditangani" tone="orange"/></div>

    <section className="finance-users payout-request-section"><div className="finance-users-heading"><div><h3>Pengajuan pencairan guru</h3><p className="muted">Periksa tujuan rekening, setujui, lalu catat referensi setelah transfer dilakukan.</p></div></div><div className="content-subtabs"><button className={requestTab==="queue"?"active":""} onClick={()=>setRequestTab("queue")}>Perlu tindakan <span>{queue.length}</span></button><button className={requestTab==="history"?"active":""} onClick={()=>setRequestTab("history")}>Riwayat <span>{data.payout_requests.length-queue.length}</span></button></div><Table headers={["Tanggal","Guru","Tujuan","Nominal","Status","Aksi"]}>{requests.length?requests.map(item=><tr key={item.id}><td>{new Date(item.submitted_at).toLocaleString("id-ID")}</td><td><b>{item.teacher_email}</b></td><td><b>{item.provider}</b><small>{item.account_holder_name} · •••• {item.account_number.slice(-4)}</small></td><td><b>{money.format(item.amount)}</b></td><td><RequestPill status={item.status}/></td><td>{(item.status==="submitted"||item.status==="approved")?<button className="button" onClick={()=>setReviewTarget(item)}>{item.status==="submitted"?"Periksa":"Catat transfer"}</button>:<small>{item.transfer_reference||item.admin_note||"—"}</small>}</td></tr>):<tr><td colSpan={6} className="empty-state">{requestTab==="queue"?"Tidak ada pengajuan yang perlu ditangani.":"Belum ada riwayat pengajuan."}</td></tr>}</Table></section>

    <div className="finance-grid"><form key={data.settings.updated_at} className="card finance-policy" onSubmit={saveSettings}><div className="finance-card-title"><div><h3>Kebijakan Versi B</h3><p>Berlaku sama untuk seluruh guru.</p></div><span>Global</span></div><div className="finance-form-grid"><NumberField name="upload_fee" label="Honorarium paket/mapel" value={data.settings.teacher_upload_fee} prefix="Rp" max={999999999}/><NumberField name="sales_bonus" label="Bonus penjualan" value={data.settings.teacher_sales_bonus_percent} suffix="%"/><NumberField name="hold_days" label="Masa tahan bonus" value={data.settings.commission_hold_days} suffix="hari" max={90}/><NumberField name="discount" label="Diskon katalog" value={data.settings.default_discount_percent} suffix="%"/><NumberField name="minimum_payout" label="Minimum pencairan" value={data.settings.minimum_payout} prefix="Rp" max={999999999}/><label><span>Siklus pencairan</span><select name="cycle" defaultValue={data.settings.payout_cycle}><option value="weekly">Mingguan</option><option value="monthly">Bulanan</option><option value="manual">Manual</option></select></label></div><button className="button full" disabled={saving}>{saving?"Menyimpan...":"Simpan kebijakan"}</button></form><div className="card finance-guide"><p className="eyebrow">Alur profesional</p><h3>Dari komisi sampai diterima</h3><div className="payout-process"><span>1. Guru mengajukan</span><span>2. Admin memeriksa</span><span>3. Admin menyetujui</span><span>4. Dana ditransfer</span></div><ul><li>Saldo terkunci agar tidak dapat diajukan dua kali.</li><li>Data rekening disalin saat pengajuan sebagai arsip.</li><li>Penolakan wajib disertai alasan.</li><li>Pencairan selesai setelah referensi transfer dicatat.</li></ul></div></div>
    <section className="finance-users"><div className="finance-users-heading"><div><h3>Ledger komisi</h3><p className="muted">Rincian honorarium, bonus penjualan, dan koreksi refund.</p></div></div><Table headers={["Tanggal","Guru","Paket/Mapel","Sumber","Dasar","Nominal","Status"]}>{data.commissions.length?data.commissions.map(item=><tr key={item.id}><td>{new Date(item.created_at).toLocaleDateString("id-ID")}</td><td><b>{item.teacher_email}</b></td><td>{item.package_title}{item.invoice_number&&<small>{item.invoice_number}</small>}</td><td>{kindLabel[item.kind]}{item.rate_percent>0&&<small>{item.rate_percent}%</small>}</td><td>{item.kind==="upload_fee"?"Tetap per paket/mapel":money.format(item.base_amount)}</td><td><b>{money.format(item.amount)}</b></td><td><span className={`commission-status ${item.status}`}>{commissionStatus[item.status]}</span></td></tr>):<tr><td colSpan={7} className="empty-state">Belum ada komisi tercatat.</td></tr>}</Table></section>
    {reviewTarget&&<ReviewModal request={reviewTarget} saving={saving} onClose={()=>setReviewTarget(null)} onSubmit={review}/>}
  </section>;
}

function ReviewModal({request,saving,onClose,onSubmit}:{request:TeacherWithdrawalRequest;saving:boolean;onClose:()=>void;onSubmit:(event:FormEvent<HTMLFormElement>)=>void}) {
  const next = request.status==="submitted"?"approved":"paid";
  return <div className="modal-backdrop" onMouseDown={event=>{if(event.target===event.currentTarget&&!saving)onClose()}}><form className="card checkout-modal payout-review-modal" onSubmit={onSubmit} onMouseDown={event=>event.stopPropagation()}><button type="button" className="modal-close" disabled={saving} onClick={onClose}>×</button><p className="eyebrow">{request.status==="submitted"?"Pemeriksaan pengajuan":"Konfirmasi transfer"}</p><h2>{request.teacher_email}</h2><div className="admin-payout-destination"><span>{request.payout_method==="bank_transfer"?"Transfer bank":"Dompet digital"}</span><strong>{request.provider} · {request.account_number}</strong><small>a.n. {request.account_holder_name}{request.phone?` · ${request.phone}`:""}</small></div><div className="checkout-total"><span>Nominal pengajuan</span><strong>{money.format(request.amount)}</strong></div><input type="hidden" name="status" value={next}/>{request.status==="submitted"?<><label>Catatan pemeriksaan <small>(opsional)</small></label><textarea name="note" maxLength={500} placeholder="Tambahkan catatan untuk guru..."/><button className="button full" disabled={saving}>{saving?"Memproses...":"Setujui pengajuan"}</button><button type="submit" name="status" value="rejected" className="danger-action payout-reject" disabled={saving}>Tolak pengajuan</button><small className="form-helper">Alasan wajib diisi jika pengajuan ditolak.</small></>:<><label>Nomor referensi transfer</label><input name="reference" maxLength={120} placeholder="Contoh: TRX-BCA-20260907-001" required autoFocus/><label>Catatan pembayaran <small>(opsional)</small></label><textarea name="note" maxLength={500}/><button className="button full" disabled={saving}>{saving?"Memproses...":"Konfirmasi dana sudah ditransfer"}</button></>}</form></div>;
}
function RequestPill({status}:{status:TeacherWithdrawalRequest["status"]}){return <span className={`request-status ${status}`}>{requestStatus[status]}</span>}
function Metric({label,value,hint,tone}:{label:string;value:string;hint:string;tone:string}){return <article className={`card finance-metric ${tone}`}><span>{label}</span><strong>{value}</strong><small>{hint}</small></article>}
function NumberField({name,label,value,suffix,prefix,max=100}:{name:string;label:string;value?:number|null;suffix?:string;prefix?:string;max?:number}){return <label><span>{label}</span><div className="finance-number">{prefix&&<i>{prefix}</i>}<input name={name} type="number" min="0" max={max} step={max===100?"0.01":max===90?"1":"1000"} defaultValue={value??""} required/>{suffix&&<i>{suffix}</i>}</div></label>}
function Table({headers,children}:{headers:string[];children:React.ReactNode}){return <div className="card finance-table-wrap"><table className="finance-table"><thead><tr>{headers.map(item=><th key={item}>{item}</th>)}</tr></thead><tbody>{children}</tbody></table></div>}
