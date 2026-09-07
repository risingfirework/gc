"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { AdminExam, AdminPackage, AdminQuestion, api, ExamEntry, MasterItem, PackageBundleCreate, SiteSettings, Status, TeacherDashboardData, User } from "@/services/api";
import { PackageExamFields } from "./PackageExamFields";
import ProfileEditor from "./ProfileEditor";
import TKAQuestionFields from "./TKAQuestionFields";
import LogoutButton from "./LogoutButton";
import CatalogStats from "./CatalogStats";
import BrandLogo from "./BrandLogo";
import QuestionBankTools from "./QuestionBankTools";

type TeacherView = "ringkasan" | "katalog" | "komisi" | "pencairan" | "profil";
type Dialog = { kind: "package"; item?: AdminPackage } | { kind: "question"; item?: AdminQuestion } | null;
type DialogSubmit = { form: FormData } | { bundle: PackageBundleCreate } | { update: { form: FormData; exam: ExamEntry } };
const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });

const statusLabel = (status: Status) => (status === "active" ? "Aktif" : "Nonaktif");
function StatusPill({ status }: { status: Status }) {
  return <span className={`status-pill ${status}`}>{statusLabel(status)}</span>;
}
function PackageStatusPill({ status }: { status: Status }) {
  return <span className={`status-pill ${status}`}>{status === "active" ? "Publish" : "Menunggu verifikasi"}</span>;
}

export default function TeacherDashboard({ user, onLogout, loggingOut, onUserUpdate }: { user: User; onLogout: () => void; loggingOut: boolean; onUserUpdate?: (u: User) => void }) {
  const [view, setView] = useState<TeacherView>("ringkasan");
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);
  const [data, setData] = useState<TeacherDashboardData | null>(null);
  const [dialog, setDialog] = useState<Dialog>(null);
  const [catalogStatusTab, setCatalogStatusTab] = useState<Status>("active");
  const [catalogQuery, setCatalogQuery] = useState("");
  const [error, setError] = useState("");
  const [commissionMessage, setCommissionMessage] = useState("");
  const [saving, setSaving] = useState("");
  const [siteSettings,setSiteSettings]=useState<SiteSettings|null>(null);

  const reload = useCallback(async () => {
    const dashboard = await api.teacherDashboard();
    setData(dashboard);
  }, []);
  useEffect(() => {
    Promise.all([api.teacherDashboard(),api.siteSettings()])
      .then(([dashboard,settings])=>{setData(dashboard);setSiteSettings(settings)})
      .catch((reason) => setError(reason instanceof Error ? reason.message : "Data guru gagal dimuat."));
  }, []);

  async function action(task: () => Promise<unknown>) {
    setSaving("mutation"); setError("");
    try { await task(); await reload(); setDialog(null); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Operasi gagal."); }
    finally { setSaving(""); }
  }
  async function remove(kind: "package" | "question", id: string, label: string) {
    if (!window.confirm(`Hapus ${label}? Tindakan ini tidak dapat dibatalkan.`)) return;
    await action(() => kind === "package" ? api.deleteTeacherPackage(id) : api.deleteTeacherQuestion(id));
  }
  async function toggleQuestionStatus(item: AdminQuestion) {
    await action(() => api.updateTeacherQuestion(item.id, { exam_id: item.exam_id, subject_name: item.subject_name, content_text: item.content_text, question_type: item.question_type, presentation_type: "single", group_code: "", stimulus_text: "", question_image_url: item.question_image_url, stimulus_image_url: "", category_labels: item.category_labels, options: item.options, correct_answer: item.correct_answer, score_weight: item.score_weight, explanation_text: item.explanation_text, status: item.status === "active" ? "inactive" : "active" }));
  }
  async function savePayoutAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving("payout-account"); setError(""); setCommissionMessage("");
    try {
      await api.updateTeacherPayoutAccount({ method:String(form.get("method")) as "bank_transfer"|"e_wallet", provider:String(form.get("provider")), account_number:String(form.get("account_number")), account_holder_name:String(form.get("account_holder_name")), phone:String(form.get("phone")) });
      await reload();
      setCommissionMessage("Data pencairan berhasil disimpan dan siap digunakan oleh admin finance.");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Data pencairan gagal disimpan."); }
    finally { setSaving(""); }
  }

  async function requestPayout() {
    setSaving("payout-request"); setError(""); setCommissionMessage("");
    try { const item=await api.createTeacherPayoutRequest(); await reload(); setCommissionMessage(`Pengajuan ${money.format(item.amount)} berhasil dikirim ke admin finance.`); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengajuan pencairan gagal dikirim."); }
    finally { setSaving(""); }
  }
  async function cancelPayoutRequest(id:string) {
    if (!window.confirm("Batalkan pengajuan pencairan ini? Saldo akan kembali tersedia.")) return;
    setSaving("payout-cancel"); setError("");
    try { await api.cancelTeacherPayoutRequest(id); await reload(); setCommissionMessage("Pengajuan berhasil dibatalkan."); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Pengajuan tidak dapat dibatalkan."); }
    finally { setSaving(""); }
  }

  const menus: Array<[TeacherView, string]> = [["ringkasan", "Ringkasan"], ["katalog", "Bank Soal Saya"], ["komisi", "Komisi Saya"], ["pencairan", "Pencairan Dana"], ["profil", "Profil"]];
  const allPackages = data?.packages ?? [];
  const catalogQueryLower = catalogQuery.trim().toLowerCase();
  const visiblePackages = allPackages.filter((item) => item.status === catalogStatusTab && (!catalogQueryLower || item.kode.toLowerCase().includes(catalogQueryLower) || item.title.toLowerCase().includes(catalogQueryLower)));
  const packages = allPackages;
  const selectedPackage = packages.find((item) => item.id === selectedPackageId) ?? null;
  const packageExams = data?.exams.filter((item) => item.package_id === selectedPackageId) ?? [];
  const packageExamIds = new Set(packageExams.map((item) => item.id));
  const packageQuestions = data?.questions.filter((item) => packageExamIds.has(item.exam_id)) ?? [];
  const recentSales = (data?.transactions ?? []).filter((item) => item.status === "paid").slice(0, 8);

  return <main><div className="admin-layout">
    <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/><span>{siteSettings?.platform_name??"TKA Juara"}<small>Panel Guru</small></span></Link><nav>{menus.map(([key, label]) => <button key={key} className={view === key ? "active" : ""} onClick={() => setView(key)}>{label}</button>)}</nav><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout}/></aside>
    <section className="admin-main">
      <header className="admin-topbar"><div><p className="eyebrow">Panel guru</p><h1>{menus.find(([key]) => key === view)?.[1]}</h1></div><button className="admin-identity" onClick={() => setView("profil")}><span>{(user.name?.trim()||user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim()||"Nama belum dilengkapi"}</strong><small>{user.email} · Guru</small></div></button></header>
      {error && <p className="error">{error}</p>}{!data && !error && <div className="card">Memuat data guru...</div>}

      {data && view === "ringkasan" && <div className="admin-view"><div className="admin-stats"><Stat label="Paket Saya" value={data.overview.total_packages} /><Stat label="Ujian" value={data.overview.total_exams} /><Stat label="Soal" value={data.overview.total_questions} /><Stat label="Penjualan" value={data.overview.total_sales} /><Stat label="Pendapatan" value={data.overview.total_revenue} money /></div><section className="teacher-sales-summary"><div className="admin-view-header"><div><p className="eyebrow">Aktivitas terbaru</p><h2>Ringkasan penjualan</h2><p className="muted">Transaksi siswa yang telah berhasil dibayar.</p></div></div><AdminTable headers={["Nomor invoice", "Tanggal transaksi"]}>{recentSales.length ? recentSales.map((item) => <tr key={item.id}><td><strong>{item.invoice_number}</strong></td><td>{new Date(item.created_at).toLocaleDateString("id-ID", { day: "2-digit", month: "long", year: "numeric" })}</td></tr>) : <tr><td colSpan={2} className="empty-state">Belum ada transaksi penjualan.</td></tr>}</AdminTable></section></div>}

      {data && view === "katalog" && !selectedPackage && <section className="admin-view catalog-admin">
        <div className="admin-catalog-heading"><div><p className="eyebrow">Bank soal milik saya</p><h2>Kelola paket, ujian, dan bank soal</h2><p className="muted">Klik salah satu kartu untuk membuka lingkungan pengelolaan lengkap.</p></div><button className="button" onClick={() => setDialog({ kind: "package" })}>+ Tambah paket</button></div>
        <div className="content-subtabs catalog-status-tabs"><button className={catalogStatusTab === "active" ? "active" : ""} onClick={() => setCatalogStatusTab("active")}>Publish <span>{allPackages.filter((item) => item.status === "active").length}</span></button><button className={catalogStatusTab === "inactive" ? "active" : ""} onClick={() => setCatalogStatusTab("inactive")}>Menunggu verifikasi <span>{allPackages.filter((item) => item.status === "inactive").length}</span></button></div>
        <div className="catalog-toolbar"><input className="catalog-search" type="search" placeholder="Cari kode soal atau nama paket..." value={catalogQuery} onChange={(event) => setCatalogQuery(event.target.value)} /></div>
        <div className="admin-package-grid">{visiblePackages.map((item) => <article className={`card admin-package-card ${item.status === "inactive" ? "inactive" : ""}`} key={item.id} onClick={() => setSelectedPackageId(item.id)}><div className="admin-package-card-top"><PackageStatusPill status={item.status} /><span className="package-code">{item.kode}</span></div><span className="package-jenjang">{item.jenjang}</span><h3>{item.title}</h3><p>{item.description}</p><strong>{item.price === 0 ? "Gratis" : money.format(item.price)}</strong><small>{(data.exams.filter((exam) => exam.package_id === item.id).length)} ujian · {data.questions.filter((q) => new Set(data.exams.filter((exam) => exam.package_id === item.id).map((exam) => exam.id)).has(q.exam_id)).length} soal</small>{item.publisher_email && <small className="package-publisher">Pembuat: {item.publisher_email}</small>}{item.status === "active" && <CatalogStats sales={item.sales_count} views={item.view_count}/>}<div><button className="table-action" onClick={(event) => { event.stopPropagation(); setDialog({ kind: "package", item }); }}>Edit</button><button className="danger-action" onClick={(event) => { event.stopPropagation(); void remove("package", item.id, item.title); }}>Hapus</button></div></article>)}
        {allPackages.length === 0 && <div className="card empty-state">Belum ada paket. Klik tombol Tambah paket untuk mulai menjual bank soal.</div>}
        {allPackages.length > 0 && visiblePackages.length === 0 && <div className="card empty-state">Tidak ada paket {catalogStatusTab === "active" ? "yang sudah dipublikasikan" : "yang sedang menunggu verifikasi"} dengan pencarian tersebut.</div>}
        </div>
      </section>}

      {data && view === "katalog" && selectedPackage && <section className="admin-view catalog-detail">
        <div className="catalog-detail-heading"><div><p className="eyebrow">Lingkungan bank soal</p><h2>{selectedPackage.title}</h2></div><div className="catalog-detail-actions"><PackageStatusPill status={selectedPackage.status} /><button className="table-action" onClick={() => setDialog({ kind: "package", item: selectedPackage })}>Edit paket</button><button className="button secondary" onClick={() => setSelectedPackageId(null)}>← Kembali ke daftar</button></div></div>
        <div className="card admin-package-spec"><div className="admin-package-spec-desc"><span>Deskripsi</span><p>{selectedPackage.description || "Tidak ada deskripsi."}</p></div><div className="admin-package-spec-grid"><div><span>Kode soal</span><strong>{selectedPackage.kode}</strong></div><div><span>Jenjang</span><strong>{selectedPackage.jenjang}</strong></div><div><span>Harga</span><strong>{selectedPackage.price === 0 ? "Gratis" : money.format(selectedPackage.price)}</strong></div><div><span>Masa aktif</span><strong>{selectedPackage.validity_days} hari</strong></div><div><span>Ujian</span><strong>{packageExams.length}</strong></div><div><span>Total soal</span><strong>{packageQuestions.length}</strong></div><div><span>Dibuat</span><strong>{new Date(selectedPackage.created_at).toLocaleDateString("id-ID")}</strong></div></div></div>
        <div className="bank-section"><ViewHeader title="Bank soal" templates onAdd={() => setDialog({ kind: "question" })} /><QuestionBankTools packageTitle={selectedPackage.title} packageCode={selectedPackage.kode} level={selectedPackage.jenjang} durationMinutes={packageExams[0]?.duration_minutes} questions={packageQuestions.filter(item=>item.status==="active")}/><AdminTable headers={["Ujian", "Mapel", "Bentuk", "Soal", "Kunci", "Bobot", "Status", "Aksi"]}>{packageQuestions.length ? packageQuestions.map((item) => <tr key={item.id}><td>{item.exam_title}</td><td>{item.subject_name}</td><td>{questionTypeLabel(item.question_type)}</td><td>{item.content_text.length > 60 ? `${item.content_text.slice(0, 60)}...` : item.content_text}</td><td>{item.correct_answer}</td><td>{item.score_weight}</td><td><StatusPill status={item.status} /></td><td><button className="table-action" onClick={() => setDialog({ kind: "question", item })}>Edit</button><button className={`table-action ${item.status === "active" ? "toggle-off" : ""}`} disabled={saving !== ""} onClick={() => void toggleQuestionStatus(item)}>{item.status === "active" ? "Nonaktifkan" : "Aktifkan"}</button><button className="danger-action" onClick={() => void remove("question", item.id, item.subject_name)}>Hapus</button></td></tr>) : <tr><td colSpan={8} className="empty-state">Belum ada soal di paket ini.</td></tr>}</AdminTable></div>
      </section>}

      {data && view === "komisi" && <section className="admin-view teacher-commission-view">
        <div className="admin-view-header"><div><p className="eyebrow">Mekanisme Versi B</p><h2>Honorarium dan bonus saya</h2><p className="muted">Honorarium diperoleh setelah paket diverifikasi admin; bonus dihitung dari pembayaran aktual siswa.</p></div></div>
        <div className="finance-metrics"><CommissionMetric label="Honorarium upload" value={data.commission_summary.upload_fees}/><CommissionMetric label="Bonus penjualan" value={data.commission_summary.sales_bonus}/><CommissionMetric label="Masih ditahan" value={data.commission_summary.held}/><CommissionMetric label="Dapat dicairkan" value={data.commission_summary.available}/><CommissionMetric label="Sudah dibayar" value={data.commission_summary.paid}/></div>
        <div className="admin-view-header"><div><h2>Rincian komisi</h2><p className="muted">Catatan lengkap sumber pendapatan dan statusnya.</p></div><button className="button" onClick={()=>setView("pencairan")}>Ajukan pencairan</button></div>
        <AdminTable headers={["Tanggal","Paket/Mapel","Sumber","Nominal","Status"]}>{data.commissions.length?data.commissions.map(item=><tr key={item.id}><td>{new Date(item.created_at).toLocaleDateString("id-ID")}</td><td>{item.package_title}{item.invoice_number&&<small>{item.invoice_number}</small>}</td><td>{item.kind==="upload_fee"?"Honorarium upload":item.kind==="sales_bonus"?`Bonus penjualan ${item.rate_percent}%`:"Koreksi refund"}</td><td><strong>{money.format(item.amount)}</strong></td><td><PayoutStatus status={item.status}/></td></tr>):<tr><td colSpan={5} className="empty-state">Belum ada komisi tercatat.</td></tr>}</AdminTable>
      </section>}

      {data && view === "pencairan" && <section className="admin-view teacher-commission-view">
        <div className="admin-view-header"><div><p className="eyebrow">Pencairan dana</p><h2>Kelola pengajuan pencairan</h2><p className="muted">Saldo, rekening, dan perkembangan pengajuan tersedia dalam satu halaman.</p></div></div>
        {commissionMessage && <p className="finance-success">{commissionMessage}</p>}
        <div className="card withdrawal-hero"><div><span>Saldo dapat diajukan</span><strong>{money.format(data.commission_summary.available)}</strong><small>Minimum pencairan {money.format(data.finance_settings.minimum_payout)}</small></div><div className="withdrawal-checks"><span className={data.payout_account.updated_at?"ready":""}>1 <small>Rekening lengkap</small></span><span className={data.commission_summary.available>=data.finance_settings.minimum_payout?"ready":""}>2 <small>Minimum terpenuhi</small></span><span className={data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")?"ready":""}>3 <small>Diproses admin</small></span></div><button className="button" disabled={saving!==""||!data.payout_account.updated_at||data.commission_summary.available<data.finance_settings.minimum_payout||data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")} onClick={()=>void requestPayout()}>{saving==="payout-request"?"Mengirim...":data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")?"Pengajuan sedang diproses":"Ajukan seluruh saldo"}</button></div>
        <form key={data.payout_account.updated_at ?? "new"} className="card payout-account-card" onSubmit={savePayoutAccount}>
          <div className="payout-account-heading"><div className="payout-account-icon" aria-hidden="true">Rp</div><div><p className="eyebrow">Tujuan pencairan</p><h2>Data rekening penerima</h2><p className="muted">Pastikan data sesuai dengan rekening aktif agar proses transfer tidak tertunda.</p></div><span className={`payout-readiness ${data.payout_account.updated_at ? "ready" : "pending"}`}>{data.payout_account.updated_at ? "Data lengkap" : "Belum dilengkapi"}</span></div>
          <div className="payout-form-grid">
            <label><span>Metode pencairan</span><select name="method" defaultValue={data.payout_account.method || "bank_transfer"} required><option value="bank_transfer">Transfer bank</option><option value="e_wallet">Dompet digital (e-wallet)</option></select></label>
            <label><span>Bank atau penyedia e-wallet</span><input name="provider" defaultValue={data.payout_account.provider} list="payout-providers" placeholder="Contoh: BCA atau DANA" maxLength={80} autoComplete="organization" required/><datalist id="payout-providers"><option value="BCA"/><option value="Bank Mandiri"/><option value="BNI"/><option value="BRI"/><option value="BSI"/><option value="CIMB Niaga"/><option value="PermataBank"/><option value="Bank Jago"/><option value="SeaBank"/><option value="DANA"/><option value="GoPay"/><option value="OVO"/><option value="ShopeePay"/><option value="LinkAja"/></datalist></label>
            <label><span>Nomor rekening atau akun</span><input name="account_number" defaultValue={data.payout_account.account_number} inputMode="numeric" pattern="[0-9]{6,30}" minLength={6} maxLength={30} placeholder="Masukkan nomor tanpa spasi" autoComplete="off" required/><small>Gunakan angka saja, 6–30 digit.</small></label>
            <label><span>Nama pemilik rekening</span><input name="account_holder_name" defaultValue={data.payout_account.account_holder_name || user.name} maxLength={120} placeholder="Sesuai buku tabungan atau akun" autoComplete="name" required/></label>
            <label><span>Nomor WhatsApp aktif <em>Opsional</em></span><input name="phone" defaultValue={data.payout_account.phone || user.phone} inputMode="tel" pattern="(\+62|62|0)[0-9]{8,13}" maxLength={16} placeholder="Contoh: 081234567890" autoComplete="tel"/><small>Digunakan jika admin perlu konfirmasi pencairan.</small></label>
          </div>
          <div className="payout-form-footer"><div className="payout-security"><span aria-hidden="true">✓</span><p><strong>Khusus administrasi pencairan</strong><small>Informasi ini hanya dapat diakses oleh guru terkait dan admin finance.</small></p></div><button className="button" disabled={saving === "payout-account"}>{saving === "payout-account" ? "Menyimpan..." : data.payout_account.updated_at ? "Perbarui data pencairan" : "Simpan data pencairan"}</button></div>
        </form>
        <div className="admin-view-header"><div><h2>Status pengajuan</h2><p className="muted">Pantau proses pemeriksaan dan transfer oleh admin.</p></div></div>
        <AdminTable headers={["Tanggal","Tujuan","Nominal","Status","Catatan/Aksi"]}>{data.payout_requests.length?data.payout_requests.map(item=><tr key={item.id}><td>{new Date(item.submitted_at).toLocaleString("id-ID")}</td><td>{item.provider}<small>•••• {item.account_number.slice(-4)}</small></td><td><strong>{money.format(item.amount)}</strong></td><td><PayoutStatus status={item.status}/></td><td>{item.status==="submitted"?<button className="danger-action" disabled={saving!==""} onClick={()=>void cancelPayoutRequest(item.id)}>Batalkan</button>:item.transfer_reference||item.admin_note||"—"}</td></tr>):<tr><td colSpan={5} className="empty-state">Belum ada pengajuan pencairan.</td></tr>}</AdminTable>
      </section>}

      {view === "profil" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div><ProfileEditor user={user} onUpdated={onUserUpdate} /></section>}
    </section>
    {dialog && <CrudDialog dialog={dialog} exams={selectedPackageId ? packageExams : data?.exams ?? []} levels={data?.levels ?? []} mapels={data?.mapels ?? []} academicYears={data?.academic_years ?? []} siteSettings={siteSettings} defaultQuestionSubject={packageQuestions[0]?.subject_name} user={user} busy={saving !== ""} onClose={() => setDialog(null)} onSubmit={(value) => void action(async () => {
      if (dialog.kind === "package") { if (!dialog.item && "bundle" in value) { await api.createTeacherPackageBundle(value.bundle); } else if ("update" in value && dialog.item) { const { form, exam } = value.update; await api.updateTeacherPackage(dialog.item.id, { title: String(form.get("title")), kode: String(form.get("kode")), description: String(form.get("description")), price: Number(form.get("price")), validity_days: Number(form.get("validity_days")), status: String(form.get("status")) as Status, jenjang: String(form.get("jenjang")) as User["school_level"] }); const examTarget = data?.exams.find((item) => item.package_id === dialog.item!.id); if (examTarget) await api.updateTeacherExam(examTarget.id, { package_id: examTarget.package_id, title: exam.title, mapel_id: exam.mapel_id, tahun_ajaran_id: exam.tahun_ajaran_id, duration_minutes: exam.duration_minutes, total_questions: exam.total_questions, passing_score: exam.passing_score, status: exam.status }); } }
      if (dialog.kind === "question" && "form" in value) { const form = value.form; const input = { exam_id: String(form.get("exam_id")), subject_name: String(form.get("subject_name")), content_text: String(form.get("content_text")), question_type: String(form.get("question_type")) as AdminQuestion["question_type"], presentation_type: String(form.get("presentation_type")) as AdminQuestion["presentation_type"], group_code: String(form.get("group_code")), stimulus_text: String(form.get("stimulus_text")), question_image_url: String(form.get("question_image_url")), stimulus_image_url: String(form.get("stimulus_image_url")), category_labels: JSON.parse(String(form.get("category_labels_json"))) as string[], options: JSON.parse(String(form.get("options_json"))) as { key: string; content: string; image_url?: string }[], correct_answer: String(form.get("correct_answer")), score_weight: Number(form.get("score_weight")), explanation_text: String(form.get("explanation_text")), status: String(form.get("status")) as Status }; if (dialog.item) await api.updateTeacherQuestion(dialog.item.id, input); else await api.createTeacherQuestion(input); }
    })}/>}
  </div></main>;
}

function CrudDialog({ dialog, exams, levels, mapels, academicYears, siteSettings, defaultQuestionSubject, user, busy, onClose, onSubmit }: { dialog: Exclude<Dialog, null>; exams: AdminExam[]; levels: string[]; mapels: MasterItem[]; academicYears: MasterItem[]; siteSettings:SiteSettings|null; defaultQuestionSubject?: string; user: User; busy: boolean; onClose: () => void; onSubmit: (value: DialogSubmit) => void }) {
  const [options, setOptions] = useState<{ key: string; content: string }[]>(dialog.kind === "question" ? (dialog.item?.options ?? [{ key: "A", content: "" }, { key: "B", content: "" }]) : []);
  const bundleRef = useRef<{ getValues: () => ExamEntry }>(null);
  const questionExam = dialog.kind === "question" ? exams.find((item) => item.id === dialog.item?.exam_id) ?? exams[0] : undefined;
  const questionSubject = dialog.kind === "question" ? mapels.find((item) => item.id === questionExam?.mapel_id)?.nama ?? dialog.item?.subject_name ?? defaultQuestionSubject ?? "" : "";
  const missingQuestionContext = dialog.kind === "question" && (!questionExam || !questionSubject);
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    if (dialog.kind === "question") form.set("options_json", JSON.stringify(options));
    if (dialog.kind === "package" && bundleRef.current) { const exam = bundleRef.current.getValues(); const pkg = { title: String(form.get("title")), kode: String(form.get("kode")), description: String(form.get("description")), price: Number(form.get("price")), validity_days: Number(form.get("validity_days")), status: String(form.get("status")) as Status, jenjang: String(form.get("jenjang")) as User["school_level"] }; if (!dialog.item) onSubmit({ bundle: { package: pkg, exam: { ...exam, title: String(form.get("jenjang")) } } }); else onSubmit({ update: { form, exam } }); }
    else onSubmit({ form });
  }
  return <div className="modal-backdrop"><form className="card admin-form" onSubmit={submit}><button type="button" className="modal-close" onClick={onClose}>x</button><h2>{"item" in dialog && dialog.item ? "Edit" : "Tambah"} {dialog.kind === "package" ? "Paket Soal" : "Soal"}</h2>
    {dialog.kind === "package" && <><label>Nama paket</label><input name="title" defaultValue={dialog.item?.title} required /><div className="form-grid"><div><label>Kode soal</label><input name="kode" defaultValue={dialog.item?.kode} placeholder="cth: TKA-2026-SMA" maxLength={50} required /></div><div><label>Jenjang</label><select name="jenjang" defaultValue={dialog.item?.jenjang ?? "SMA"} required>{levels.map((level) => <option key={level} value={level}>{level}</option>)}</select></div></div><label>Pembuat paket soal</label><input value={user.email} disabled /><label>Deskripsi</label><textarea name="description" defaultValue={dialog.item?.description} /><div className="form-grid"><div><label>Harga</label><div className="price-input"><span>Rp</span><input name="price" type="number" min="0" step="1000" defaultValue={dialog.item?.price ?? 0} onChange={(event) => { const value = Number(event.target.value); event.target.setCustomValidity(value > 0 && value % 1000 !== 0 ? "Harga harus kelipatan 1.000" : ""); }} required /></div></div><div><label>Masa aktif (hari)</label><input name="validity_days" type="number" min="1" defaultValue={dialog.item?.validity_days ?? siteSettings?.default_package_validity_days ?? 7} required /></div></div><input type="hidden" name="status" value="inactive"/><p className="form-helper">Paket masuk ke antrean verifikasi admin. Honorarium upload dicatat otomatis setelah paket disetujui dan dipublikasikan.</p><hr /><PackageExamFields ref={bundleRef} mapels={mapels} academicYears={academicYears} defaults={siteSettings?{duration_minutes:siteSettings.default_exam_duration_minutes,total_questions:siteSettings.default_exam_total_questions,passing_score:siteSettings.default_passing_score}:undefined} defaultExam={dialog.item ? exams.find((item) => item.package_id === dialog.item?.id) : undefined} /></>}
    {dialog.kind === "question" && <><input type="hidden" name="exam_id" value={questionExam?.id ?? ""} readOnly /><input type="hidden" name="subject_name" value={questionSubject} readOnly />{missingQuestionContext && <p className="error">Lengkapi mata pelajaran melalui Edit paket sebelum menambah soal.</p>}<TKAQuestionFields item={dialog.item} options={options} setOptions={setOptions}/><div className="form-grid"><div><label>Bobot soal</label><input name="score_weight" type="number" min="0.001" step="0.001" defaultValue={dialog.item?.score_weight ?? 1} required /></div><div><label>Status</label><select name="status" defaultValue={dialog.item?.status ?? "active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></div></div><label>Pembahasan</label><textarea name="explanation_text" defaultValue={dialog.item?.explanation_text} /></>}
    <button className="button full" disabled={busy || missingQuestionContext} type="submit">{busy ? "Menyimpan..." : "Simpan"}</button>
  </form></div>;
}
function ViewHeader({ title, onAdd, templates = false }: { title: string; onAdd: () => void; templates?: boolean }) { return <div className="admin-view-header"><h2>{title}</h2><div className="template-actions">{templates && <><a className="button secondary" href="/templates/template-bank-soal-tka.docx" download>Template Word</a><a className="button secondary" href="/templates/template-bank-soal-tka.xlsx" download>Template Excel</a></>}<button className="button" onClick={onAdd}>+ Tambah</button></div></div>; }
function questionTypeLabel(type: AdminQuestion["question_type"]) { return type === "multiple_choice" ? "PGK MCMA" : type === "category" ? "PGK Kategori" : "PG Sederhana"; }
function Stat({ label, value, money: isMoney }: { label: string; value: number; money?: boolean }) { return <article className="card admin-stat"><span>{label}</span><strong>{isMoney ? money.format(value) : value}</strong></article>; }
function CommissionMetric({label,value}:{label:string;value:number}) { return <article className="card finance-metric"><span>{label}</span><strong>{money.format(value)}</strong></article>; }
function PayoutStatus({status}:{status:string}) { const labels:Record<string,string>={pending:"Ditahan",available:"Tersedia",submitted:"Menunggu pemeriksaan",approved:"Disetujui",paid:"Berhasil dibayar",rejected:"Ditolak",cancelled:"Dibatalkan"}; return <span className={`request-status ${status}`}>{labels[status]??status}</span>; }
function AdminTable({ headers, children }: { headers: string[]; children: React.ReactNode }) { return <div className="card admin-table-card"><div className="ranking-table-wrap"><table className="ranking-table"><thead><tr>{headers.map((header) => <th key={header}>{header}</th>)}</tr></thead><tbody>{children}</tbody></table></div></div>; }
