"use client";

import Link from "next/link";
import { ChangeEvent, FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { AdminExam, AdminPackage, AdminQuestion, api, CBTPublishSetting, ExamEntry, MasterItem, PackageBundleCreate, SiteSettings, Status, TeacherDashboardData, User } from "@/services/api";
import { PackageExamFields } from "./PackageExamFields";
import ProfileEditor from "./ProfileEditor";
import TKAQuestionFields from "./TKAQuestionFields";
import LogoutButton from "./LogoutButton";
import CatalogStats from "./CatalogStats";
import BrandLogo from "./BrandLogo";
import QuestionBankTools from "./QuestionBankTools";
import DataImage from "./DataImage";
import ConfirmModal from "./ConfirmModal";
import CBTSettingsPanel from "./CBTSettingsPanel";

type TeacherView = "ringkasan" | "katalog" | "cbt" | "komisi" | "pencairan" | "profil";
type Dialog = { kind: "package"; item?: AdminPackage; defaultExamType?: "sell" | "cbt" } | { kind: "question"; item?: AdminQuestion } | null;
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
  const [toast, setToast] = useState("");
  const [catalogStatusTab, setCatalogStatusTab] = useState<Status>("active");
  const [catalogQuery, setCatalogQuery] = useState("");
  const [error, setError] = useState("");
  const [commissionMessage, setCommissionMessage] = useState("");
  const [saving, setSaving] = useState("");
  const [siteSettings,setSiteSettings]=useState<SiteSettings|null>(null);
  const [appealPreview, setAppealPreview] = useState("");
  const [appealSaving, setAppealSaving] = useState(false);
  const [payoutModal, setPayoutModal] = useState(false);
  const [selectedQuestionIds, setSelectedQuestionIds] = useState<Set<string>>(new Set());
  const [removeTarget, setRemoveTarget] = useState<{ kind: "package" | "question"; id: string; label: string; isCBT?: boolean } | null>(null);
  const [bulkDeleteTarget, setBulkDeleteTarget] = useState<{ mode: "all" | "selected"; packageId: string; count: number } | null>(null);
  const [cbtSettings, setCbtSettings] = useState<CBTPublishSetting[]>([]);

  const verificationStatus = user.teacher_verification_status ?? "approved";

  const reload = useCallback(async () => {
    const dashboard = await api.teacherDashboard();
    setData(dashboard);
  }, []);
  useEffect(() => {
    if ((user.teacher_verification_status ?? "approved") !== "approved") return;
    Promise.all([api.teacherDashboard(), api.siteSettings()])
      .then(([dashboard, settings]) => { setData(dashboard); setSiteSettings(settings); })
      .catch((reason) => setError(reason instanceof Error ? reason.message : "Data guru gagal dimuat."));
  }, [user.teacher_verification_status]);
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(""), 3500); return () => window.clearTimeout(timer); }, [toast]);
  useEffect(() => {
    if (view !== "cbt") return;
    api.teacherCBTSettings().then(setCbtSettings).catch((reason) => setError(reason instanceof Error ? reason.message : "Pengaturan CBT gagal dimuat."));
  }, [view]);

  async function action(task: () => Promise<unknown>, success?: string) {
    setSaving("mutation"); setError("");
    try { await task(); await reload(); setDialog(null); if (success) setToast(success); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Operasi gagal."); }
    finally { setSaving(""); }
  }
  function remove(kind: "package" | "question", id: string, label: string, isCBT = false) {
    setRemoveTarget({ kind, id, label, isCBT });
  }
  async function confirmRemove() {
    if (!removeTarget) return;
    const target = removeTarget;
    setRemoveTarget(null);
    const message = target.kind === "package" ? "Paket soal berhasil dihapus." : "Soal berhasil dihapus.";
    await action(() => target.kind === "package" ? api.deleteTeacherPackage(target.id) : api.deleteTeacherQuestion(target.id), message);
  }
  async function toggleQuestionStatus(item: AdminQuestion) {
    await action(() => api.updateTeacherQuestion(item.id, { exam_id: item.exam_id, subject_name: item.subject_name, content_text: item.content_text, question_type: item.question_type, presentation_type: "single", group_code: "", stimulus_text: "", question_image_url: item.question_image_url, stimulus_image_url: "", category_labels: item.category_labels, options: item.options, correct_answer: item.correct_answer, score_weight: item.score_weight, explanation_text: item.explanation_text, status: item.status === "active" ? "inactive" : "active" }));
  }
  function toggleQuestionSelection(id: string) {
    setSelectedQuestionIds((current) => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; });
  }
  async function confirmBulkDeleteQuestions() {
    if (!bulkDeleteTarget) return;
    const target = bulkDeleteTarget;
    setBulkDeleteTarget(null);
    const ids = target.mode === "selected" ? [...selectedQuestionIds] : [];
    await action(async () => { await api.bulkDeleteTeacherQuestions(ids, ids.length ? undefined : target.packageId); setSelectedQuestionIds(new Set()); });
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

  async function requestPayout(amount:number) {
    setSaving("payout-request"); setError(""); setCommissionMessage("");
    try { const item=await api.createTeacherPayoutRequest(amount); await reload(); setCommissionMessage(`Pengajuan ${money.format(item.amount)} berhasil dikirim ke admin finance.`); }
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

  const menus: Array<[TeacherView, string]> = [["ringkasan", "Ringkasan"], ["katalog", "Bank Soal Saya"], ["cbt", "Bank Soal CBT"], ["komisi", "Komisi Saya"], ["pencairan", "Pencairan Dana"], ["profil", "Profil"]];
  const allPackages = data?.packages ?? [];
  const catalogQueryLower = catalogQuery.trim().toLowerCase();
  const visiblePackages = allPackages.filter((item) => (view === "cbt" ? item.exam_type === "cbt" : item.exam_type !== "cbt") && item.status === catalogStatusTab && (!catalogQueryLower || item.kode.toLowerCase().includes(catalogQueryLower) || item.title.toLowerCase().includes(catalogQueryLower)));
  const packages = allPackages;
  const selectedPackage = packages.find((item) => item.id === selectedPackageId) ?? null;
  const packageExams = data?.exams.filter((item) => item.package_id === selectedPackageId) ?? [];
  const packageExamIds = new Set(packageExams.map((item) => item.id));
  const packageQuestions = data?.questions.filter((item) => packageExamIds.has(item.exam_id)) ?? [];
  const recentSales = (data?.transactions ?? []).filter((item) => item.status === "paid").slice(0, 8);

  const navigate = (next: TeacherView) => {
    setView(next);
    setSelectedPackageId(null);
    setCatalogQuery("");
  };

  function pickAppealFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError("");
    if (!file.type.startsWith("image/")) { setError("Bukti sanggah harus berupa gambar."); return; }
    if (file.size > 4 * 1024 * 1024) { setError("Ukuran gambar maksimal 4 MB."); return; }
    const reader = new FileReader();
    reader.onload = () => setAppealPreview(String(reader.result ?? ""));
    reader.onerror = () => setError("Gambar gagal dibaca, coba lagi.");
    reader.readAsDataURL(file);
  }
  async function submitAppeal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (appealSaving || !appealPreview) return;
    setAppealSaving(true); setError("");
    try {
      await api.teacherAppeal(appealPreview);
      const current = await api.currentUser();
      onUserUpdate?.(current);
      setAppealPreview("");
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Sanggah gagal dikirim."); }
    finally { setAppealSaving(false); }
  }

  if (verificationStatus !== "approved") {
    const pending = verificationStatus === "pending" || verificationStatus === "rejected";
    return <main><div className="admin-layout teacher-verification-gate">
      <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/><span>{siteSettings?.platform_name??"TKA Juara"}<small>Verifikasi Akun Guru</small></span></Link><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout}/></aside>
      <section className="admin-main">
        <header className="admin-topbar"><div><p className="eyebrow">Verifikasi data guru</p><h1>{verificationStatus === "rejected" ? "Pendaftaran perlu ditinjau ulang" : "Menunggu persetujuan operator"}</h1></div><button className="admin-identity"><span>{(user.name?.trim()||user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim()||"Nama belum dilengkapi"}</strong><small>{user.email} · Guru</small></div></button></header>
        {error && <p className="error">{error}</p>}
        <div className="card gate-card">
          <span className="gate-badge">{verificationStatus === "rejected" ? "DITOLAK" : "PENDING"}</span>
          <h2>{pending && verificationStatus === "rejected" ? "Pendaftaran Anda belum disetujui" : "Mohon Tunggu"}</h2>
          <p>Mohon Tunggu, data anda sedang tahap verifikasi bahwa anda terdaftar dalam data pendidikan nasional. Operator akan memeriksa No. KTP <strong>{user.teacher_ktp || "—"}</strong> Anda.</p>
          {verificationStatus === "pending" && user.teacher_appeal_image ? <><div className="gate-appeal-sent"><span aria-hidden="true">✓</span><p><strong>Bukti sanggah telah dikirim</strong><small>Bukti Anda sedang diperiksa ulang oleh operator.</small></p></div><div className="gate-proof-preview"><DataImage src={user.teacher_appeal_image} alt="Bukti sanggah terkirim"/></div></> : null}
          {verificationStatus === "rejected" && <>
            <div className="gate-rejection"><span aria-hidden="true">!</span><p><strong>Alasan penolakan</strong><small>{user.teacher_rejection_reason || "Data tidak dapat diverifikasi dari data pendidikan nasional."}</small></p></div>
            <form className="gate-appeal-form" onSubmit={submitAppeal}>
              <label className="gate-file-browser"><span>Upload bukti sanggah</span><input name="appeal_proof" type="file" accept="image/*" onChange={pickAppealFile} required/><small>Lampirkan gambar KTP atau surat keterangan mengajar dari kepala sekolah.</small></label>
              {appealPreview && <div className="gate-proof-preview"><DataImage src={appealPreview} alt="Pratinjau bukti sanggah"/></div>}
              <button className="button" disabled={appealSaving || !appealPreview}>{appealSaving ? "Mengirim bukti..." : "Kirim bukti sanggah"}</button>
            </form>
          </>}
        </div>
      </section>
    </div></main>;
  }

  return <main><div className="admin-layout">
    <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/><span>{siteSettings?.platform_name??"TKA Juara"}<small>Panel Guru</small></span></Link><nav>{menus.map(([key, label]) => <button key={key} className={view === key ? "active" : ""} onClick={() => navigate(key)}>{label}</button>)}</nav><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout}/></aside>
    <section className="admin-main">
      <header className="admin-topbar"><div><p className="eyebrow">Panel guru</p><h1>{menus.find(([key]) => key === view)?.[1]}</h1></div><button className="admin-identity" onClick={() => navigate("profil")}><span>{(user.name?.trim()||user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim()||"Nama belum dilengkapi"}</strong><small>{user.email} · Guru</small></div></button></header>
      {error && <p className="error">{error}</p>}{!data && !error && <div className="card">Memuat data guru...</div>}

      {data && view === "ringkasan" && <div className="admin-view"><div className="admin-stats"><Stat label="Paket Saya" value={data.overview.total_packages} /><Stat label="Ujian" value={data.overview.total_exams} /><Stat label="Soal" value={data.overview.total_questions} /><Stat label="Penjualan" value={data.overview.total_sales} /><Stat label="Pendapatan" value={data.overview.total_revenue} money /></div><section className="teacher-sales-summary"><div className="admin-view-header"><div><p className="eyebrow">Aktivitas terbaru</p><h2>Ringkasan penjualan</h2><p className="muted">Transaksi siswa yang telah berhasil dibayar.</p></div></div><AdminTable headers={["Nomor invoice", "Tanggal transaksi"]}>{recentSales.length ? recentSales.map((item) => <tr key={item.id}><td><strong>{item.invoice_number}</strong></td><td>{new Date(item.created_at).toLocaleDateString("id-ID", { day: "2-digit", month: "long", year: "numeric" })}</td></tr>) : <tr><td colSpan={2} className="empty-state">Belum ada transaksi penjualan.</td></tr>}</AdminTable></section></div>}

      {data && (view === "katalog" || view === "cbt") && !selectedPackage && <section className="admin-view catalog-admin">
        <div className="admin-catalog-heading"><div><p className="eyebrow">{view === "cbt" ? "Bank soal CBT milik saya" : "Bank soal milik saya"}</p><h2>{view === "cbt" ? "Paket ujian CBT yang kamu buat" : "Kelola paket, ujian, dan bank soal"}</h2><p className="muted">{view === "cbt" ? "Hanya menampilkan paket dengan tipe CBT." : "Klik salah satu kartu untuk membuka lingkungan pengelolaan lengkap."}</p></div><button className="button" onClick={() => setDialog(view === "cbt" ? { kind: "package", defaultExamType: "cbt" } : { kind: "package" })}>+ Tambah paket</button></div>
        <div className="content-subtabs catalog-status-tabs"><button className={catalogStatusTab === "active" ? "active" : ""} onClick={() => setCatalogStatusTab("active")}>Publish <span>{allPackages.filter((item) => (view === "cbt" ? item.exam_type === "cbt" : item.exam_type !== "cbt") && item.status === "active").length}</span></button><button className={catalogStatusTab === "inactive" ? "active" : ""} onClick={() => setCatalogStatusTab("inactive")}>Menunggu verifikasi <span>{allPackages.filter((item) => (view === "cbt" ? item.exam_type === "cbt" : item.exam_type !== "cbt") && item.status === "inactive").length}</span></button></div>
        <div className="catalog-toolbar"><input className="catalog-search" type="search" placeholder="Cari kode soal atau nama paket..." value={catalogQuery} onChange={(event) => setCatalogQuery(event.target.value)} /></div>
        <div className="admin-package-grid">{visiblePackages.map((item) => <article className={`card admin-package-card ${item.status === "inactive" ? "inactive" : ""}`} key={item.id} onClick={() => setSelectedPackageId(item.id)}><div className="admin-package-card-top"><PackageStatusPill status={item.status} /><span className="package-code">{item.kode}</span></div><span className="package-jenjang">{item.jenjang}</span><span className="package-exam-type">{item.exam_type === "cbt" ? "CBT" : "Sell"}</span>{item.kategori_name ? <small className="package-publisher">Kategori: {item.kategori_name}{item.kelas_name ? ` · Kelas ${item.kelas_name}` : ""}</small> : null}<h3>{item.title}</h3><p>{item.description}</p><strong>{item.price === 0 ? "Gratis" : money.format(item.price)}</strong><small>{(data.exams.filter((exam) => exam.package_id === item.id).length)} ujian · {data.questions.filter((q) => new Set(data.exams.filter((exam) => exam.package_id === item.id).map((exam) => exam.id)).has(q.exam_id)).length} soal</small>{item.publisher_email && <small className="package-publisher">Pembuat: {item.publisher_email}</small>}{item.status === "active" && <CatalogStats sales={item.sales_count} views={item.view_count}/>}<div><button className="table-action" onClick={(event) => { event.stopPropagation(); setDialog({ kind: "package", item }); }}>Edit</button><button className="danger-action" onClick={(event) => { event.stopPropagation(); void remove("package", item.id, item.title, item.exam_type === "cbt"); }}>Hapus</button></div></article>)}
        {allPackages.length === 0 && <div className="card empty-state">Belum ada paket. Klik tombol Tambah paket untuk mulai menjual bank soal.</div>}
        {allPackages.length > 0 && visiblePackages.length === 0 && <div className="card empty-state">Tidak ada paket {catalogStatusTab === "active" ? "yang sudah dipublikasikan" : "yang sedang menunggu verifikasi"} dengan pencarian tersebut.</div>}
        </div>
      </section>}

      {data && (view === "katalog" || view === "cbt") && selectedPackage && <section className="admin-view catalog-detail">
        <div className="catalog-detail-heading"><div><p className="eyebrow">Lingkungan bank soal</p><h2>{selectedPackage.title}</h2></div><div className="catalog-detail-actions"><PackageStatusPill status={selectedPackage.status} /><button className="table-action" onClick={() => setDialog({ kind: "package", item: selectedPackage })}>Edit paket</button><button className="button secondary" onClick={() => setSelectedPackageId(null)}>← Kembali ke daftar</button></div></div>
        <div className="card admin-package-spec"><div className="admin-package-spec-desc"><span>Deskripsi</span><p>{selectedPackage.description || "Tidak ada deskripsi."}</p></div><div className="admin-package-spec-grid"><div><span>Kode soal</span><strong>{selectedPackage.kode}</strong></div><div><span>Jenjang</span><strong>{selectedPackage.jenjang}</strong></div><div><span>Harga</span><strong>{selectedPackage.price === 0 ? "Gratis" : money.format(selectedPackage.price)}</strong></div>{selectedPackage.exam_type === "cbt" && <div><span>Kode rahasia (token CBT)</span><strong>{selectedPackage.cbt_token || "—"}</strong></div>}<div><span>Masa aktif</span><strong>{selectedPackage.validity_days} hari</strong></div><div><span>Ujian</span><strong>{packageExams.length}</strong></div><div><span>Total soal</span><strong>{packageQuestions.length}</strong></div><div><span>Dibuat</span><strong>{new Date(selectedPackage.created_at).toLocaleDateString("id-ID")}</strong></div></div></div>
        <div className="bank-section">{view === "cbt" && <div className="bank-subsection"><div className="section-heading"><div><p className="eyebrow">Publish pembahasan</p><h3>Pembahasan CBT</h3></div></div><p className="muted" style={{ margin: "0 0 12px" }}>Publikasikan pembahasan agar siswa yang sudah mengerjakan dapat melihat analitik hasil ujiannya.</p><CBTSettingsPanel items={cbtSettings.filter((item) => item.package_id === selectedPackage.id)} saving={saving !== ""} loadParticipants={(examID) => api.teacherCBTParticipants(examID)} onToggle={async (item, publish) => { setSaving("cbt-publish"); try { const updated = await api.teacherSetCBTPublish(item.exam_id, publish); setCbtSettings((current) => current.map((it) => it.exam_id === updated.exam_id ? updated : it)); setError(""); } catch (reason) { setError(reason instanceof Error ? reason.message : "Gagal mengubah pengaturan CBT."); } finally { setSaving(""); } }}/></div>}<ViewHeader title="Bank soal" templates onAdd={() => setDialog({ kind: "question" })} /><QuestionBankTools packageTitle={selectedPackage.title} packageCode={selectedPackage.kode} level={selectedPackage.jenjang} durationMinutes={packageExams[0]?.duration_minutes} questions={packageQuestions.filter(item=>item.status==="active")} examId={packageExams[0]?.id} subjectName={(data?.mapels??[]).find((item)=>item.id===packageExams[0]?.mapel_id)?.nama??packageQuestions[0]?.subject_name??""} onImportQuestion={async (input)=>{const created=await api.createTeacherQuestion(input); setData((current)=>current?{...current,questions:current.questions.some((item)=>item.id===created.id)?current.questions:[...current.questions,created]}:current);}}/><AdminTable headers={["", "Ujian", "Mapel", "Bentuk", "Soal", "Kunci", "Bobot", "Status", "Aksi"]}>{packageQuestions.length ? packageQuestions.map((item) => <tr key={item.id}><td><input type="checkbox" aria-label={`Pilih soal ${item.subject_name}`} checked={selectedQuestionIds.has(item.id)} onChange={() => toggleQuestionSelection(item.id)} /></td><td>{item.exam_title}</td><td>{item.subject_name}</td><td>{questionTypeLabel(item.question_type)}</td><td>{item.content_text.length > 60 ? `${item.content_text.slice(0, 60)}...` : item.content_text}</td><td>{item.correct_answer}</td><td>{item.score_weight}</td><td><StatusPill status={item.status} /></td><td><button className="table-action" onClick={() => setDialog({ kind: "question", item })}>Edit</button><button className={`table-action ${item.status === "active" ? "toggle-off" : ""}`} disabled={saving !== ""} onClick={() => void toggleQuestionStatus(item)}>{item.status === "active" ? "Nonaktifkan" : "Aktifkan"}</button><button className="danger-action" onClick={() => void remove("question", item.id, item.subject_name)}>Hapus</button></td></tr>) : <tr><td colSpan={9} className="empty-state">Belum ada soal di paket ini.</td></tr>}</AdminTable>{packageQuestions.length > 0 && <div className="bulk-actions"><button className="button secondary" type="button" disabled={!selectedQuestionIds.size || saving !== ""} onClick={() => setBulkDeleteTarget({ mode: "selected", packageId: selectedPackage.id, count: selectedQuestionIds.size })}>Hapus soal terpilih ({selectedQuestionIds.size})</button><button className="danger-action" type="button" disabled={packageQuestions.length === 0 || saving !== ""} onClick={() => setBulkDeleteTarget({ mode: "all", packageId: selectedPackage.id, count: packageQuestions.length })}>Hapus semua soal</button></div>}</div>
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
        <div className="card withdrawal-hero"><div><span>Saldo dapat diajukan</span><strong>{money.format(data.commission_summary.available)}</strong><small>Minimum pencairan {money.format(data.finance_settings.minimum_payout)}</small></div><div className="withdrawal-checks"><span className={data.payout_account.updated_at?"ready":""}>1 <small>Rekening lengkap</small></span><span className={data.commission_summary.available>=data.finance_settings.minimum_payout?"ready":""}>2 <small>Minimum terpenuhi</small></span><span className={data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")?"ready":""}>3 <small>Diproses admin</small></span></div><button className="button" disabled={saving!==""||!data.payout_account.updated_at||data.commission_summary.available<data.finance_settings.minimum_payout||data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")} onClick={()=>setPayoutModal(true)}>{data.payout_requests.some(item=>item.status==="submitted"||item.status==="approved")?"Pengajuan sedang diproses":"Ajukan pencairan"}</button></div>
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
        <AdminTable headers={["Tanggal","Tujuan","Nominal","Status","Catatan/Aksi"]}>{data.payout_requests.length?data.payout_requests.map(item=><tr key={item.id}><td>{new Date(item.submitted_at).toLocaleString("id-ID")}</td><td>{item.provider}<small>•••• {item.account_number.slice(-4)}</small></td><td><strong>{money.format(item.amount)}</strong></td><td><PayoutStatus status={item.status}/></td><td>{item.status==="submitted"?<button className="danger-action" disabled={saving!==""} onClick={()=>void cancelPayoutRequest(item.id)}>Batalkan</button>:item.status==="paid"&&item.proof_url?<a className="proof-link" href={item.proof_url} target="_blank" rel="noreferrer">Lihat bukti transfer</a>:item.transfer_reference||item.admin_note||"—"}</td></tr>):<tr><td colSpan={5} className="empty-state">Belum ada pengajuan pencairan.</td></tr>}</AdminTable>
      </section>}

      {view === "profil" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div><ProfileEditor user={user} onUpdated={onUserUpdate} /></section>}
    </section>
    {dialog && <CrudDialog dialog={dialog} exams={selectedPackageId ? packageExams : data?.exams ?? []} levels={data?.levels ?? []} mapels={data?.mapels ?? []} academicYears={data?.academic_years ?? []} kategoris={data?.kategoris ?? []} kelas={data?.kelas ?? []} siteSettings={siteSettings} defaultQuestionSubject={packageQuestions[0]?.subject_name} user={user} busy={saving !== ""} onClose={() => setDialog(null)} onSubmit={(value) => { const successBox = dialog.kind === "package" ? (dialog.item ? "Paket soal berhasil diperbarui." : "Paket soal berhasil dibuat.") : "Soal berhasil disimpan."; void action(async () => {
      if (dialog.kind === "package") { if (!dialog.item && "bundle" in value) { await api.createTeacherPackageBundle(value.bundle); } else if ("update" in value && dialog.item) { const { form, exam } = value.update; const examType = String(form.get("exam_type")) as AdminPackage["exam_type"]; const isCbt = examType === "cbt"; const startDateRaw = form.get("start_date"); const endDateRaw = form.get("end_date"); await api.updateTeacherPackage(dialog.item.id, { title: String(form.get("title")), kode: String(form.get("kode")), description: String(form.get("description")), price: Number(form.get("price")), validity_days: Number(form.get("validity_days")), status: String(form.get("status")) as Status, jenjang: String(form.get("jenjang")) as User["school_level"], exam_type: examType, cbt_token: String(form.get("cbt_token") ?? ""), start_date: isCbt && startDateRaw ? new Date(String(startDateRaw) + "T00:00:00Z").toISOString() : undefined, end_date: isCbt && endDateRaw ? new Date(String(endDateRaw) + "T23:59:59Z").toISOString() : undefined, kategori_id: String(form.get("kategori_id")), kelas_id: String(form.get("kelas_id")) }); const examTarget = data?.exams.find((item) => item.package_id === dialog.item!.id); if (examTarget) await api.updateTeacherExam(examTarget.id, { package_id: examTarget.package_id, title: exam.title, mapel_id: exam.mapel_id, tahun_ajaran_id: exam.tahun_ajaran_id, duration_minutes: exam.duration_minutes, total_questions: exam.total_questions, passing_score: exam.passing_score, status: exam.status }); } }
      if (dialog.kind === "question" && "form" in value) { const form = value.form; const input = { exam_id: String(form.get("exam_id")), subject_name: String(form.get("subject_name")), content_text: String(form.get("content_text")), question_type: String(form.get("question_type")) as AdminQuestion["question_type"], presentation_type: String(form.get("presentation_type")) as AdminQuestion["presentation_type"], group_code: String(form.get("group_code")), stimulus_text: String(form.get("stimulus_text")), question_image_url: String(form.get("question_image_url")), stimulus_image_url: String(form.get("stimulus_image_url")), category_labels: JSON.parse(String(form.get("category_labels_json"))) as string[], options: JSON.parse(String(form.get("options_json"))) as { key: string; content: string; image_url?: string }[], correct_answer: String(form.get("correct_answer")), score_weight: Number(form.get("score_weight")), explanation_text: String(form.get("explanation_text")), status: String(form.get("status")) as Status }; if (dialog.item) await api.updateTeacherQuestion(dialog.item.id, input); else await api.createTeacherQuestion(input); }
    }, successBox); }} />}
    {payoutModal && data && <PayoutRequestModal available={data.commission_summary.available} minimum={data.finance_settings.minimum_payout} saving={saving === "payout-request"} onClose={() => setPayoutModal(false)} onSubmit={(amount) => { setPayoutModal(false); void requestPayout(amount); }} />}
    {toast && <div className="success-toast" role="status"><span>✓</span><p>{toast}</p><button type="button" aria-label="Tutup notifikasi" onClick={() => setToast("")}>×</button></div>}
    {removeTarget && <ConfirmModal title={removeTarget.kind === "package" ? "Hapus paket soal" : "Hapus soal"} message={removeTarget.kind === "package" ? `Paket soal "${removeTarget.label}" beserta seluruh ujian dan soal di dalamnya akan dihapus permanen.${removeTarget.isCBT ? " Untuk paket CBT, data riwayat pengerjaan siswa juga akan dihapus." : ""} Tindakan ini tidak dapat dibatalkan.` : `Soal "${removeTarget.label}" akan dihapus permanen dari bank soal.`} busy={saving === "mutation"} onClose={() => setRemoveTarget(null)} onConfirm={() => void confirmRemove()} />}
    {bulkDeleteTarget && <ConfirmModal title={bulkDeleteTarget.mode === "all" ? "Hapus semua soal" : "Hapus soal terpilih"} message={bulkDeleteTarget.mode === "all" ? `Seluruh ${bulkDeleteTarget.count} soal pada paket "${selectedPackage?.title ?? ""}" akan dihapus permanen. Tindakan ini tidak dapat dibatalkan.` : `${bulkDeleteTarget.count} soal terpilih akan dihapus permanen dari bank soal. Tindakan ini tidak dapat dibatalkan.`} busy={saving === "mutation"} onClose={() => setBulkDeleteTarget(null)} onConfirm={() => void confirmBulkDeleteQuestions()} />}
  </div></main>;
}

function CrudDialog({ dialog, exams, levels, mapels, academicYears, kategoris, kelas, siteSettings, defaultQuestionSubject, user, busy, onClose, onSubmit }: { dialog: Exclude<Dialog, null>; exams: AdminExam[]; levels: string[]; mapels: MasterItem[]; academicYears: MasterItem[]; kategoris: MasterItem[]; kelas: MasterItem[]; siteSettings:SiteSettings|null; defaultQuestionSubject?: string; user: User; busy: boolean; onClose: () => void; onSubmit: (value: DialogSubmit) => void }) {
  const [options, setOptions] = useState<{ key: string; content: string }[]>(dialog.kind === "question" ? (dialog.item?.options ?? [{ key: "A", content: "" }, { key: "B", content: "" }]) : []);
  const [pkgExamType, setPkgExamType] = useState<AdminPackage["exam_type"]>(dialog.kind === "package" ? (dialog.item?.exam_type ?? dialog.defaultExamType ?? "sell") : "sell");
  const [cbtToken, setCbtToken] = useState<string>(dialog.kind === "package" ? (dialog.item?.cbt_token ?? "") : "");
  const [pkgPrice, setPkgPrice] = useState<string>(dialog.kind === "package" ? String(dialog.item?.price ?? 0) : "0");
  const [pkgStartDate, setPkgStartDate] = useState<string>(dialog.kind === "package" ? (dialog.item?.start_date?.slice(0, 10) ?? "") : "");
  const [pkgEndDate, setPkgEndDate] = useState<string>(dialog.kind === "package" ? (dialog.item?.end_date?.slice(0, 10) ?? "") : "");
  function generateCBTToken() { const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"; let out = ""; for (let index = 0; index < 6; index++) out += alphabet[Math.floor(Math.random() * alphabet.length)]; return out; }
  const bundleRef = useRef<{ getValues: () => ExamEntry }>(null);
  const questionExam = dialog.kind === "question" ? exams.find((item) => item.id === dialog.item?.exam_id) ?? exams[0] : undefined;
  const questionSubject = dialog.kind === "question" ? mapels.find((item) => item.id === questionExam?.mapel_id)?.nama ?? dialog.item?.subject_name ?? defaultQuestionSubject ?? "" : "";
  const missingQuestionContext = dialog.kind === "question" && (!questionExam || !questionSubject);
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    if (dialog.kind === "question") form.set("options_json", JSON.stringify(options));
    if (dialog.kind === "package" && bundleRef.current) { const exam = bundleRef.current.getValues(); const isCbt = pkgExamType === "cbt"; const pkg = { title: String(form.get("title")), kode: isCbt ? "" : String(form.get("kode") ?? ""), description: String(form.get("description")), price: isCbt ? 0 : Number(form.get("price")), validity_days: isCbt ? 0 : Number(form.get("validity_days")), status: isCbt ? "active" : (String(form.get("status") ?? "inactive") as Status), jenjang: String(form.get("jenjang")) as User["school_level"], exam_type: String(form.get("exam_type")) as AdminPackage["exam_type"], cbt_token: String(form.get("cbt_token") ?? ""), start_date: isCbt && pkgStartDate ? new Date(pkgStartDate + "T00:00:00Z").toISOString() : undefined, end_date: isCbt && pkgEndDate ? new Date(pkgEndDate + "T23:59:59Z").toISOString() : undefined, kategori_id: String(form.get("kategori_id")), kelas_id: String(form.get("kelas_id")) }; if (!dialog.item) onSubmit({ bundle: { package: pkg, exam: { ...exam, title: String(form.get("jenjang")) } } }); else onSubmit({ update: { form, exam } }); }
    else onSubmit({ form });
  }
  return <div className="modal-backdrop"><form className={`card admin-form ${dialog.kind === "question" ? "question-dialog" : ""}`} onSubmit={submit}><button type="button" className="modal-close" onClick={onClose}>x</button><h2>{"item" in dialog && dialog.item ? "Edit" : "Tambah"} {dialog.kind === "package" ? "Paket Soal" : "Soal"}</h2>
    {dialog.kind === "package" && <><label>Nama paket</label><input name="title" defaultValue={dialog.item?.title} required /><label>Jenis ujian</label><select name="exam_type" value={pkgExamType} onChange={(event) => { const next = event.target.value as AdminPackage["exam_type"]; setPkgExamType(next); if (next === "cbt" && !cbtToken) setCbtToken(generateCBTToken()); }} required><option value="sell">Sell</option><option value="cbt">CBT</option></select>{pkgExamType === "cbt" && <div className="form-grid"><div><label>Kode rahasia (token CBT)</label><input name="cbt_token" value={cbtToken} onChange={(event) => setCbtToken(event.target.value.toUpperCase())} maxLength={8} placeholder="cth: 9K4PT2" required /><p className="form-helper">Siswa wajib memasukkan kode ini saat memulai ujian CBT. Harga otomatis menjadi Gratis (Rp 0).</p></div><div><label>Acak token</label><button type="button" className="button" onClick={() => setCbtToken(generateCBTToken())}>Generate ulang</button></div></div>}<label>Kategori tes</label><select name="kategori_id" defaultValue={dialog.item?.kategori_id ?? kategoris[0]?.id ?? ""} required>{kategoris.map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}</select>{!kategoris.length && <p className="form-helper">Belum ada kategori. Operator perlu menambahkan lewat Master Data.</p>}<label>Keterangan kelas</label><select name="kelas_id" defaultValue={dialog.item?.kelas_id ?? kelas[0]?.id ?? ""} required>{kelas.map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}</select>{!kelas.length && <p className="form-helper">Belum ada kelas. Operator perlu menambahkan lewat Master Data.</p>}{pkgExamType === "cbt" && <div className="form-grid"><div><label>Tanggal mulai aktif</label><input name="start_date" type="date" value={pkgStartDate} onChange={(event) => setPkgStartDate(event.target.value)} required /><p className="form-helper">Ujian CBT tersedia mulai tanggal ini.</p></div><div><label>Tanggal berakhir aktif</label><input name="end_date" type="date" value={pkgEndDate} onChange={(event) => setPkgEndDate(event.target.value)} min={pkgStartDate || undefined} required /><p className="form-helper">Setelah tanggal ini paket otomatis nonaktif.</p></div></div>}<div className="form-grid">{pkgExamType !== "cbt" && <div><label>Kode soal</label><input name="kode" defaultValue={dialog.item?.kode} placeholder="cth: TKA-2026-SMA" maxLength={50} required /></div>}<div><label>Jenjang</label><select name="jenjang" defaultValue={dialog.item?.jenjang ?? "SMA"} required>{levels.map((level) => <option key={level} value={level}>{level}</option>)}</select></div></div><label>Pembuat paket soal</label><input value={user.email} disabled /><label>Deskripsi</label><textarea name="description" defaultValue={dialog.item?.description} />{pkgExamType === "cbt" && <p className="form-helper">Harga otomatis Gratis (Rp 0) dan langsung dipublikasikan aktif tanpa persetujuan. Masa aktif dihitung dari rentang tanggal di atas.</p>}{pkgExamType !== "cbt" && <div className="form-grid"><div><label>Harga</label><div className="price-input"><span>Rp</span><input name="price" type="number" min="0" step="1000" value={pkgPrice} onChange={(event) => { const value = event.target.value; setPkgPrice(value); event.target.setCustomValidity(value !== "" && Number(value) > 0 && Number(value) % 1000 !== 0 ? "Harga harus kelipatan 1.000" : ""); }} required /></div></div><div><label>Masa aktif (hari)</label><input name="validity_days" type="number" min="1" defaultValue={dialog.item?.validity_days ?? siteSettings?.default_package_validity_days ?? 7} required /></div></div>}{pkgExamType === "cbt" && <input type="hidden" name="status" value="active" />}{pkgExamType !== "cbt" && <input type="hidden" name="status" value="inactive" />}{pkgExamType !== "cbt" && <p className="form-helper">Paket masuk ke antrean verifikasi admin. Honorarium upload dicatat otomatis setelah paket disetujui dan dipublikasikan.</p>}<hr /><PackageExamFields ref={bundleRef} mapels={mapels} academicYears={academicYears} defaults={siteSettings?{duration_minutes:siteSettings.default_exam_duration_minutes,total_questions:siteSettings.default_exam_total_questions,passing_score:siteSettings.default_passing_score}:undefined} defaultExam={dialog.item ? exams.find((item) => item.package_id === dialog.item?.id) : undefined} /></>}
    {dialog.kind === "question" && <><input type="hidden" name="exam_id" value={questionExam?.id ?? ""} readOnly /><input type="hidden" name="subject_name" value={questionSubject} readOnly />{missingQuestionContext && <p className="error">Lengkapi mata pelajaran melalui Edit paket sebelum menambah soal.</p>}<TKAQuestionFields item={dialog.item} options={options} setOptions={setOptions}/><div className="form-grid"><div><label>Bobot soal</label><input name="score_weight" type="number" min="0.001" step="0.001" defaultValue={dialog.item?.score_weight ?? 1} required /></div><div><label>Status</label><select name="status" defaultValue={dialog.item?.status ?? "active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></div></div><label>Pembahasan</label><textarea name="explanation_text" defaultValue={dialog.item?.explanation_text} /></>}
    <button className="button full" disabled={busy || missingQuestionContext} type="submit">{busy ? "Menyimpan..." : "Simpan"}</button>
  </form></div>;
}
function ViewHeader({ title, onAdd, templates = false }: { title: string; onAdd: () => void; templates?: boolean }) { return <div className="admin-view-header"><h2>{title}</h2><div className="template-actions">{templates && <><a className="button secondary" href="/templates/template-bank-soal-tka.docx" download>Template Word</a><a className="button secondary" href="/templates/template-bank-soal-tka.xlsx" download>Template Excel</a></>}<button className="button" onClick={onAdd}>+ Tambah</button></div></div>; }
function questionTypeLabel(type: AdminQuestion["question_type"]) { return type === "multiple_choice" ? "PGK MCMA" : type === "category" ? "PGK Kategori" : type === "essay" ? "Esai" : "PG Sederhana"; }
function Stat({ label, value, money: isMoney }: { label: string; value: number; money?: boolean }) { return <article className="card admin-stat"><span>{label}</span><strong>{isMoney ? money.format(value) : value}</strong></article>; }
function CommissionMetric({label,value}:{label:string;value:number}) { return <article className="card finance-metric"><span>{label}</span><strong>{money.format(value)}</strong></article>; }
function PayoutStatus({status}:{status:string}) { const labels:Record<string,string>={pending:"Ditahan",available:"Tersedia",submitted:"Menunggu pemeriksaan",approved:"Disetujui",paid:"Berhasil dibayar",rejected:"Ditolak",cancelled:"Dibatalkan"}; return <span className={`request-status ${status}`}>{labels[status]??status}</span>; }
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
