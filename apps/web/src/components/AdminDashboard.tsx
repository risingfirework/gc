"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, AdminDashboardData, AdminExam, AdminPackage, AdminQuestion, ExamEntry, GlobalRanking, MasterCategory, MasterItem, PackageBundleCreate, SiteSettings, Status, Testimonial, User, UserFinanceProfile } from "@/services/api";
import { PackageExamFields } from "./PackageExamFields";
import ProfileEditor from "./ProfileEditor";
import TKAQuestionFields from "./TKAQuestionFields";
import FinancePanel from "./FinancePanel";
import LogoutButton from "./LogoutButton";
import CatalogStats from "./CatalogStats";
import SiteSettingsPanel from "./SiteSettingsPanel";
import BrandLogo from "./BrandLogo";
import QuestionBankTools from "./QuestionBankTools";

type AdminView = "ringkasan" | "users" | "katalog" | "finance" | "transaksi" | "ranking" | "master" | "testimoni" | "pengaturan" | "profil";
type Dialog = { kind: "user"; item?: User } | { kind: "package"; item?: AdminPackage } | { kind: "question"; item?: AdminQuestion } | { kind: "master"; category: MasterCategory; item?: MasterItem } | null;
type DialogSubmit = { form: FormData } | { bundle: PackageBundleCreate } | { update: { form: FormData; exam: ExamEntry } };
const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
function formatDuration(seconds: number) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
}

const masterCategories: MasterCategory[] = ["mapel", "jenjang", "tahun-ajaran"];
const masterLabel = (category: MasterCategory) => category === "mapel" ? "Mapel" : category === "jenjang" ? "Jenjang" : "Tahun Ajaran";

const statusLabel = (status: Status) => (status === "active" ? "Aktif" : "Nonaktif");
const userRoleLabel = (role: User["role"]) => role === "student" ? "Siswa" : role === "teacher" ? "Guru" : "Admin";
const userInitials = (item: User) => {
  const source = item.name?.trim() || item.email.split("@")[0];
  return source.split(/[\s._-]+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
};
function StatusPill({ status }: { status: Status }) {
  return <span className={`status-pill ${status}`}>{statusLabel(status)}</span>;
}
function PackageStatusPill({ status }: { status: Status }) {
  return <span className={`status-pill ${status}`}>{status === "active" ? "Publish" : "Menunggu verifikasi"}</span>;
}
function QuickFilterIcon({ kind }: { kind: "all" | "teacher" | "SD" | "SMP" | "SMA" }) {
  if (kind === "teacher") return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 19.5V5.8C4 4.8 4.8 4 5.8 4H19v13H5.8A1.8 1.8 0 0 0 4 18.8m0 .7c0 1 .8 1.8 1.8 1.8H20V7M8 8h7M8 11h5"/></svg>;
  if (kind === "all") return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M16 20v-1.5a3.5 3.5 0 0 0-3.5-3.5h-5A3.5 3.5 0 0 0 4 18.5V20m6-9a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Zm7-1a3 3 0 0 1 0 5.8m3 4.2v-1.5a3.5 3.5 0 0 0-2.5-3.35"/></svg>;
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 10 9-6 9 6-9 6-9-6Zm4 2.7V17c2.8 2 7.2 2 10 0v-4.3M21 10v6"/></svg>;
}

export default function AdminDashboard({ user, onLogout, loggingOut, onUserUpdate }: { user: User; onLogout: () => void; loggingOut: boolean; onUserUpdate?: (u: User) => void }) {
  const [view, setView] = useState<AdminView>("ringkasan");
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);
  const [data, setData] = useState<AdminDashboardData | null>(null);
  const [userFinance, setUserFinance] = useState<UserFinanceProfile[]>([]);
  const [ranking, setRanking] = useState<GlobalRanking[]>([]);
  const [rankingLevel, setRankingLevel] = useState<"semua" | "SD" | "SMP" | "SMA">("semua");
  const [dialog, setDialog] = useState<Dialog>(null);
  const [masterTab, setMasterTab] = useState<MasterCategory>("mapel");
  const [catalogTab, setCatalogTab] = useState<string>("semua");
  const [catalogStatusTab, setCatalogStatusTab] = useState<Status>("active");
  const [catalogQuery, setCatalogQuery] = useState("");
  const [userQuery, setUserQuery] = useState("");
  const [userRoleFilter, setUserRoleFilter] = useState<"all" | User["role"]>("all");
  const [userLevelFilter, setUserLevelFilter] = useState<"all" | User["school_level"]>("all");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState("");
  const [testimonials, setTestimonials] = useState<Testimonial[]>([]);
  const [testimonialFilter, setTestimonialFilter] = useState<"all" | "pending" | "approved" | "rejected">("all");
  const [siteSettings,setSiteSettings]=useState<SiteSettings|null>(null);

  const reload = useCallback(async () => {
    const [dashboard, ranks, finance] = await Promise.all([api.adminDashboard(), api.globalRanking(rankingLevel === "semua" ? undefined : rankingLevel), api.adminFinance()]);
    setData(dashboard);
    setRanking(ranks);
    setUserFinance(finance.users);
  }, [rankingLevel]);
  useEffect(() => {
    Promise.all([api.adminDashboard(), api.adminFinance(),api.siteSettings()]).then(([dashboard, finance,settings]) => { setData(dashboard); setUserFinance(finance.users); setSiteSettings(settings); }).catch((reason) => setError(reason instanceof Error ? reason.message : "Data admin gagal dimuat."));
  }, []);
  useEffect(() => {
    api.globalRanking(rankingLevel === "semua" ? undefined : rankingLevel).then(setRanking).catch((reason) => setError(reason instanceof Error ? reason.message : "Ranking gagal dimuat."));
  }, [rankingLevel]);

  async function action(task: () => Promise<unknown>) {
    setSaving("mutation"); setError("");
    try { await task(); await reload(); setDialog(null); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Operasi gagal."); }
    finally { setSaving(""); }
  }
  async function remove(kind: "user"|"package"|"question", id: string, label: string) {
    if (!window.confirm(`Hapus ${label}? Tindakan ini tidak dapat dibatalkan.`)) return;
    await action(() => kind === "user" ? api.deleteAdminUser(id) : kind === "package" ? api.deleteAdminPackage(id) : api.deleteAdminQuestion(id));
  }
  async function toggleUserStatus(target: User) {
    const profile = userFinance.find((item) => item.user_id === target.id);
    if (!profile) { setError("Status pengguna belum tersedia. Muat ulang halaman lalu coba lagi."); return; }
    const nextStatus = profile.account_status === "active" ? "hold" : "active";
    if (nextStatus === "hold" && !window.confirm(`Nonaktifkan akun ${target.email}? Pengguna tidak dapat melakukan transaksi sampai akun diaktifkan kembali.`)) return;
    await action(() => api.updateUserFinance(target.id, {
      commission_percent: profile.commission_percent ?? null,
      discount_percent: profile.discount_percent ?? null,
      account_status: nextStatus,
      notes: profile.notes,
    }));
  }
  async function togglePackageStatus(item: AdminPackage) {
    await action(() => api.updateAdminPackage(item.id, { title: item.title, kode: item.kode, description: item.description, price: item.price, validity_days: item.validity_days, status: item.status === "active" ? "inactive" : "active", jenjang: item.jenjang }));
  }
  async function toggleQuestionStatus(item: AdminQuestion) {
    await action(() => api.updateAdminQuestion(item.id, { exam_id: item.exam_id, subject_name: item.subject_name, content_text: item.content_text, question_type: item.question_type, presentation_type: "single", group_code: "", stimulus_text: "", question_image_url: item.question_image_url, stimulus_image_url: "", category_labels: item.category_labels, options: item.options, correct_answer: item.correct_answer, score_weight: item.score_weight, explanation_text: item.explanation_text, status: item.status === "active" ? "inactive" : "active" }));
  }
  async function removeMaster(category: MasterCategory, id: string, nama: string) {
    if (!window.confirm(`Hapus ${masterLabel(category)} "${nama}"? Tindakan ini tidak dapat dibatalkan.`)) return;
    await action(() => api.deleteMaster(category, id));
  }
  async function setTestimonialStatus(item: Testimonial, status: "approved" | "rejected") {
    await action(async () => { await api.adminUpdateTestimonialStatus(item.id, status); await loadTestimonials(); });
  }
  async function removeTestimonial(item: Testimonial) {
    if (!window.confirm(`Hapus testimoni dari ${item.user_email}? Tindakan ini tidak dapat dibatalkan.`)) return;
    await action(async () => { await api.adminDeleteTestimonial(item.id); await loadTestimonials(); });
  }
  const loadTestimonials = useCallback(async () => {
    try { setTestimonials(await api.adminListTestimonials()); } catch (reason) { setError(reason instanceof Error ? reason.message : "Testimoni gagal dimuat."); }
  }, []);
  useEffect(() => { void loadTestimonials(); }, [loadTestimonials]);

  const menus: Array<[AdminView, string]> = [["ringkasan","Ringkasan"],["users","Kelola User"],["katalog","Katalog Paket Soal"],["finance","Finance"],["master","Master Data"],["transaksi","Transaksi"],["ranking","Ranking"],["testimoni","Testimoni"],["pengaturan","Pengaturan"],["profil","Profil"]];
  const masterItems = (category: MasterCategory): MasterItem[] => category === "mapel" ? data?.mapels ?? [] : category === "jenjang" ? data?.jenjangs ?? [] : data?.academic_years ?? [];
  const allPackages = data?.packages ?? [];
  const publishedCount = allPackages.filter((item) => item.status === "active").length;
  const publishedTeacherCount = allPackages.filter((item) => item.status === "active" && item.publisher_email).length;
  const waitingCount = allPackages.filter((item) => item.status === "inactive").length;
  const waitingTeacherCount = allPackages.filter((item) => item.status === "inactive" && item.publisher_email).length;
  const q = catalogQuery.trim().toLowerCase();
  const visiblePackages = allPackages.filter((item) => item.status === catalogStatusTab && (catalogTab === "semua" || item.jenjang === catalogTab) && (!q || item.kode.toLowerCase().includes(q) || item.title.toLowerCase().includes(q) || (item.publisher_email ?? "").toLowerCase().includes(q)));
  const packages = allPackages;
  const selectedPackage = packages.find((item) => item.id === selectedPackageId) ?? null;
  const packageExams = data?.exams.filter((item) => item.package_id === selectedPackageId) ?? [];
  const packageExamIds = new Set(packageExams.map((item) => item.id));
  const packageQuestions = data?.questions.filter((item) => packageExamIds.has(item.exam_id)) ?? [];
  const visibleUsers = useMemo(() => {
    const query = userQuery.trim().toLowerCase();
    return (data?.users ?? []).filter((item) => {
      const matchesQuery = !query || item.email.toLowerCase().includes(query) || (item.name ?? "").toLowerCase().includes(query);
      const matchesRole = userRoleFilter === "all" || item.role === userRoleFilter;
      const matchesLevel = userLevelFilter === "all" || item.school_level === userLevelFilter;
      return matchesQuery && matchesRole && matchesLevel;
    });
  }, [data?.users, userLevelFilter, userQuery, userRoleFilter]);
  const userCounts = useMemo(() => ({
    all: data?.users.length ?? 0,
    student: data?.users.filter((item) => item.role === "student").length ?? 0,
    teacher: data?.users.filter((item) => item.role === "teacher").length ?? 0,
    admin: data?.users.filter((item) => item.role === "admin").length ?? 0,
  }), [data?.users]);
  const userFinanceById = useMemo(() => new Map(userFinance.map((item) => [item.user_id, item])), [userFinance]);
  const selectQuickUserFilter = (filter: "all" | "teacher" | User["school_level"]) => {
    if (filter === "all") { setUserRoleFilter("all"); setUserLevelFilter("all"); return; }
    if (filter === "teacher") { setUserRoleFilter("teacher"); setUserLevelFilter("all"); return; }
    setUserRoleFilter("student"); setUserLevelFilter(filter);
  };
  const quickFilterActive = (filter: "all" | "teacher" | User["school_level"]) => filter === "all"
    ? userRoleFilter === "all" && userLevelFilter === "all"
    : filter === "teacher"
      ? userRoleFilter === "teacher" && userLevelFilter === "all"
      : userRoleFilter === "student" && userLevelFilter === filter;

  return <main><div className="admin-layout">
    <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/><span>{siteSettings?.platform_name??"TKA Juara"}<small>Administrator</small></span></Link><nav>{menus.map(([key,label]) => <button key={key} className={view===key?"active":""} onClick={()=>setView(key)}>{label}</button>)}</nav><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout}/></aside>
    <section className="admin-main">
      <header className="admin-topbar"><div><p className="eyebrow">Panel administrator</p><h1>{menus.find(([key])=>key===view)?.[1]}</h1></div><button className="admin-identity" onClick={()=>setView("profil")}><span>{(user.name?.trim()||user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim()||"Nama belum dilengkapi"}</strong><small>{user.email} · Administrator</small></div></button></header>
      {error && <p className="error">{error}</p>}{!data && !error && <div className="card">Memuat data administrasi...</div>}

      {data && view==="ringkasan" && <div className="admin-view"><div className="admin-stats"><Stat label="Total User" value={data.overview.total_users}/><Stat label="Siswa" value={data.overview.total_students}/><Stat label="Paket" value={data.overview.total_packages}/><Stat label="Ujian" value={data.overview.total_exams}/><Stat label="Transaksi" value={data.overview.total_transactions}/><Stat label="Pembayaran Berhasil" value={data.overview.paid_transactions}/></div></div>}

      {data && view==="users" && <section className="admin-view user-management">
        <div className="user-management-heading"><div><p className="eyebrow">Manajemen akses</p><h2>Data pengguna</h2><p className="muted">Kelola akun, peran, dan jenjang pengguna dalam satu tempat.</p></div><button className="button user-add-button" onClick={()=>setDialog({kind:"user"})}><span aria-hidden="true">+</span> Tambah pengguna</button></div>
        <div className="user-summary-grid">
          <article className="user-summary-card total"><span className="user-summary-icon" aria-hidden="true">U</span><div><small>Total pengguna</small><strong>{userCounts.all}</strong></div></article>
          <article className="user-summary-card student"><span className="user-summary-icon" aria-hidden="true">S</span><div><small>Siswa</small><strong>{userCounts.student}</strong></div></article>
          <article className="user-summary-card teacher"><span className="user-summary-icon" aria-hidden="true">G</span><div><small>Guru</small><strong>{userCounts.teacher}</strong></div></article>
          <article className="user-summary-card admin"><span className="user-summary-icon" aria-hidden="true">A</span><div><small>Administrator</small><strong>{userCounts.admin}</strong></div></article>
        </div>
        <div className="card user-directory">
          <div className="user-toolbar">
            <div className="user-search"><span aria-hidden="true">⌕</span><input type="search" aria-label="Cari pengguna" placeholder="Cari nama atau email..." value={userQuery} onChange={(event)=>setUserQuery(event.target.value)}/></div>
            <button className="user-search-button" type="button">Cari pengguna</button>
          </div>
          <div className="user-quick-filters" aria-label="Filter cepat pengguna">{([
            ["all","Semua pengguna","Semua peran"],
            ["teacher","Guru","Akun pengajar"],
            ["SD","SD","Siswa sekolah dasar"],
            ["SMP","SMP","Siswa sekolah menengah pertama"],
            ["SMA","SMK","Siswa SMA / SMK"],
          ] as const).map(([key,label,description])=><button key={key} type="button" className={quickFilterActive(key)?"active":""} aria-pressed={quickFilterActive(key)} onClick={()=>selectQuickUserFilter(key)}><span className="quick-filter-icon"><QuickFilterIcon kind={key}/></span><span><strong>{label}</strong><small>{description}</small></span></button>)}</div>
          <div className="user-results-meta"><span>Menampilkan <strong>{visibleUsers.length}</strong> dari {data.users.length} pengguna</span>{saving && <span className="user-saving">Menyimpan perubahan...</span>}</div>
          <div className="user-table-wrap"><table className="user-table"><thead><tr><th>Pengguna</th><th>Peran</th><th>Jenjang</th><th>Status</th><th>Terdaftar</th><th><span className="sr-only">Aksi</span></th></tr></thead><tbody>{visibleUsers.map((item)=>{
            const isCurrentUser = item.id === user.id;
            const accountActive = userFinanceById.get(item.id)?.account_status !== "hold";
            return <tr key={item.id}><td><div className="user-cell"><span className={`user-avatar ${item.role}`}>{userInitials(item)}</span><div><strong>{item.name?.trim() || item.email.split("@")[0]}</strong><span>{item.email}</span></div>{isCurrentUser&&<span className="current-user-badge">Akun Anda</span>}</div></td><td><span className={`user-readonly-pill role-${item.role}`}><span aria-hidden="true"/>{userRoleLabel(item.role)}</span></td><td><span className="user-readonly-pill level">{item.school_level === "SMA" ? "SMA / SMK" : item.school_level}</span></td><td><span className={`user-account-status ${accountActive?"active":"inactive"}`}><span aria-hidden="true"/>{accountActive?"Aktif":"Nonaktif"}</span></td><td><div className="user-date"><strong>{new Date(item.created_at).toLocaleDateString("id-ID",{day:"2-digit",month:"short",year:"numeric"})}</strong><span>{new Date(item.created_at).toLocaleDateString("id-ID",{weekday:"long"})}</span></div></td><td className="user-actions"><div className="user-row-actions"><button className={`user-status-button ${accountActive?"deactivate":"activate"}`} disabled={isCurrentUser||saving!==""} title={isCurrentUser?"Akun yang sedang digunakan harus tetap aktif":accountActive?"Nonaktifkan akun":"Aktifkan akun"} onClick={()=>void toggleUserStatus(item)}>{accountActive?"Nonaktifkan":"Aktifkan"}</button><button className="user-edit-button" disabled={saving!==""} onClick={()=>setDialog({kind:"user",item})}><span aria-hidden="true">✎</span> Edit</button><button className="user-delete-button" aria-label={`Hapus ${item.email}`} title={isCurrentUser?"Akun yang sedang digunakan tidak dapat dihapus":"Hapus pengguna"} disabled={isCurrentUser||saving!==""} onClick={()=>void remove("user",item.id,item.email)}><span aria-hidden="true">×</span><span>Hapus</span></button></div></td></tr>;
          })}{visibleUsers.length===0&&<tr><td colSpan={6}><div className="user-empty"><span aria-hidden="true">⌕</span><strong>Pengguna tidak ditemukan</strong><p>Ubah kata kunci atau filter untuk melihat hasil lainnya.</p><button onClick={()=>{setUserQuery("");setUserRoleFilter("all");setUserLevelFilter("all");}}>Reset filter</button></div></td></tr>}</tbody></table></div>
        </div>
      </section>}

      {data && view==="katalog" && !selectedPackage && <section className="admin-view catalog-admin">
        <div className="admin-catalog-heading"><div><p className="eyebrow">Katalog paket soal</p><h2>Kelola paket, ujian, dan bank soal</h2><p className="muted">Klik salah satu kartu untuk membuka lingkungan pengelolaan lengkap.</p></div><button className="button" onClick={()=>setDialog({kind:"package"})}>+ Tambah paket</button></div>
        <div className="content-subtabs catalog-status-tabs"><button className={catalogStatusTab==="active"?"active":""} onClick={()=>setCatalogStatusTab("active")}><b>Publish</b><small>{publishedTeacherCount} guru · {publishedCount-publishedTeacherCount} platform</small><span>{publishedCount}</span></button><button className={catalogStatusTab==="inactive"?"active":""} onClick={()=>setCatalogStatusTab("inactive")}><b>Menunggu verifikasi</b><small>{waitingTeacherCount} dari guru</small><span>{waitingCount}</span></button></div>
        <div className="catalog-toolbar"><input className="catalog-search" type="search" placeholder="Cari kode soal, nama paket, atau pembuat..." value={catalogQuery} onChange={(event)=>setCatalogQuery(event.target.value)}/><div className="content-subtabs">{["semua","SD","SMP","SMA"].map((tab)=><button key={tab} className={catalogTab===tab?"active":""} onClick={()=>setCatalogTab(tab)}>{tab==="semua"?"Semua":tab}</button>)}</div></div>
        <div className="admin-package-grid">{visiblePackages.map((item)=><article className={`card admin-package-card ${item.status==="inactive"?"inactive":""}`} key={item.id} onClick={()=>setSelectedPackageId(item.id)}><div className="admin-package-card-top"><PackageStatusPill status={item.status}/><span className="package-code">{item.kode}</span></div><span className="package-jenjang">{item.jenjang}</span><h3>{item.title}</h3><p>{item.description}</p><strong>{item.price===0?"Gratis":money.format(item.price)}</strong><small>{(data.exams.filter((exam)=>exam.package_id===item.id).length)} ujian · {data.questions.filter((q)=>new Set(data.exams.filter((exam)=>exam.package_id===item.id).map((exam)=>exam.id)).has(q.exam_id)).length} soal</small>{item.publisher_email&&<small className="package-publisher">Pembuat: {item.publisher_email}</small>}{item.status==="active"&&<CatalogStats sales={item.sales_count} views={item.view_count}/>}<div><button className="table-action" onClick={(event)=>{event.stopPropagation();setDialog({kind:"package",item})}}>Edit</button><button className="danger-action" onClick={(event)=>{event.stopPropagation();void remove("package",item.id,item.title)}}>Hapus</button></div></article>)}
        {visiblePackages.length===0 && <div className="card empty-state">Tidak ada paket {catalogStatusTab==="active"?"yang sudah dipublikasikan":"yang sedang menunggu verifikasi"} untuk filter ini.</div>}</div>
      </section>}

      {data && view==="katalog" && selectedPackage && <section className="admin-view catalog-detail">
        <div className="catalog-detail-heading"><div><p className="eyebrow">Lingkungan katalog</p><h2>{selectedPackage.title}</h2></div><div className="catalog-detail-actions"><PackageStatusPill status={selectedPackage.status}/><button className="table-action" onClick={()=>setDialog({kind:"package",item:selectedPackage})}>Edit paket</button><button className={`table-action ${selectedPackage.status==="active"?"toggle-off":""}`} disabled={saving!==""} onClick={()=>void togglePackageStatus(selectedPackage)}>{selectedPackage.status==="active"?"Nonaktifkan":selectedPackage.publisher_email?"Verifikasi & publikasikan":"Aktifkan"}</button><button className="button secondary" onClick={()=>setSelectedPackageId(null)}>← Kembali ke katalog</button></div></div>
        <div className="card admin-package-spec"><div className="admin-package-spec-desc"><span>Deskripsi</span><p>{selectedPackage.description || "Tidak ada deskripsi."}</p></div><div className="admin-package-spec-grid"><div><span>Kode soal</span><strong>{selectedPackage.kode}</strong></div><div><span>Jenjang</span><strong>{selectedPackage.jenjang}</strong></div><div><span>Harga</span><strong>{selectedPackage.price===0?"Gratis":money.format(selectedPackage.price)}</strong></div><div><span>Masa aktif</span><strong>{selectedPackage.validity_days} hari</strong></div><div><span>Ujian</span><strong>{packageExams.length}</strong></div><div><span>Total soal</span><strong>{packageQuestions.length}</strong></div><div><span>Dibuat</span><strong>{new Date(selectedPackage.created_at).toLocaleDateString("id-ID")}</strong></div>{selectedPackage.publisher_email&&<div><span>Pembuat</span><strong>{selectedPackage.publisher_email}</strong></div>}</div></div>
        <div className="bank-section"><ViewHeader title="Bank soal" templates onAdd={()=>setDialog({kind:"question"})}/><QuestionBankTools packageTitle={selectedPackage.title} packageCode={selectedPackage.kode} level={selectedPackage.jenjang} durationMinutes={packageExams[0]?.duration_minutes} questions={packageQuestions.filter(item=>item.status==="active")}/><AdminTable headers={["Ujian","Mapel","Bentuk","Soal","Kunci","Bobot","Status","Aksi"]}>{packageQuestions.length?packageQuestions.map((item)=><tr key={item.id}><td>{item.exam_title}</td><td>{item.subject_name}</td><td>{questionTypeLabel(item.question_type)}</td><td>{item.content_text.length>60?`${item.content_text.slice(0,60)}...`:item.content_text}</td><td>{item.correct_answer}</td><td>{item.score_weight}</td><td><StatusPill status={item.status}/></td><td><button className="table-action" onClick={()=>setDialog({kind:"question",item})}>Edit</button><button className={`table-action ${item.status==="active"?"toggle-off":""}`} disabled={saving!==""} onClick={()=>void toggleQuestionStatus(item)}>{item.status==="active"?"Nonaktifkan":"Aktifkan"}</button><button className="danger-action" onClick={()=>void remove("question",item.id,item.subject_name)}>Hapus</button></td></tr>):<tr><td colSpan={8} className="empty-state">Belum ada soal di paket ini.</td></tr>}</AdminTable></div>
      </section>}

      {data && view==="transaksi" && <AdminTable headers={["Invoice","User","Paket","Nominal","Status"]}>{data.transactions.length?data.transactions.map((item)=><tr key={item.id}><td>{item.invoice_number}</td><td>{item.user_email}</td><td>{item.package_title}</td><td>{money.format(item.amount)}</td><td><span className={`status-pill ${item.status}`}>{item.status}</span></td></tr>):<tr><td colSpan={5} className="empty-state">Belum ada transaksi.</td></tr>}</AdminTable>}
      {view==="finance"&&<FinancePanel/>}
      {data && view==="master" && <section className="admin-view master-view">
        <div className="content-subtabs">{masterCategories.map((category)=><button key={category} className={masterTab===category?"active":""} onClick={()=>setMasterTab(category)}>{masterLabel(category)}</button>)}</div>
        <ViewHeader title={`Data ${masterLabel(masterTab)}`} onAdd={()=>setDialog({kind:"master",category:masterTab})}/>
        <AdminTable headers={["Nama","Aksi"]}>{masterItems(masterTab).length?masterItems(masterTab).map((item)=><tr key={item.id}><td>{item.nama}</td><td><button className="table-action" onClick={()=>setDialog({kind:"master",category:masterTab,item})}>Edit</button><button className="danger-action" onClick={()=>void removeMaster(masterTab,item.id,item.nama)}>Hapus</button></td></tr>):<tr><td colSpan={2} className="empty-state">Belum ada data.</td></tr>}</AdminTable>
      </section>}
      {view==="ranking" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Peringkat siswa</p><h2>Ranking berdasarkan jenjang</h2></div><span className="muted">Skor tertinggi, waktu tercepat, tanggal terawal</span></div><div className="content-subtabs">{(["semua","SD","SMP","SMA"] as const).map((tab)=><button key={tab} className={rankingLevel===tab?"active":""} onClick={()=>setRankingLevel(tab)}>{tab==="semua"?"Semua":tab}</button>)}</div><AdminTable headers={["Peringkat","Pengguna","Jenjang","Nilai","Waktu","Tanggal"]}>{ranking.map((item)=><tr key={`${item.rank}-${item.display_name}`}><td>#{item.rank}</td><td>{item.display_name}</td><td>{item.school_level}</td><td><strong>{item.score.toFixed(2)}</strong></td><td>{formatDuration(item.duration_seconds)}</td><td>{new Date(item.finished_at).toLocaleDateString("id-ID",{day:"2-digit",month:"short",year:"numeric"})}</td></tr>)}</AdminTable></section>}
      {view==="testimoni" && <section className="admin-view">
        <div className="admin-view-header"><div><p className="eyebrow">Testimoni siswa</p><h2>Kelola persetujuan testimoni</h2></div></div>
        <div className="content-subtabs">{(["all","pending","approved","rejected"] as const).map((tab)=><button key={tab} className={testimonialFilter===tab?"active":""} onClick={()=>setTestimonialFilter(tab)}>{tab==="all"?"Semua":tab==="pending"?"Menunggu":tab==="approved"?"Disetujui":"Ditolak"}</button>)}</div>
        <AdminTable headers={["Pengguna","Testimoni","Status","Tanggal","Aksi"]}>{testimonials.filter((item)=>testimonialFilter==="all"||item.status===testimonialFilter).length?testimonials.filter((item)=>testimonialFilter==="all"||item.status===testimonialFilter).map((item)=><tr key={item.id}><td><div>{item.user_name || item.user_email}<small>{item.user_email}</small></div></td><td style={{maxWidth:420}}>{item.quote}</td><td><span className={`status-pill ${item.status}`}>{item.status==="approved"?"Disetujui":item.status==="rejected"?"Ditolak":"Menunggu"}</span></td><td>{new Date(item.created_at).toLocaleDateString("id-ID")}</td><td>{item.status!=="approved"&&<button className="table-action" disabled={saving!==""} onClick={()=>void setTestimonialStatus(item,"approved")}>Setujui</button>}{item.status!=="rejected"&&<button className="table-action" disabled={saving!==""} onClick={()=>void setTestimonialStatus(item,"rejected")}>Tolak</button>}<button className="danger-action" disabled={saving!==""} onClick={()=>void removeTestimonial(item)}>Hapus</button></td></tr>):<tr><td colSpan={5} className="empty-state">Tidak ada testimoni pada kategori ini.</td></tr>}</AdminTable>
      </section>}
      {view==="pengaturan" && siteSettings && <SiteSettingsPanel initial={siteSettings} onSaved={setSiteSettings}/>}
      {view==="profil" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div><ProfileEditor user={user} onUpdated={onUserUpdate} /></section>}
    </section>
    {dialog && <CrudDialog dialog={dialog} exams={selectedPackageId ? packageExams : data?.exams??[]} levels={data?.levels??[]} mapels={data?.mapels??[]} academicYears={data?.academic_years??[]} siteSettings={siteSettings} defaultQuestionSubject={packageQuestions[0]?.subject_name} user={user} busy={saving!==""} onClose={()=>setDialog(null)} onSubmit={(value)=>void action(async()=>{
      if(dialog.kind==="user"&&"form" in value){const form=value.form;const role=String(form.get("role")) as User["role"];const schoolLevel=String(form.get("school_level")) as User["school_level"];if(dialog.item)await api.updateAdminUser(dialog.item.id,role,schoolLevel);else await api.createAdminUser({email:String(form.get("email")),password:String(form.get("password")),role,school_level:schoolLevel})}
      if(dialog.kind==="package"){if(!dialog.item&&"bundle" in value){const input=value.bundle;await api.createAdminPackageBundle(input)}else if("update" in value&&dialog.item){const {form,exam}=value.update;await api.updateAdminPackage(dialog.item.id,{title:String(form.get("title")),kode:String(form.get("kode")),description:String(form.get("description")),price:Number(form.get("price")),validity_days:Number(form.get("validity_days")),status:String(form.get("status")) as Status,jenjang:String(form.get("jenjang")) as User["school_level"]});const examTarget=data?.exams.find((item)=>item.package_id===dialog.item!.id);if(examTarget)await api.updateAdminExam(examTarget.id,{package_id:examTarget.package_id,title:exam.title,mapel_id:exam.mapel_id,tahun_ajaran_id:exam.tahun_ajaran_id,duration_minutes:exam.duration_minutes,total_questions:exam.total_questions,passing_score:exam.passing_score,status:exam.status})}}
      if(dialog.kind==="question"&&"form" in value){const form=value.form;const input={exam_id:String(form.get("exam_id")),subject_name:String(form.get("subject_name")),content_text:String(form.get("content_text")),question_type:String(form.get("question_type")) as AdminQuestion["question_type"],presentation_type:String(form.get("presentation_type")) as AdminQuestion["presentation_type"],group_code:String(form.get("group_code")),stimulus_text:String(form.get("stimulus_text")),question_image_url:String(form.get("question_image_url")),stimulus_image_url:String(form.get("stimulus_image_url")),category_labels:JSON.parse(String(form.get("category_labels_json"))) as string[],options:JSON.parse(String(form.get("options_json"))) as {key:string;content:string;image_url?:string}[],correct_answer:String(form.get("correct_answer")),score_weight:Number(form.get("score_weight")),explanation_text:String(form.get("explanation_text")),status:String(form.get("status")) as Status};if(dialog.item)await api.updateAdminQuestion(dialog.item.id,input);else await api.createAdminQuestion(input)}
      if(dialog.kind==="master"&&"form" in value){const nama=String(value.form.get("nama"));if(dialog.item)await api.updateMaster(dialog.category,dialog.item.id,nama);else await api.createMaster(dialog.category,nama)}
    })}/>}
  </div></main>;
}

function CrudDialog({dialog,exams,levels,mapels,academicYears,siteSettings,defaultQuestionSubject,user,busy,onClose,onSubmit}:{dialog:Exclude<Dialog,null>;exams:AdminExam[];levels:string[];mapels:MasterItem[];academicYears:MasterItem[];siteSettings:SiteSettings|null;defaultQuestionSubject?:string;user:User;busy:boolean;onClose:()=>void;onSubmit:(value:DialogSubmit)=>void}){
  const [options,setOptions] = useState<{key:string;content:string}[]>(dialog.kind==="question"?(dialog.item?.options??[{key:"A",content:""},{key:"B",content:""}]):[]);
  const bundleRef = useRef<{getValues:()=>ExamEntry}>(null);
  const questionExam = dialog.kind==="question" ? exams.find((item)=>item.id===dialog.item?.exam_id)??exams[0] : undefined;
  const questionSubject = dialog.kind==="question" ? mapels.find((item)=>item.id===questionExam?.mapel_id)?.nama??dialog.item?.subject_name??defaultQuestionSubject??"" : "";
  const missingQuestionContext = dialog.kind==="question"&&(!questionExam||!questionSubject);
  function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    if(dialog.kind==="question") form.set("options_json", JSON.stringify(options));
    if(dialog.kind==="package"&&bundleRef.current){const exam=bundleRef.current.getValues();const pkg={title:String(form.get("title")),kode:String(form.get("kode")),description:String(form.get("description")),price:Number(form.get("price")),validity_days:Number(form.get("validity_days")),status:String(form.get("status")) as Status,jenjang:String(form.get("jenjang")) as User["school_level"]};if(!dialog.item)onSubmit({bundle:{package:pkg,exam:{...exam,title:String(form.get("jenjang"))}}});else onSubmit({update:{form,exam}})}
    else onSubmit({form});
  }
  return <div className="modal-backdrop"><form className="card admin-form" onSubmit={submit}><button type="button" className="modal-close" onClick={onClose}>x</button><h2>{"item" in dialog&&dialog.item?"Edit":"Tambah"} {dialog.kind==="user"?"User":dialog.kind==="package"?"Paket Soal":dialog.kind==="master"?masterLabel(dialog.category):"Soal"}</h2>
    {dialog.kind==="user"&&<><label>Email</label><input name="email" type="email" defaultValue={dialog.item?.email} disabled={Boolean(dialog.item)} required/>{!dialog.item&&<><label>Password</label><input name="password" type="password" minLength={8} required/></>}<label>Peran</label><select name="role" defaultValue={dialog.item?.role??"student"} disabled={dialog.item?.id===user.id}><option value="student">Siswa</option><option value="teacher">Guru</option><option value="admin">Admin</option></select>{dialog.item?.id===user.id&&<input type="hidden" name="role" value={dialog.item.role}/>}<label>Jenjang</label><select name="school_level" defaultValue={dialog.item?.school_level??"SMA"}><option>SD</option><option>SMP</option><option value="SMA">SMA / SMK</option></select>{dialog.item&&<p className="form-helper">Email tidak dapat diubah. Gunakan formulir ini untuk memperbarui peran dan jenjang pengguna.</p>}</>}
    {dialog.kind==="package"&&<><label>Nama paket</label><input name="title" defaultValue={dialog.item?.title} required/><div className="form-grid"><div><label>Kode soal</label><input name="kode" defaultValue={dialog.item?.kode} placeholder="cth: TKA-2026-SMA" maxLength={50} required/></div><div><label>Jenjang</label><select name="jenjang" defaultValue={dialog.item?.jenjang??"SMA"} required>{levels.map((level)=><option key={level} value={level}>{level}</option>)}</select></div></div><label>Pembuat paket soal</label><input value={user.email} disabled/><label>Deskripsi</label><textarea name="description" defaultValue={dialog.item?.description}/><div className="form-grid"><div><label>Harga</label><div className="price-input"><span>Rp</span><input name="price" type="number" min="0" step="1000" defaultValue={dialog.item?.price??0} onChange={(event)=>{const value=Number(event.target.value);event.target.setCustomValidity(value>0&&value%1000!==0?"Harga harus kelipatan 1.000":"")}} required/></div></div><div><label>Masa aktif (hari)</label><input name="validity_days" type="number" min="1" defaultValue={dialog.item?.validity_days??siteSettings?.default_package_validity_days??7} required/></div></div><label>Status</label><select name="status" defaultValue={dialog.item?.status??"active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select><hr/><PackageExamFields ref={bundleRef} mapels={mapels} academicYears={academicYears} defaults={siteSettings?{duration_minutes:siteSettings.default_exam_duration_minutes,total_questions:siteSettings.default_exam_total_questions,passing_score:siteSettings.default_passing_score}:undefined} defaultExam={dialog.item?exams.find((item)=>item.package_id===dialog.item?.id):undefined}/></>}
    {dialog.kind==="question"&&<><input type="hidden" name="exam_id" value={questionExam?.id??""} readOnly/><input type="hidden" name="subject_name" value={questionSubject} readOnly/>{missingQuestionContext&&<p className="error">Lengkapi mata pelajaran melalui Edit paket sebelum menambah soal.</p>}<TKAQuestionFields item={dialog.item} options={options} setOptions={setOptions}/><div className="form-grid"><div><label>Bobot soal</label><input name="score_weight" type="number" min="0.001" step="0.001" defaultValue={dialog.item?.score_weight??1} required/></div><div><label>Status</label><select name="status" defaultValue={dialog.item?.status??"active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></div></div><label>Pembahasan</label><textarea name="explanation_text" defaultValue={dialog.item?.explanation_text}/></>}
    {dialog.kind==="master"&&<><label>Nama</label><input name="nama" defaultValue={dialog.item?.nama} required/></>}
    <button className="button full" disabled={busy||missingQuestionContext} type="submit">{busy?"Menyimpan...":"Simpan"}</button>
  </form></div>;
}
function ViewHeader({title,onAdd,templates=false}:{title:string;onAdd:()=>void;templates?:boolean}){return <div className="admin-view-header"><h2>{title}</h2><div className="template-actions">{templates&&<><a className="button secondary" href="/templates/template-bank-soal-tka.docx" download>Template Word</a><a className="button secondary" href="/templates/template-bank-soal-tka.xlsx" download>Template Excel</a></>}<button className="button" onClick={onAdd}>+ Tambah</button></div></div>}
function questionTypeLabel(type:AdminQuestion["question_type"]){return type==="multiple_choice"?"PGK MCMA":type==="category"?"PGK Kategori":"PG Sederhana"}
function Stat({label,value}:{label:string;value:number}) { return <article className="card admin-stat"><span>{label}</span><strong>{value}</strong></article>; }
function AdminTable({headers,children}:{headers:string[];children:React.ReactNode}) { return <div className="card admin-table-card"><div className="ranking-table-wrap"><table className="ranking-table"><thead><tr>{headers.map((header)=><th key={header}>{header}</th>)}</tr></thead><tbody>{children}</tbody></table></div></div>; }
