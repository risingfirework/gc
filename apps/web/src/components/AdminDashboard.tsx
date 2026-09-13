"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { api, AdminDashboardData, AdminExam, AdminPackage, AdminPackagesPage, AdminPage, AdminQuestion, AdminTransaction, AdminUsersPage, AuditLog, CBTPublishSetting, ExamEntry, FinanceOverview, GlobalRanking, MasterCategory, MasterItem, PackageBundleCreate, SiteSettings, Status, TeacherVerification, Testimonial, User, UserFinanceProfile } from "@/services/api";
import { PackageExamFields } from "./PackageExamFields";
import ProfileEditor from "./ProfileEditor";
import TKAQuestionFields from "./TKAQuestionFields";
import FinancePanel from "./FinancePanel";
import LogoutButton from "./LogoutButton";
import NotificationBell from "./NotificationBell";
import CatalogStats from "./CatalogStats";
import SiteSettingsPanel from "./SiteSettingsPanel";
import VideoManagerPanel from "./VideoManagerPanel";
import DataImage from "./DataImage";
import ConfirmModal from "./ConfirmModal";
import CBTSettingsPanel from "./CBTSettingsPanel";
import BrandLogo from "./BrandLogo";
import QuestionBankTools from "./QuestionBankTools";

type AdminView = "ringkasan" | "users" | "katalog" | "cbt" | "cbt-pengaturan" | "finance" | "transaksi" | "ranking" | "master" | "testimoni" | "video" | "pengaturan" | "audit" | "verifikasi" | "profil";
type Dialog = { kind: "user"; item?: User } | { kind: "package"; item?: AdminPackage; defaultExamType?: "sell" | "cbt" } | { kind: "question"; item?: AdminQuestion } | { kind: "master"; category: MasterCategory; item?: MasterItem } | null;
type DialogSubmit = { form: FormData } | { bundle: PackageBundleCreate } | { update: { form: FormData; exam: ExamEntry } };
const money = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
function formatDuration(seconds: number) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
}
type SummaryWidget = "kpis" | "revenue" | "popular" | "recent" | "newusers" | "attention" | "ranking";
const SUMMARY_WIDGETS: Array<{ id: SummaryWidget; label: string; hint: string }> = [
  { id: "kpis", label: "Angka kunci", hint: "Total user, siswa, paket, ujian, dan transaksi." },
  { id: "revenue", label: "Keuangan", hint: "Pendapatan, honorarium, dan antrean pencairan." },
  { id: "popular", label: "Paket terlaris", hint: "Paket dengan penjualan terbanyak." },
  { id: "recent", label: "Transaksi terbaru", hint: "Invoice terakhir yang masuk." },
  { id: "newusers", label: "Pendaftar baru", hint: "Akun pengguna terbaru." },
  { id: "attention", label: "Perlu perhatian", hint: "Verifikasi guru, pencairan, dan paket nonaktif." },
  { id: "ranking", label: "Peringkat teratas", hint: "Siswa terbaik pada ranking global." },
];
function useDebounced<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

const masterCategories: MasterCategory[] = ["mapel", "jenjang", "tahun-ajaran", "kategori", "kelas"];
  const masterLabel = (category: MasterCategory) => category === "mapel" ? "Mapel" : category === "jenjang" ? "Jenjang" : category === "tahun-ajaran" ? "Tahun Ajaran" : category === "kategori" ? "Kategori Tes" : "Kelas";

const statusLabel = (status: Status) => (status === "active" ? "Aktif" : "Nonaktif");
const userRoleLabel = (role: User["role"]) => role === "student" ? "Siswa" : role === "teacher" ? "Guru" : role === "owner" ? "Pemilik" : role === "finance" ? "Finance" : role === "affiliate" ? "Affiliate" : "Operator";
const adminViewLabels: Record<AdminView, string> = { ringkasan:"Ringkasan", users:"Kelola User", katalog:"Katalog Paket Soal", cbt:"Bank Soal CBT", "cbt-pengaturan":"Pengaturan CBT", finance:"Finance", master:"Master Data", transaksi:"Transaksi", ranking:"Ranking", testimoni:"Testimoni", video:"Menu Video", pengaturan:"Pengaturan", audit:"Audit Log", verifikasi:"Verifikasi Guru", profil:"Profil" };
const sidebarGroups: Array<{ name: string; views: AdminView[] }> = [
  { name: "Konten", views: ["katalog", "cbt", "cbt-pengaturan", "master"] },
  { name: "Interaksi", views: ["ranking", "testimoni", "video"] },
  { name: "Pengguna", views: ["users", "verifikasi"] },
  { name: "Keuangan", views: ["finance", "transaksi"] },
  { name: "Sistem", views: ["pengaturan", "audit"] },
];
const groupOfView = (key: AdminView) => sidebarGroups.find((group) => group.views.includes(key))?.name ?? null;
const auditActionLabel = (action: string) => (({ update_finance_settings: "Ubah kebijakan komisi", update_user_finance: "Ubah profil keuangan", payout_paid: "Pencairan dibayar", payout_review: "Tinjau pengajuan pencairan", create_user: "Buat pengguna", update_user: "Ubah pengguna", delete_user: "Hapus pengguna", update_site_settings: "Ubah pengaturan situs", approve_teacher: "Setujui guru", reject_teacher: "Tolak guru", cbt_publish_pembahasan: "Publish pembahasan CBT" }) as Record<string, string>)[action] ?? action;
const formatAuditDetail = (detail: Record<string, unknown>) => {
  const label = (({ commission_percent: "komisi", discount_percent: "diskon", account_status: "status", reference: "referensi", note: "catatan", role: "peran", from_role: "dari", to_role: "ke", school_level: "jenjang", platform_commission_percent: "komisi platform", teacher_sales_bonus_percent: "bonus penjualan", affiliate_rate_percent: "komisi affiliate", referral_code: "kode rujukan", tax_percent: "pajak", minimum_payout: "minimal payout", payout_cycle: "siklus", auto_payout: "auto", amount: "nominal", source: "sumber", email: "email", platform_name: "nama platform" }) as Record<string, string>);
  return Object.entries(detail ?? {}).filter(([, v]) => v !== null && v !== undefined && v !== "").map(([k, v]) => `${(label[k] ?? k)}=${String(v)}`).join(" · ");
};
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
  const role = user.role;
  const allowedViews: AdminView[] = role === "owner"
    ? ["ringkasan", "users", "katalog", "cbt", "cbt-pengaturan", "finance", "master", "transaksi", "ranking", "testimoni", "video", "pengaturan", "audit", "verifikasi", "profil"]
    : role === "finance"
      ? ["finance", "transaksi", "profil"]
      : ["ringkasan", "katalog", "cbt", "cbt-pengaturan", "master", "transaksi", "ranking", "testimoni", "video", "pengaturan", "verifikasi", "profil"];
  const [view, setView] = useState<AdminView>(allowedViews[0]);
  const [openGroups, setOpenGroups] = useState<Set<string>>(new Set());
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);
  const [data, setData] = useState<AdminDashboardData | null>(null);
  const [userFinance, setUserFinance] = useState<UserFinanceProfile[]>([]);
  const [usersPage, setUsersPage] = useState<AdminUsersPage>({ items: [], page: 1, count: 25, total: 0, summary: { total: 0, students: 0, teachers: 0, admins: 0 } });
  const [packagesPage, setPackagesPage] = useState<AdminPackagesPage>({ items: [], page: 1, count: 12, total: 0, counts: { active: 0, inactive: 0, active_teacher: 0, inactive_teacher: 0 } });
  const [pkgExamsPage, setPkgExamsPage] = useState<AdminPage<AdminExam>>({ items: [], page: 1, count: 25, total: 0 });
  const [pkgQuestionsPage, setPkgQuestionsPage] = useState<AdminPage<AdminQuestion>>({ items: [], page: 1, count: 25, total: 0 });
  const [rankingPageData, setRankingPageData] = useState<AdminPage<GlobalRanking>>({ items: [], page: 1, count: 50, total: 0 });
  const [userPage, setUserPage] = useState(1);
  const [packagePage, setPackagePage] = useState(1);
  const [pkgExamPage, setPkgExamPage] = useState(1);
  const [pkgQuestionPage, setPkgQuestionPage] = useState(1);
  const [rankingPage, setRankingPage] = useState(1);
  const [refreshKey, setRefreshKey] = useState(0);
  const [transactions, setTransactions] = useState<{ items: AdminTransaction[]; page: number; count: number; total: number }>({ items: [], page: 1, count: 25, total: 0 });
  const [txnStatus, setTxnStatus] = useState("");
  const [txnPage, setTxnPage] = useState(1);
  const [audit, setAudit] = useState<AuditLog[]>([]);
  const [rankingLevel, setRankingLevel] = useState<"semua" | "SD" | "SMP" | "SMA">("semua");
  const [rankingMode, setRankingMode] = useState<"score" | "activity">("score");
  const [dialog, setDialog] = useState<Dialog>(null);
  const [toast, setToast] = useState("");
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
  const [siteSettings, setSiteSettings] = useState<SiteSettings | null>(null);
  const [verifications, setVerifications] = useState<AdminPage<TeacherVerification>>({ items: [], page: 1, count: 25, total: 0 });
  const [verifPage, setVerifPage] = useState(1);
  const [verifFilter, setVerifFilter] = useState<"all" | "pending" | "rejected">("all");
  const [simpkbPreview, setSIMPKBPreview] = useState<TeacherVerification | null>(null);
  const [verifyConfirmItem, setVerifyConfirmItem] = useState<TeacherVerification | null>(null);
  const [verifyRejectItem, setVerifyRejectItem] = useState<TeacherVerification | null>(null);
  const [verifyRejectReason, setVerifyRejectReason] = useState("");
  const [appealPreview, setAppealPreview] = useState<TeacherVerification | null>(null);
  const [selectedQuestionIds, setSelectedQuestionIds] = useState<Set<string>>(new Set());
  const [removeTarget, setRemoveTarget] = useState<{ kind: "user" | "package" | "question"; id: string; label: string; isCBT?: boolean } | null>(null);
  const [bulkDeleteTarget, setBulkDeleteTarget] = useState<{ mode: "all" | "selected"; packageId: string; count: number } | null>(null);
  const [summaryWidgets, setSummaryWidgets] = useState<SummaryWidget[]>(() => {
    try {
      const saved = typeof localStorage === "undefined" ? null : JSON.parse(localStorage.getItem("tka.adminSummary.widgets") ?? "null");
      if (Array.isArray(saved) && saved.length) return saved as SummaryWidget[];
    } catch { /* ignore */ }
    return SUMMARY_WIDGETS.map((widget) => widget.id);
  });
  const [summaryCustomize, setSummaryCustomize] = useState(false);
  const [summaryFinance, setSummaryFinance] = useState<FinanceOverview | null>(null);
  const [payoutQueueCount, setPayoutQueueCount] = useState(0);
  const [pendingVerifications, setPendingVerifications] = useState(0);
  const [cbtSettings, setCbtSettings] = useState<CBTPublishSetting[]>([]);
  useEffect(() => {
    try { localStorage.setItem("tka.adminSummary.widgets", JSON.stringify(summaryWidgets)); } catch { /* ignore */ }
  }, [summaryWidgets]);
  useEffect(() => {
    if (role !== "owner") return;
    api.adminFinance().then((finance) => {
      setSummaryFinance(finance.overview);
      setPayoutQueueCount(finance.payout_requests.filter((item) => item.status === "submitted" || item.status === "approved").length);
    }).catch(() => { /* widget tetap menampilkan data yang tersedia */ });
    api.teacherVerifications(1, 1, "pending").then((result) => setPendingVerifications(result.total)).catch(() => { /* ignore */ });
  }, [role, refreshKey]);
  const toggleSummaryWidget = (id: SummaryWidget) => setSummaryWidgets((current) => current.includes(id) ? current.filter((widget) => widget !== id) : [...current, id]);

  const debouncedUserQuery = useDebounced(userQuery);
  const debouncedCatalogQuery = useDebounced(catalogQuery);

  const effectiveView: AdminView = allowedViews.includes(view) ? view : allowedViews[0];

  const navigate = (next: AdminView) => {
    setView(next);
    setSelectedPackageId(null);
    setPkgExamPage(1);
    setPkgQuestionPage(1);
    const group = groupOfView(next);
    setOpenGroups(group ? new Set([group]) : new Set());
  };

  const reload = useCallback(async () => {
    const tasks: Promise<unknown>[] = [];
    if (role === "owner" || role === "admin") tasks.push(api.adminDashboard().then(setData));
    if (role === "owner") tasks.push(api.adminFinance().then((value) => setUserFinance(value.users)));
    tasks.push(api.adminTransactions(txnPage, 25, txnStatus || undefined).then(setTransactions));
    tasks.push(api.globalRanking(rankingLevel === "semua" ? undefined : rankingLevel, rankingPage, 50, rankingMode === "activity" ? "activity" : undefined).then((result) => setRankingPageData(result)));
    await Promise.all(tasks);
  }, [rankingLevel, rankingPage, rankingMode, role, txnPage, txnStatus]);
  useEffect(() => {
    api.adminTransactions(txnPage, 25, txnStatus || undefined).then(setTransactions).catch((reason) => setError(reason instanceof Error ? reason.message : "Transaksi gagal dimuat."));
  }, [role, refreshKey, txnPage, txnStatus]);
  useEffect(() => {
    if (effectiveView !== "cbt-pengaturan" || (role !== "owner" && role !== "admin")) return;
    api.adminCBTSettings().then(setCbtSettings).catch((reason) => setError(reason instanceof Error ? reason.message : "Pengaturan CBT gagal dimuat."));
  }, [effectiveView, role, refreshKey]);
  useEffect(() => {
    if (role !== "owner") return;
    api.adminUsers(userPage, 25, {
      role: userRoleFilter === "all" ? undefined : userRoleFilter,
      level: userLevelFilter === "all" ? undefined : userLevelFilter,
      q: debouncedUserQuery.trim() || undefined,
    }).then(setUsersPage).catch((reason) => setError(reason instanceof Error ? reason.message : "Data pengguna gagal dimuat."));
  }, [role, refreshKey, userPage, userRoleFilter, userLevelFilter, debouncedUserQuery]);
  useEffect(() => {
    if (role !== "owner" && role !== "admin") return;
    api.adminPackages(packagePage, 12, {
      status: catalogStatusTab,
      jenjang: catalogTab === "semua" ? undefined : catalogTab,
      q: debouncedCatalogQuery.trim() || undefined,
      examType: effectiveView === "cbt" ? "cbt" : "sell",
    }).then(setPackagesPage).catch((reason) => setError(reason instanceof Error ? reason.message : "Katalog gagal dimuat."));
  }, [role, refreshKey, packagePage, catalogStatusTab, catalogTab, debouncedCatalogQuery, effectiveView]);
  useEffect(() => {
    if (!selectedPackageId) return;
    api.adminExams(selectedPackageId, pkgExamPage, 25).then(setPkgExamsPage).catch((reason) => setError(reason instanceof Error ? reason.message : "Ujian gagal dimuat."));
  }, [selectedPackageId, refreshKey, pkgExamPage]);
  useEffect(() => {
    if (!selectedPackageId) return;
    api.adminQuestions(selectedPackageId, pkgQuestionPage, 25).then(setPkgQuestionsPage).catch((reason) => setError(reason instanceof Error ? reason.message : "Bank soal gagal dimuat."));
  }, [selectedPackageId, refreshKey, pkgQuestionPage]);
  useEffect(() => {
    const tasks: Promise<unknown>[] = [];
    if (role === "owner" || role === "admin") tasks.push(api.adminDashboard().then(setData));
    if (role === "owner") tasks.push(api.adminFinance().then((value) => setUserFinance(value.users)));
    tasks.push(api.siteSettings().then(setSiteSettings));
    if (role === "owner") tasks.push(api.adminAudit().then(setAudit));
    Promise.all(tasks).catch((reason) => setError(reason instanceof Error ? reason.message : "Data admin gagal dimuat."));
  }, [role, refreshKey]);
  useEffect(() => {
    api.globalRanking(rankingLevel === "semua" ? undefined : rankingLevel, rankingPage, 50, rankingMode === "activity" ? "activity" : undefined).then(setRankingPageData).catch((reason) => setError(reason instanceof Error ? reason.message : "Ranking gagal dimuat."));
  }, [rankingLevel, rankingPage, rankingMode, refreshKey]);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- bump pagination to page 1 when filters change
  useEffect(() => { setUserPage(1); }, [userQuery, userRoleFilter, userLevelFilter]);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- bump pagination to page 1 when filters change
  useEffect(() => { setPackagePage(1); }, [catalogStatusTab, catalogTab, catalogQuery, effectiveView]);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- bump pagination to page 1 when filters change
  useEffect(() => { setRankingPage(1); }, [rankingLevel, rankingMode]);
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(""), 3500); return () => window.clearTimeout(timer); }, [toast]);

  async function action(task: () => Promise<unknown>, success?: string) {
    setSaving("mutation"); setError("");
    try { await task(); setRefreshKey((n) => n + 1); await reload(); setDialog(null); if (success) setToast(success); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Operasi gagal."); }
    finally { setSaving(""); }
  }
  function remove(kind: "user"|"package"|"question", id: string, label: string, isCBT = false) {
    setRemoveTarget({ kind, id, label, isCBT });
  }
  async function confirmRemove() {
    if (!removeTarget) return;
    const target = removeTarget;
    setRemoveTarget(null);
    const message = target.kind === "package" ? "Paket soal berhasil dihapus." : target.kind === "user" ? "Pengguna berhasil dihapus." : "Soal berhasil dihapus.";
    await action(() => target.kind === "user" ? api.deleteAdminUser(target.id) : target.kind === "package" ? api.deleteAdminPackage(target.id) : api.deleteAdminQuestion(target.id), message);
  }
  function toggleQuestionSelection(id: string) {
    setSelectedQuestionIds((current) => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; });
  }
  async function confirmBulkDeleteQuestions() {
    if (!bulkDeleteTarget) return;
    const target = bulkDeleteTarget;
    setBulkDeleteTarget(null);
    const ids = target.mode === "selected" ? [...selectedQuestionIds] : [];
    await action(async () => { await api.bulkDeleteAdminQuestions(ids, ids.length ? undefined : target.packageId); setSelectedQuestionIds(new Set()); });
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
    await action(() => api.updateAdminPackage(item.id, { title: item.title, kode: item.kode, description: item.description, price: item.price, validity_days: item.validity_days, status: item.status === "active" ? "inactive" : "active", jenjang: item.jenjang, exam_type: item.exam_type, cbt_token: item.cbt_token ?? "", kategori_id: item.kategori_id ?? "", kelas_id: item.kelas_id ?? "" }));
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
  useEffect(() => { if (role === "owner" || role === "admin") void loadTestimonials(); }, [role, loadTestimonials]);
  useEffect(() => {
    if (role !== "owner" && role !== "admin") return;
    api.teacherVerifications(verifPage, 25, verifFilter === "all" ? undefined : verifFilter).then((result) => {
      setVerifications(result);
      for (const item of result.items) {
        if (item.simpkb_status === "pending" && item.teacher_ktp) {
          api.checkTeacherSIMPKB(item.id).catch(() => {});
        }
      }
    }).catch((reason) => setError(reason instanceof Error ? reason.message : "Data verifikasi guru gagal dimuat."));
  }, [role, refreshKey, verifPage, verifFilter]);
  // eslint-disable-next-line react-hooks/set-state-in-effect -- bump pagination to page 1 when filter changes
  useEffect(() => { setVerifPage(1); }, [verifFilter]);

  const menus: Array<[AdminView, string]> = allowedViews.map((key) => [key, adminViewLabels[key]]);
  const topLevelViews = allowedViews.filter((key) => groupOfView(key) === null);
  const masterItems = (category: MasterCategory): MasterItem[] => category === "mapel" ? data?.mapels ?? [] : category === "jenjang" ? data?.jenjangs ?? [] : category === "tahun-ajaran" ? data?.academic_years ?? [] : category === "kategori" ? data?.kategoris ?? [] : data?.kelas ?? [];
  const publishedCount = packagesPage.counts.active;
  const publishedTeacherCount = packagesPage.counts.active_teacher;
  const waitingCount = packagesPage.counts.inactive;
  const waitingTeacherCount = packagesPage.counts.inactive_teacher;
  const visiblePackages = packagesPage.items;
  const selectedPackage = packagesPage.items.find((item) => item.id === selectedPackageId) ?? null;
  const packageExams = pkgExamsPage.items;
  const packageQuestions = pkgQuestionsPage.items;
  const visibleUsers = usersPage.items;
  const userCounts = usersPage.summary;
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
  const downloadInvoice = async (item: AdminTransaction) => {
    try {
      const blob = await api.downloadInvoice(item.id);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `${item.invoice_number || "invoice"}.pdf`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Invoice gagal diunduh."); }
  };
  async function refundTransaction(item: AdminTransaction) {
    const reason = window.prompt(`Refund transaksi ${item.invoice_number}?\nMasukkan alasan refund (wajib):`, "");
    if (reason === null) return;
    if (!reason.trim()) { setError("Alasan refund wajib diisi."); return; }
    await action(() => api.adminRefundTransaction(item.id, reason.trim()));
    setError("");
  }
  async function approveTeacher(item: TeacherVerification) {
    setVerifyConfirmItem(item);
  }
  async function confirmApproveTeacher() {
    if (!verifyConfirmItem) return;
    const item = verifyConfirmItem;
    setVerifyConfirmItem(null);
    await action(() => api.approveTeacher(item.id));
  }
  async function rejectTeacher(item: TeacherVerification) {
    setVerifyRejectItem(item);
    setVerifyRejectReason("");
  }
  async function confirmRejectTeacher() {
    if (!verifyRejectItem) return;
    if (!verifyRejectReason.trim()) { setError("Alasan penolakan wajib diisi."); return; }
    const item = verifyRejectItem;
    const reason = verifyRejectReason.trim();
    setVerifyRejectItem(null);
    setVerifyRejectReason("");
    await action(() => api.rejectTeacher(item.id, reason));
    setError("");
  }
  async function checkSIMPKB(item: TeacherVerification) {
    setSaving("simpkb-check"); setError(""); setSIMPKBPreview(null);
    try {
      await api.checkTeacherSIMPKB(item.id);
      let finalItem = item;
      for (let attempt = 0; attempt < 12; attempt++) {
        await new Promise((resolve) => setTimeout(resolve, 4000));
        const fresh = await api.teacherVerifications(verifPage, 25, verifFilter === "all" ? undefined : verifFilter);
        setVerifications(fresh);
        const row = fresh.items.find((it) => it.id === item.id);
        if (row) finalItem = row;
        if (!row || row.simpkb_status !== "checking") break;
      }
      if (finalItem.simpkb_image) setSIMPKBPreview(finalItem);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Pengecekan SIMPKB gagal."); }
    finally { setSaving(""); }
  }

  return <main><div className="admin-layout">
    <aside className="admin-sidebar"><Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/><span>{siteSettings?.platform_name??""}<small>Administrator</small></span></Link><nav>{topLevelViews.map((key) => <button key={key} className={effectiveView===key?"active":""} onClick={()=>navigate(key)}>{adminViewLabels[key]}</button>)}
  {sidebarGroups.map((group) => {
    const views = group.views.filter((key) => allowedViews.includes(key));
    if (views.length === 0) return null;
    const active = views.includes(effectiveView);
    const open = openGroups.has(group.name) || active;
    return <div className="sidebar-group" key={group.name}><button type="button" className={"sidebar-group-head" + (active ? " active" : "")} aria-expanded={open} onClick={() => setOpenGroups((current) => { const next = new Set(current); if (next.has(group.name)) next.delete(group.name); else { next.clear(); next.add(group.name); } return next; })}>{group.name}<span className={"group-chevron" + (open ? " open" : "")}>{open ? "▲" : "▼"}</span></button>{open && views.map((key) => <button key={key} className={"sidebar-submenu" + (effectiveView===key ? " active" : "")} onClick={()=>navigate(key)}>{adminViewLabels[key]}</button>)}</div>;
  })}
  </nav><LogoutButton className="admin-logout" loggingOut={loggingOut} onLogout={onLogout}/></aside>
    <section className="admin-main">
      <header className="admin-topbar"><div><p className="eyebrow">Panel administrator</p><h1>{menus.find(([key])=>key===view)?.[1]}</h1></div><span className="topbar-actions"><NotificationBell/><button className="admin-identity" onClick={()=>navigate("profil")}><span>{(user.name?.trim()||user.email)[0].toUpperCase()}</span><div><strong>{user.name?.trim()||"Nama belum dilengkapi"}</strong><small>{user.email} · {userRoleLabel(user.role)}</small></div></button></span></header>
      {error && <p className="error">{error}</p>}{!data && !error && role !== "finance" && <div className="card">Memuat data administrasi...</div>}

      {data && effectiveView==="ringkasan" && <div className="admin-view summary-view">
        <div className="summary-hero"><div><p className="eyebrow">Dashboard administrator</p><h2>Ringkasan operasional</h2><p className="muted">{new Date().toLocaleDateString("id-ID",{weekday:"long",day:"numeric",month:"long",year:"numeric"})}</p></div><div className="summary-actions"><button className="button secondary" onClick={()=>setSummaryCustomize((open)=>!open)}>{summaryCustomize?"Tutup pengaturan":"Sesuaikan tampilan"}</button><button className="button" onClick={()=>setDialog({kind:"package"})}>+ Tambah paket</button></div></div>
        {summaryCustomize&&<div className="summary-customize"><p className="eyebrow">Pilih yang ditampilkan</p><p className="muted">Aktifkan atau nonaktifkan modul pada ringkasan. Pilihan tersimpan otomatis di perangkat ini.</p><div className="summary-widget-options">{SUMMARY_WIDGETS.map((widget)=><label key={widget.id} className={summaryWidgets.includes(widget.id)?"on":""}><input type="checkbox" checked={summaryWidgets.includes(widget.id)} onChange={()=>toggleSummaryWidget(widget.id)}/><strong>{widget.label}</strong><small>{widget.hint}</small></label>)}</div></div>}
        {summaryWidgets.includes("kpis")&&<div className="admin-stats summary-kpis"><Stat label="Total User" value={data.overview.total_users}/><Stat label="Siswa" value={data.overview.total_students}/><Stat label="Paket Soal" value={data.overview.total_packages}/><Stat label="Ujian" value={data.overview.total_exams}/><Stat label="Transaksi" value={data.overview.total_transactions}/><Stat label="Pembayaran Berhasil" value={data.overview.paid_transactions}/></div>}
        <div className="summary-grid">
          <div className="summary-main">
            {summaryWidgets.includes("revenue")&&summaryFinance&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Keuangan</p><h3>Ringkasan pendapatan</h3></div><button className="button secondary small-button" onClick={()=>navigate("finance")}>Buka finance</button></div>
              <div className="summary-finance"><div className="summary-metric green"><small>Pendapatan kotor</small><strong>{money.format(summaryFinance.gross_revenue)}</strong></div><div className="summary-metric"><small>Transaksi berhasil</small><strong>{summaryFinance.paid_transactions}</strong></div><div className="summary-metric orange"><small>Antrean pencairan</small><strong>{payoutQueueCount}</strong></div></div>
            </section>}
            {summaryWidgets.includes("popular")&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Kinerja</p><h3>Paket terlaris</h3></div><button className="button secondary small-button" onClick={()=>navigate("katalog")}>Buka katalog</button></div>
              {[...data.packages].sort((a,b)=>b.sales_count-a.sales_count).slice(0,5).map((item,index)=><div className="summary-row" key={item.id}><div className="summary-rank"><strong>{index+1}</strong></div><div className="grow"><strong>{item.title}</strong><small>{item.exam_type==="cbt"?"CBT · ":"Penjualan · "}{item.status==="active"?"Publish":"Menunggu verifikasi"} · {item.publisher_email}</small></div><div className="value"><strong>{item.sales_count}</strong><small>terjual</small></div></div>)}
              {data.packages.length===0&&<p className="summary-empty">Belum ada paket soal.</p>}
            </section>}
            {summaryWidgets.includes("recent")&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Aktivitas</p><h3>Transaksi terbaru</h3></div><button className="button secondary small-button" onClick={()=>navigate("transaksi")}>Buka transaksi</button></div>
              {[...data.transactions].sort((a,b)=>new Date(b.created_at).getTime()-new Date(a.created_at).getTime()).slice(0,6).map((item)=><div className="summary-row" key={item.id}><span className={`summary-badge ${item.status}`}>{item.status==="paid"?"Lunas":item.status==="pending"?"Pending":item.status}</span><div className="grow"><strong>{item.user_email}</strong><small>{item.package_title}</small></div><div className="value"><strong>{money.format(item.amount)}</strong><small>{item.invoice_number}</small></div></div>)}
              {data.transactions.length===0&&<p className="summary-empty">Belum ada transaksi.</p>}
            </section>}
          </div>
          <div className="summary-side summary-gap">
            {summaryWidgets.includes("attention")&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Tindak lanjut</p><h3>Perlu perhatian</h3></div></div>
              <div className="summary-list">
                <button className="summary-row" onClick={()=>navigate("verifikasi")}><span className="summary-badge warn">{pendingVerifications}</span><div className="grow"><strong>Verifikasi guru tertunda</strong><small>{pendingVerifications===0?"Tidak ada antrean menunggu":"Menunggu persetujuan SIMPKB"}</small></div></button>
                {role==="owner"&&<button className="summary-row" onClick={()=>navigate("finance")}><span className="summary-badge warn">{payoutQueueCount}</span><div className="grow"><strong>Permintaan pencairan</strong><small>{payoutQueueCount===0?"Tidak ada antrean":"Menunggu proses transfer"}</small></div></button>}
                <button className="summary-row" onClick={()=>navigate("katalog")}><span className={`summary-badge ${data.packages.some((item)=>item.status!=="active")?"warn":"ok"}`}>{data.packages.filter((item)=>item.status!=="active").length}</span><div className="grow"><strong>Paket belum dipublish</strong><small>Menunggu verifikasi admin</small></div></button>
              </div>
            </section>}
            {summaryWidgets.includes("newusers")&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Pertumbuhan</p><h3>Pendaftar baru</h3></div><button className="button secondary small-button" onClick={()=>navigate("users")}>Buka user</button></div>
              {[...data.users].sort((a,b)=>new Date(b.created_at).getTime()-new Date(a.created_at).getTime()).slice(0,6).map((item)=><div className="summary-row" key={item.id}><span className="summary-avatar" aria-hidden="true">{(item.name?.trim()||item.email)[0].toUpperCase()}</span><div className="grow"><strong>{item.name?.trim()||item.email}</strong><small>{userRoleLabel(item.role)} · {new Date(item.created_at).toLocaleDateString("id-ID",{day:"numeric",month:"short",year:"numeric"})}</small></div></div>)}
              {data.users.length===0&&<p className="summary-empty">Belum ada pengguna.</p>}
            </section>}
            {summaryWidgets.includes("ranking")&&<section className="card summary-card">
              <div className="summary-card-heading"><div><p className="eyebrow">Kompetisi</p><h3>Peringkat teratas</h3></div><button className="button secondary small-button" onClick={()=>navigate("ranking")}>Buka ranking</button></div>
              {rankingPageData.items.slice(0,5).map((item)=><div className="summary-row" key={item.rank}><div className="summary-rank"><strong>{item.rank}</strong></div><div className="grow"><strong>{item.display_name}</strong><small>{item.school_level} · {item.score.toFixed(1)}% persentil</small></div><div className="value"><strong>{formatDuration(item.duration_seconds)}</strong><small>{item.exams_done} ujian</small></div></div>)}
              {rankingPageData.items.length===0&&<p className="summary-empty">Belum ada data peringkat.</p>}
            </section>}
          </div>
        </div>
      </div>}

      {data && effectiveView==="users" && <section className="admin-view user-management">
        <div className="user-management-heading"><div><p className="eyebrow">Manajemen akses</p><h2>Data pengguna</h2><p className="muted">Kelola akun, peran, dan jenjang pengguna dalam satu tempat.</p></div><button className="button user-add-button" onClick={()=>setDialog({kind:"user"})}><span aria-hidden="true">+</span> Tambah pengguna</button></div>
        <div className="user-summary-grid">
          <article className="user-summary-card total"><span className="user-summary-icon" aria-hidden="true">U</span><div><small>Total pengguna</small><strong>{userCounts.total}</strong></div></article>
          <article className="user-summary-card student"><span className="user-summary-icon" aria-hidden="true">S</span><div><small>Siswa</small><strong>{userCounts.students}</strong></div></article>
          <article className="user-summary-card teacher"><span className="user-summary-icon" aria-hidden="true">G</span><div><small>Guru</small><strong>{userCounts.teachers}</strong></div></article>
          <article className="user-summary-card admin"><span className="user-summary-icon" aria-hidden="true">A</span><div><small>Administrator</small><strong>{userCounts.admins}</strong></div></article>
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
          <div className="user-results-meta"><span>Menampilkan <strong>{visibleUsers.length}</strong> dari {usersPage.total} pengguna</span>{saving && <span className="user-saving">Menyimpan perubahan...</span>}</div>
          <div className="user-table-wrap"><table className="user-table"><thead><tr><th>Pengguna</th><th>Peran</th><th>Jenjang</th><th>Status</th><th>Terdaftar</th><th><span className="sr-only">Aksi</span></th></tr></thead><tbody>{visibleUsers.map((item)=>{
            const isCurrentUser = item.id === user.id;
            const accountActive = userFinanceById.get(item.id)?.account_status !== "hold";
            return <tr key={item.id}><td><div className="user-cell"><span className={`user-avatar ${item.role}`}>{userInitials(item)}</span><div><strong>{item.name?.trim() || item.email.split("@")[0]}</strong><span>{item.email}</span></div>{isCurrentUser&&<span className="current-user-badge">Akun Anda</span>}</div></td><td><span className={`user-readonly-pill role-${item.role}`}><span aria-hidden="true"/>{userRoleLabel(item.role)}</span></td><td><span className="user-readonly-pill level">{item.school_level === "SMA" ? "SMA / SMK" : item.school_level}</span></td><td><span className={`user-account-status ${accountActive?"active":"inactive"}`}><span aria-hidden="true"/>{accountActive?"Aktif":"Nonaktif"}</span></td><td><div className="user-date"><strong>{new Date(item.created_at).toLocaleDateString("id-ID",{day:"2-digit",month:"short",year:"numeric"})}</strong><span>{new Date(item.created_at).toLocaleDateString("id-ID",{weekday:"long"})}</span></div></td><td className="user-actions"><div className="user-row-actions"><button className={`user-status-button ${accountActive?"deactivate":"activate"}`} disabled={isCurrentUser||saving!==""} title={isCurrentUser?"Akun yang sedang digunakan harus tetap aktif":accountActive?"Nonaktifkan akun":"Aktifkan akun"} onClick={()=>void toggleUserStatus(item)}>{accountActive?"Nonaktifkan":"Aktifkan"}</button><button className="user-edit-button" disabled={saving!==""} onClick={()=>setDialog({kind:"user",item})}><span aria-hidden="true">✎</span> Edit</button><button className="user-delete-button" aria-label={`Hapus ${item.email}`} title={isCurrentUser?"Akun yang sedang digunakan tidak dapat dihapus":"Hapus pengguna"} disabled={isCurrentUser||saving!==""} onClick={()=>void remove("user",item.id,item.email)}><span aria-hidden="true">×</span><span>Hapus</span></button></div></td></tr>;
          })}{visibleUsers.length===0&&<tr><td colSpan={6}><div className="user-empty"><span aria-hidden="true">⌕</span><strong>Pengguna tidak ditemukan</strong><p>Ubah kata kunci atau filter untuk melihat hasil lainnya.</p><button onClick={()=>{setUserQuery("");setUserRoleFilter("all");setUserLevelFilter("all");}}>Reset filter</button></div></td></tr>}</tbody></table></div>
          <div className="pagination-row"><button className="table-action" disabled={usersPage.page<=1||saving!==""} onClick={()=>setUserPage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {usersPage.page} · {usersPage.total} pengguna</span><button className="table-action" disabled={usersPage.page*usersPage.count>=usersPage.total||saving!==""} onClick={()=>setUserPage((p)=>p+1)}>Berikutnya ›</button></div>
        </div>
      </section>}

      {data && (effectiveView==="katalog"||effectiveView==="cbt") && !selectedPackage && <section className="admin-view catalog-admin">
        <div className="admin-catalog-heading"><div><p className="eyebrow">{effectiveView==="cbt"?"Bank soal CBT":"Katalog paket soal"}</p><h2>{effectiveView==="cbt"?"Semua paket ujian CBT dari seluruh publisher":"Kelola paket, ujian, dan bank soal"}</h2><p className="muted">{effectiveView==="cbt"?"Menampilkan hanya paket dengan tipe CBT.":"Klik salah satu kartu untuk membuka lingkungan pengelolaan lengkap."}</p></div><button className="button" onClick={()=>setDialog(effectiveView==="cbt"?{kind:"package",defaultExamType:"cbt"}:{kind:"package"})}>+ Tambah paket</button></div>
        <div className="content-subtabs catalog-status-tabs"><button className={catalogStatusTab==="active"?"active":""} onClick={()=>setCatalogStatusTab("active")}><b>Publish</b><small>{publishedTeacherCount} guru · {publishedCount-publishedTeacherCount} platform</small><span>{publishedCount}</span></button><button className={catalogStatusTab==="inactive"?"active":""} onClick={()=>setCatalogStatusTab("inactive")}><b>Menunggu verifikasi</b><small>{waitingTeacherCount} dari guru</small><span>{waitingCount}</span></button></div>
        <div className="catalog-toolbar"><input className="catalog-search" type="search" placeholder="Cari kode soal, nama paket, atau pembuat..." value={catalogQuery} onChange={(event)=>setCatalogQuery(event.target.value)}/><div className="content-subtabs">{["semua","SD","SMP","SMA"].map((tab)=><button key={tab} className={catalogTab===tab?"active":""} onClick={()=>setCatalogTab(tab)}>{tab==="semua"?"Semua":tab}</button>)}</div></div>
        <div className="admin-package-grid">{visiblePackages.map((item)=><article className={`card admin-package-card ${item.status==="inactive"?"inactive":""}`} key={item.id} onClick={()=>{setPkgExamPage(1);setPkgQuestionPage(1);setSelectedPackageId(item.id);}}><div className="admin-package-card-top"><PackageStatusPill status={item.status}/><span className="package-code">{item.kode}</span></div><span className="package-jenjang">{item.jenjang}</span><span className="package-exam-type">{item.exam_type === "cbt" ? "CBT" : "Sell"}</span>{item.kategori_name?<small className="package-publisher">Kategori: {item.kategori_name}{item.kelas_name?` · Kelas ${item.kelas_name}`:""}</small>:null}<h3>{item.title}</h3><p>{item.description}</p><strong>{item.price===0?"Gratis":money.format(item.price)}</strong><small>{(data.exams.filter((exam)=>exam.package_id===item.id).length)} ujian · {data.questions.filter((q)=>new Set(data.exams.filter((exam)=>exam.package_id===item.id).map((exam)=>exam.id)).has(q.exam_id)).length} soal</small>{item.publisher_email&&<small className="package-publisher">Pembuat: {item.publisher_email}</small>}{item.status==="active"&&<CatalogStats sales={item.sales_count} views={item.view_count}/>}<div><button className="table-action" onClick={(event)=>{event.stopPropagation();setDialog({kind:"package",item})}}>Edit</button><button className="danger-action" onClick={(event)=>{event.stopPropagation();void remove("package",item.id,item.title,item.exam_type==="cbt")}}>Hapus</button></div></article>)}
        {visiblePackages.length===0 && <div className="card empty-state">Tidak ada paket {catalogStatusTab==="active"?"yang sudah dipublikasikan":"yang sedang menunggu verifikasi"} untuk filter ini.</div>}</div>
        <div className="pagination-row"><button className="table-action" disabled={packagesPage.page<=1||saving!==""} onClick={()=>setPackagePage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {packagesPage.page} · {packagesPage.total} paket</span><button className="table-action" disabled={packagesPage.page*packagesPage.count>=packagesPage.total||saving!==""} onClick={()=>setPackagePage((p)=>p+1)}>Berikutnya ›</button></div>
      </section>}

      {data && (effectiveView==="katalog"||effectiveView==="cbt") && selectedPackage && <section className="admin-view catalog-detail">
        <div className="catalog-detail-heading"><div><p className="eyebrow">{effectiveView==="cbt"?"Detail paket CBT":"Lingkungan katalog"}</p><h2>{selectedPackage.title}</h2></div><div className="catalog-detail-actions"><PackageStatusPill status={selectedPackage.status}/><button className="table-action" onClick={()=>setDialog({kind:"package",item:selectedPackage})}>Edit paket</button><button className={`table-action ${selectedPackage.status==="active"?"toggle-off":""}`} disabled={saving!==""} onClick={()=>void togglePackageStatus(selectedPackage)}>{selectedPackage.status==="active"?"Nonaktifkan":selectedPackage.publisher_email?"Verifikasi & publikasikan":"Aktifkan"}</button><button className="button secondary" onClick={()=>setSelectedPackageId(null)}>← Kembali ke katalog</button></div></div>
        <div className="card admin-package-spec"><div className="admin-package-spec-desc"><span>Deskripsi</span><p>{selectedPackage.description || "Tidak ada deskripsi."}</p></div><div className="admin-package-spec-grid"><div><span>Kode soal</span><strong>{selectedPackage.kode}</strong></div><div><span>Jenjang</span><strong>{selectedPackage.jenjang}</strong></div><div><span>Jenis ujian</span><strong>{selectedPackage.exam_type === "cbt" ? "CBT" : "Sell"}</strong></div>{selectedPackage.exam_type==="cbt"&&<div><span>Kode rahasia (token CBT)</span><strong>{selectedPackage.cbt_token||"—"}</strong></div>}{selectedPackage.kategori_name?<div><span>Kategori</span><strong>{selectedPackage.kategori_name}</strong></div>:null}{selectedPackage.kelas_name?<div><span>Keterangan kelas</span><strong>{selectedPackage.kelas_name}</strong></div>:null}<div><span>Harga</span><strong>{selectedPackage.price===0?"Gratis":money.format(selectedPackage.price)}</strong></div><div><span>Masa aktif</span><strong>{selectedPackage.validity_days} hari</strong></div><div><span>Ujian</span><strong>{pkgExamsPage.total}</strong></div><div><span>Total soal</span><strong>{pkgQuestionsPage.total}</strong></div><div><span>Dibuat</span><strong>{new Date(selectedPackage.created_at).toLocaleDateString("id-ID")}</strong></div>{selectedPackage.publisher_email&&<div><span>Pembuat</span><strong>{selectedPackage.publisher_email}</strong></div>}</div></div>
        <div className="bank-section"><ViewHeader title="Bank soal" templates onAdd={()=>setDialog({kind:"question"})}/><QuestionBankTools packageTitle={selectedPackage.title} packageCode={selectedPackage.kode} level={selectedPackage.jenjang} durationMinutes={packageExams[0]?.duration_minutes} questions={packageQuestions.filter(item=>item.status==="active")} examId={packageExams[0]?.id} subjectName={(data?.mapels??[]).find((item)=>item.id===packageExams[0]?.mapel_id)?.nama??packageQuestions[0]?.subject_name??""} onImportQuestion={async (input)=>{const created=await api.createAdminQuestion(input); setPkgQuestionsPage((page)=>page.items.some((item)=>item.id===created.id)?page:{...page,items:[created,...page.items]});}}/><AdminTable headers={["","Ujian","Mapel","Bentuk","Soal","Kunci","Bobot","Status","Aksi"]}>{packageQuestions.length?packageQuestions.map((item)=><tr key={item.id}><td><input type="checkbox" aria-label={`Pilih soal ${item.subject_name}`} checked={selectedQuestionIds.has(item.id)} onChange={()=>toggleQuestionSelection(item.id)} /></td><td>{item.exam_title}</td><td>{item.subject_name}</td><td>{questionTypeLabel(item.question_type)}</td><td>{item.content_text.length>60?`${item.content_text.slice(0,60)}...`:item.content_text}</td><td>{item.correct_answer}</td><td>{item.score_weight}</td><td><StatusPill status={item.status}/></td><td><button className="table-action" onClick={()=>setDialog({kind:"question",item})}>Edit</button><button className={`table-action ${item.status==="active"?"toggle-off":""}`} disabled={saving!==""} onClick={()=>void toggleQuestionStatus(item)}>{item.status==="active"?"Nonaktifkan":"Aktifkan"}</button><button className="danger-action" onClick={()=>void remove("question",item.id,item.subject_name)}>Hapus</button></td></tr>):<tr><td colSpan={9} className="empty-state">Belum ada soal di paket ini.</td></tr>}</AdminTable>{packageQuestions.length>0&&<div className="bulk-actions"><button className="button secondary" type="button" disabled={!selectedQuestionIds.size||saving!==""} onClick={()=>setBulkDeleteTarget({mode:"selected",packageId:selectedPackage.id,count:selectedQuestionIds.size})}>Hapus soal terpilih ({selectedQuestionIds.size})</button><button className="danger-action" type="button" disabled={packageQuestions.length===0||saving!==""} onClick={()=>setBulkDeleteTarget({mode:"all",packageId:selectedPackage.id,count:packageQuestions.length})}>Hapus semua soal</button></div>}<div className="pagination-row"><button className="table-action" disabled={pkgQuestionsPage.page<=1||saving!==""} onClick={()=>setPkgQuestionPage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {pkgQuestionsPage.page} · {pkgQuestionsPage.total} soal</span><button className="table-action" disabled={pkgQuestionsPage.page*pkgQuestionsPage.count>=pkgQuestionsPage.total||saving!==""} onClick={()=>setPkgQuestionPage((p)=>p+1)}>Berikutnya ›</button></div></div>
      </section>}

      {effectiveView==="cbt-pengaturan" && <section className="admin-view">
        <div className="admin-view-header"><div><p className="eyebrow">Publish pembahasan</p><h2>Pengaturan CBT</h2><p className="muted">Publikasikan pembahasan ujian CBT agar siswa yang sudah mengerjakan dapat melihat analitik hasilnya.</p></div></div>
        <CBTSettingsPanel items={cbtSettings} saving={saving!==""} loadParticipants={(examID) => api.adminCBTParticipants(examID)} onToggle={async (item, publish) => { setSaving("cbt-publish"); try { const updated = await api.adminSetCBTPublish(item.exam_id, publish); setCbtSettings((current) => current.map((it) => it.exam_id === updated.exam_id ? updated : it)); setError(""); } catch (reason) { setError(reason instanceof Error ? reason.message : "Gagal mengubah pengaturan CBT."); } finally { setSaving(""); } }}/>
      </section>}
      {effectiveView==="transaksi" && <section className="admin-view">
        <div className="admin-view-header"><div><p className="eyebrow">Riwayat pembayaran</p><h2>Transaksi</h2><p className="muted">Pembelian paket soal oleh pengguna. Unduh invoice atau proses refund untuk transaksi berstatus lunas.</p></div></div>
        <div className="content-subtabs">{([["","Semua"],["paid","Lunas"],["pending","Menunggu"],["failed","Gagal"],["expired","Kedaluwarsa"],["refunded","Refund"]] as const).map(([key,label])=><button key={key} className={txnStatus===key?"active":""} onClick={()=>{setTxnStatus(key);setTxnPage(1);}}>{label}</button>)}</div>
        <AdminTable headers={["Invoice","User","Paket","Nominal","Status","Aksi"]}>{transactions.items.length?transactions.items.map((item)=><tr key={item.id}><td>{item.invoice_number}</td><td>{item.user_email}</td><td>{item.package_title}</td><td>{money.format(item.amount)}</td><td><span className={`status-pill ${item.status}`}>{item.status}</span></td><td><div className="user-row-actions"><button className="table-action" onClick={()=>void downloadInvoice(item)}>Unduh</button>{item.status==="paid"&&<button className="danger-action" disabled={saving!==""} onClick={()=>void refundTransaction(item)}>Refund</button>}</div></td></tr>):<tr><td colSpan={6} className="empty-state">Belum ada transaksi.</td></tr>}</AdminTable>
        <div className="pagination-row"><button className="table-action" disabled={transactions.page<=1||saving!==""} onClick={()=>setTxnPage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {transactions.page} · {transactions.total} transaksi</span><button className="table-action" disabled={transactions.page*transactions.count>=transactions.total||saving!==""} onClick={()=>setTxnPage((p)=>p+1)}>Berikutnya ›</button></div>
      </section>}
      {effectiveView==="finance"&&<FinancePanel canEditSettings={role==="owner"}/>}
      {effectiveView==="audit" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Jejak audit</p><h2>Audit log tindakan staff</h2><p className="muted">Pencatatan permanen atas perubahan kebijakan keuangan, user, dan pencairan.</p></div></div><AdminTable headers={["Waktu","Aktor","Aksi","Entitas","Rincian"]}>{audit.length?audit.map((item)=><tr key={item.id}><td>{new Date(item.created_at).toLocaleString("id-ID")}</td><td>{item.actor_email}</td><td>{auditActionLabel(item.action)}</td><td><div>{item.entity_type}{item.entity_id&&<small>{item.entity_id.slice(0,8)}</small>}</div></td><td style={{maxWidth:420}}>{formatAuditDetail(item.detail)}</td></tr>):<tr><td colSpan={5} className="empty-state">Belum ada aktivitas tercatat.</td></tr>}</AdminTable></section>}
      {data && effectiveView==="master" && <section className="admin-view master-view">
        <div className="content-subtabs">{masterCategories.map((category)=><button key={category} className={masterTab===category?"active":""} onClick={()=>setMasterTab(category)}>{masterLabel(category)}</button>)}</div>
        <ViewHeader title={`Data ${masterLabel(masterTab)}`} onAdd={()=>setDialog({kind:"master",category:masterTab})}/>
        <AdminTable headers={["Nama","Aksi"]}>{masterItems(masterTab).length?masterItems(masterTab).map((item)=><tr key={item.id}><td>{item.nama}</td><td><button className="table-action" onClick={()=>setDialog({kind:"master",category:masterTab,item})}>Edit</button><button className="danger-action" onClick={()=>void removeMaster(masterTab,item.id,item.nama)}>Hapus</button></td></tr>):<tr><td colSpan={2} className="empty-state">Belum ada data.</td></tr>}</AdminTable>
      </section>}
      {effectiveView==="ranking" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Peringkat siswa</p><h2>Ranking berdasarkan jenjang</h2></div><span className="muted">{rankingMode==="activity"?"Urutkan berdasarkan jumlah ujian berbeda yang diselesaikan":"Rata-rata persentil per ujian, jumlah ujian sebagai pembanding"}</span></div><div className="content-subtabs" style={{display:"flex",gap:8,alignItems:"center",flexWrap:"wrap"}}>{(["semua","SD","SMP","SMA"] as const).map((tab)=><button key={tab} className={rankingLevel===tab?"active":""} onClick={()=>setRankingLevel(tab)}>{tab==="semua"?"Semua":tab}</button>)}<span style={{width:1,height:18,background:"#dfe2eb",margin:"0 4px"}}/><button className={rankingMode==="score"?"active":""} onClick={()=>setRankingMode("score")}>Persentil</button><button className={rankingMode==="activity"?"active":""} onClick={()=>setRankingMode("activity")}>Aktivitas</button></div><AdminTable headers={["Peringkat","Pengguna","Jenjang","Ujian Selesai","Persentil","Terbaik","Waktu","Tanggal"]}>{rankingPageData.items.map((item)=><tr key={`${item.rank}-${item.display_name}`}><td>#{item.rank}</td><td>{item.display_name}</td><td>{item.school_level}</td><td><strong>{item.exams_done}</strong></td><td><strong>{item.score.toFixed(1)}%</strong></td><td>{item.best_percentile.toFixed(1)}%</td><td>{formatDuration(item.duration_seconds)}</td><td>{new Date(item.finished_at).toLocaleDateString("id-ID",{day:"2-digit",month:"short",year:"numeric"})}</td></tr>)}</AdminTable><div className="pagination-row"><button className="table-action" disabled={rankingPageData.page<=1} onClick={()=>setRankingPage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {rankingPageData.page} · {rankingPageData.total} siswa</span><button className="table-action" disabled={rankingPageData.page*rankingPageData.count>=rankingPageData.total} onClick={()=>setRankingPage((p)=>p+1)}>Berikutnya ›</button></div></section>}
      {effectiveView==="testimoni" && <section className="admin-view">
        <div className="admin-view-header"><div><p className="eyebrow">Testimoni siswa</p><h2>Kelola persetujuan testimoni</h2></div></div>
        <div className="content-subtabs">{(["all","pending","approved","rejected"] as const).map((tab)=><button key={tab} className={testimonialFilter===tab?"active":""} onClick={()=>setTestimonialFilter(tab)}>{tab==="all"?"Semua":tab==="pending"?"Menunggu":tab==="approved"?"Disetujui":"Ditolak"}</button>)}</div>
        <AdminTable headers={["Pengguna","Testimoni","Status","Tanggal","Aksi"]}>{testimonials.filter((item)=>testimonialFilter==="all"||item.status===testimonialFilter).length?testimonials.filter((item)=>testimonialFilter==="all"||item.status===testimonialFilter).map((item)=><tr key={item.id}><td><div>{item.user_name || item.user_email}<small>{item.user_email}</small></div></td><td style={{maxWidth:420}}>{item.quote}</td><td><span className={`status-pill ${item.status}`}>{item.status==="approved"?"Disetujui":item.status==="rejected"?"Ditolak":"Menunggu"}</span></td><td>{new Date(item.created_at).toLocaleDateString("id-ID")}</td><td>{item.status!=="approved"&&<button className="table-action" disabled={saving!==""} onClick={()=>void setTestimonialStatus(item,"approved")}>Setujui</button>}{item.status!=="rejected"&&<button className="table-action" disabled={saving!==""} onClick={()=>void setTestimonialStatus(item,"rejected")}>Tolak</button>}<button className="danger-action" disabled={saving!==""} onClick={()=>void removeTestimonial(item)}>Hapus</button></td></tr>):<tr><td colSpan={5} className="empty-state">Tidak ada testimoni pada kategori ini.</td></tr>}</AdminTable>
      </section>}
      {effectiveView==="video" && siteSettings && <VideoManagerPanel initial={siteSettings} onSaved={setSiteSettings}/>}
      {effectiveView==="verifikasi" && <section className="admin-view teacher-verification-view">
        <div className="admin-view-header"><div><p className="eyebrow">Verifikasi akun guru</p><h2>Persetujuan data guru</h2><p className="muted">Guru baru dapat masuk namun belum bisa mengelola bank soal sampai disetujui bahwa ia terdaftar dalam data pendidikan nasional. Bukti sanggah ikut ditampilkan untuk ditinjau ulang.</p></div><span className="muted">{verifications.total} guru perlu ditinjau</span></div>
        <div className="content-subtabs">{(["all","pending","rejected"] as const).map((tab)=><button key={tab} className={verifFilter===tab?"active":""} onClick={()=>setVerifFilter(tab)}>{tab==="all"?"Semua":tab==="pending"?"Menunggu":"Ditolak / Sanggah"}</button>)}</div>
        <AdminTable headers={["Guru","No. KTP","Jenjang","Status","Bukti Sanggah","Cek SIMPKB","Aksi"]}>{verifications.items.map((item)=><tr key={item.id}><td><div>{item.name || item.email}<small>{item.email}</small></div></td><td><strong>{item.teacher_ktp || "—"}</strong></td><td>{item.school_level}</td><td><span className={`status-pill ${item.status}`}>{item.status==="pending"?"Menunggu":"Ditolak"}</span></td><td style={{maxWidth:240}}>{item.status==="rejected" ? <div className="verification-proof-cell">{item.appeal_image ? <button type="button" className="table-action appeal-proof-btn" onClick={()=>setAppealPreview(item)}><DataImage className="appeal-proof-thumb" src={item.appeal_image} alt="Bukti sanggah" width={120} height={76}/><span>Perbesar</span></button> : <span className="muted">Belum sanggah</span>}{item.rejection_reason && <small className="muted">Penolakan: {item.rejection_reason}</small>}</div> : item.appeal_image ? <div className="verification-proof-cell"><button type="button" className="table-action appeal-proof-btn" onClick={()=>setAppealPreview(item)}><DataImage className="appeal-proof-thumb" src={item.appeal_image} alt="Bukti sanggah" width={120} height={76}/><span>Perbesar</span></button><small className="muted">Sanggah diajukan</small></div> : <span className="muted">—</span>}</td><td><div className="simpkb-check-cell"><span className={`status-pill simpkb-${item.simpkb_status}`}>{simpkbLabel(item.simpkb_status)}</span>{item.simpkb_image&&<button className="table-action" title="Lihat hasil pencarian SIMPKB" onClick={()=>setSIMPKBPreview(item)}>Lihat hasil</button>}<button className="table-action" disabled={saving!==""} onClick={()=>void checkSIMPKB(item)}>Cek ulang</button></div></td><td><div className="user-row-actions"><button className="table-action" disabled={saving!==""} onClick={()=>void approveTeacher(item)}>Setujui</button><button className="danger-action" disabled={saving!==""} onClick={()=>void rejectTeacher(item)}>Tolak</button></div></td></tr>)}{verifications.items.length===0&&<tr><td colSpan={7} className="empty-state">Tidak ada guru dalam kategori ini. Semua pendaftaran guru sudah selesai diverifikasi.</td></tr>}</AdminTable>
        <div className="pagination-row"><button className="table-action" disabled={verifications.page<=1||saving!==""} onClick={()=>setVerifPage((p)=>Math.max(1,p-1))}>‹ Sebelumnya</button><span className="muted">Hal {verifications.page} · {verifications.total} guru</span><button className="table-action" disabled={verifications.page*verifications.count>=verifications.total||saving!==""} onClick={()=>setVerifPage((p)=>p+1)}>Berikutnya ›</button></div>
      </section>}
      {effectiveView==="pengaturan" && siteSettings && <SiteSettingsPanel initial={siteSettings} onSaved={setSiteSettings}/>}
      {effectiveView==="profil" && <section className="admin-view"><div className="admin-view-header"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div><ProfileEditor user={user} onUpdated={onUserUpdate} /></section>}
    </section>
    {dialog && <CrudDialog dialog={dialog} exams={selectedPackageId ? packageExams : data?.exams??[]} levels={data?.levels??[]} mapels={data?.mapels??[]} academicYears={data?.academic_years??[]} kategoris={data?.kategoris??[]} kelas={data?.kelas??[]} siteSettings={siteSettings} defaultQuestionSubject={packageQuestions[0]?.subject_name} user={user} busy={saving!==""} onClose={()=>setDialog(null)} onSubmit={(value)=>void (async()=>{const successBox=dialog.kind==="package"?(dialog.item?"Paket soal berhasil diperbarui.":"Paket soal berhasil dibuat."):dialog.kind==="question"?"Soal berhasil disimpan.":dialog.kind==="user"?(dialog.item?"Pengguna berhasil diperbarui.":"Pengguna berhasil dibuat."):"Data berhasil disimpan.";await action(async()=>{
      if(dialog.kind==="user"&&"form" in value){const form=value.form;const role=String(form.get("role")) as User["role"];const schoolLevel=String(form.get("school_level")) as User["school_level"];if(dialog.item)await api.updateAdminUser(dialog.item.id,role,schoolLevel);else await api.createAdminUser({email:String(form.get("email")),password:String(form.get("password")),role,school_level:schoolLevel})}
      if(dialog.kind==="package"){if(!dialog.item&&"bundle" in value){const input=value.bundle;await api.createAdminPackageBundle(input)}else if("update" in value&&dialog.item){const {form,exam}=value.update;const examType=String(form.get("exam_type")) as AdminPackage["exam_type"];const isCbt=examType==="cbt";const startDateRaw=form.get("start_date");const endDateRaw=form.get("end_date");await api.updateAdminPackage(dialog.item.id,{title:String(form.get("title")),kode:String(form.get("kode")),description:String(form.get("description")),price:Number(form.get("price")),validity_days:Number(form.get("validity_days")),status:String(form.get("status")) as Status,jenjang:String(form.get("jenjang")) as User["school_level"],exam_type:examType,cbt_token:String(form.get("cbt_token")??""),start_date:isCbt&&startDateRaw?new Date(String(startDateRaw)+"T00:00:00Z").toISOString():undefined,end_date:isCbt&&endDateRaw?new Date(String(endDateRaw)+"T23:59:59Z").toISOString():undefined,kategori_id:String(form.get("kategori_id")),kelas_id:String(form.get("kelas_id"))});const examTarget=data?.exams.find((item)=>item.package_id===dialog.item!.id);if(examTarget)await api.updateAdminExam(examTarget.id,{package_id:examTarget.package_id,title:exam.title,mapel_id:exam.mapel_id,tahun_ajaran_id:exam.tahun_ajaran_id,duration_minutes:exam.duration_minutes,total_questions:exam.total_questions,passing_score:exam.passing_score,status:exam.status})}}
      if(dialog.kind==="question"&&"form" in value){const form=value.form;const input={exam_id:String(form.get("exam_id")),subject_name:String(form.get("subject_name")),content_text:String(form.get("content_text")),question_type:String(form.get("question_type")) as AdminQuestion["question_type"],presentation_type:String(form.get("presentation_type")) as AdminQuestion["presentation_type"],group_code:String(form.get("group_code")),stimulus_text:String(form.get("stimulus_text")),question_image_url:String(form.get("question_image_url")),stimulus_image_url:String(form.get("stimulus_image_url")),category_labels:JSON.parse(String(form.get("category_labels_json"))) as string[],options:JSON.parse(String(form.get("options_json"))) as {key:string;content:string;image_url?:string}[],correct_answer:String(form.get("correct_answer")),score_weight:Number(form.get("score_weight")),explanation_text:String(form.get("explanation_text")),status:String(form.get("status")) as Status};if(dialog.item)await api.updateAdminQuestion(dialog.item.id,input);else await api.createAdminQuestion(input)}
      if(dialog.kind==="master"&&"form" in value){const nama=String(value.form.get("nama"));if(dialog.item)await api.updateMaster(dialog.category,dialog.item.id,nama);else await api.createMaster(dialog.category,nama)}
      },successBox);})()}/>}
    {simpkbPreview && <div className="modal-backdrop" role="presentation" onMouseDown={(event)=>{if(event.target===event.currentTarget)setSIMPKBPreview(null)}}><div className="card checkout-modal simpkb-preview" role="dialog" aria-modal="true" onMouseDown={(event)=>event.stopPropagation()}><button type="button" className="modal-close" onClick={()=>setSIMPKBPreview(null)}>×</button><p className="eyebrow">Hasil pencarian SIMPKB</p><h2>{simpkbPreview.name || simpkbPreview.email}</h2><p className="muted">NIK: {simpkbPreview.teacher_ktp} · {simpkbLabel(simpkbPreview.simpkb_status)}</p>{simpkbPreview.simpkb_image ? <DataImage className="simpkb-preview-img" src={simpkbPreview.simpkb_image} alt="Screenshot hasil pencarian SIMPKB" width={1200} height={1600}/> : <p className="muted">Belum ada gambar.</p>}</div></div>}
    {appealPreview && <div className="modal-backdrop" role="presentation" onMouseDown={(event)=>{if(event.target===event.currentTarget)setAppealPreview(null)}}><div className="card checkout-modal simpkb-preview" role="dialog" aria-modal="true" onMouseDown={(event)=>event.stopPropagation()}><button type="button" className="modal-close" onClick={()=>setAppealPreview(null)}>×</button><p className="eyebrow">Bukti sanggah</p><h2>{appealPreview.name || appealPreview.email}</h2><p className="muted">NIK: {appealPreview.teacher_ktp} · Sanggah atas penolakan</p>{appealPreview.appeal_image ? <DataImage className="simpkb-preview-img" src={appealPreview.appeal_image} alt="Bukti sanggah guru" width={1200} height={1600}/> : <p className="muted">Belum ada gambar.</p>}</div></div>}
    {verifyConfirmItem && createPortal(<div className="verify-modal-backdrop" role="presentation" onMouseDown={(event)=>{if(event.target===event.currentTarget)setVerifyConfirmItem(null)}}><section className="verify-modal" role="dialog" aria-modal="true" aria-labelledby="verify-approve-title" onMouseDown={(event)=>event.stopPropagation()}><button type="button" className="verify-modal-close" disabled={saving!==""} onClick={()=>setVerifyConfirmItem(null)}>×</button><div className="verify-modal-icon verify-modal-icon--success" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 13l4 4L19 7"/></svg></div><p className="eyebrow">Setujui pendaftaran</p><h2 id="verify-approve-title">Setujui guru ini?</h2><p className="muted">Akun <strong>{verifyConfirmItem.name || verifyConfirmItem.email}</strong> akan langsung dapat mengakses panel guru dan mengelola bank soal.</p><div className="verify-modal-actions"><button type="button" className="verify-cancel" disabled={saving!==""} onClick={()=>setVerifyConfirmItem(null)}>Batal</button><button type="button" className="verify-confirm-approve" disabled={saving!==""} onClick={()=>void confirmApproveTeacher()}>{saving==="mutation"?"Memproses...":"Ya, setujui"}</button></div></section></div>, document.body)}
    {verifyRejectItem && createPortal(<div className="verify-modal-backdrop" role="presentation" onMouseDown={(event)=>{if(event.target===event.currentTarget)setVerifyRejectItem(null)}}><section className="verify-modal" role="dialog" aria-modal="true" aria-labelledby="verify-reject-title" onMouseDown={(event)=>event.stopPropagation()}><button type="button" className="verify-modal-close" disabled={saving!==""} onClick={()=>setVerifyRejectItem(null)}>×</button><div className="verify-modal-icon verify-modal-icon--danger" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M18 6L6 18M6 6l12 12"/></svg></div><p className="eyebrow">Tolak pendaftaran</p><h2 id="verify-reject-title">Tolak guru ini?</h2><p className="muted">Alasan penolakan akan ditampilkan pada halaman guru beserta tombol sanggah.</p><label className="verify-label">Alasan penolakan <span>(wajib)</span></label><textarea className="verify-textarea" placeholder="Tuliskan alasan penolakan..." value={verifyRejectReason} onChange={(event)=>setVerifyRejectReason(event.target.value)} rows={3} autoFocus/><div className="verify-modal-actions"><button type="button" className="verify-cancel" disabled={saving!==""} onClick={()=>setVerifyRejectItem(null)}>Batal</button><button type="button" className="verify-confirm-reject" disabled={saving!==""||!verifyRejectReason.trim()} onClick={()=>void confirmRejectTeacher()}>{saving==="mutation"?"Memproses...":"Ya, tolak"}</button></div></section></div>, document.body)}
    {removeTarget && <ConfirmModal title={removeTarget.kind==="user"?"Hapus pengguna":removeTarget.kind==="package"?"Hapus paket soal":"Hapus soal"} message={removeTarget.kind==="package"?`Paket soal "${removeTarget.label}" beserta seluruh ujian dan soal di dalamnya akan dihapus permanen.${removeTarget.isCBT?" Untuk paket CBT, data riwayat pengerjaan siswa juga akan dihapus.":""}`:removeTarget.kind==="user"?`Akun ${removeTarget.label} akan dihapus permanen beserta seluruh datanya. Tindakan ini tidak dapat dibatalkan.`:`Soal "${removeTarget.label}" akan dihapus permanen dari bank soal.`} busy={saving==="mutation"} onClose={()=>setRemoveTarget(null)} onConfirm={()=>void confirmRemove()}/>}
    {bulkDeleteTarget && <ConfirmModal title={bulkDeleteTarget.mode==="all"?"Hapus semua soal":"Hapus soal terpilih"} message={bulkDeleteTarget.mode==="all"?`Seluruh ${bulkDeleteTarget.count} soal pada paket "${selectedPackage?.title??""}" akan dihapus permanen. Tindakan ini tidak dapat dibatalkan.`:`${bulkDeleteTarget.count} soal terpilih akan dihapus permanen dari bank soal. Tindakan ini tidak dapat dibatalkan.`} busy={saving==="mutation"} onClose={()=>setBulkDeleteTarget(null)} onConfirm={()=>void confirmBulkDeleteQuestions()}/>}
    {toast&&<div className="success-toast" role="status"><span>✓</span><p>{toast}</p><button type="button" aria-label="Tutup notifikasi" onClick={()=>setToast("")}>×</button></div>}
  </div></main>;
}

function CrudDialog({dialog,exams,levels,mapels,academicYears,kategoris,kelas,siteSettings,defaultQuestionSubject,user,busy,onClose,onSubmit}:{dialog:Exclude<Dialog,null>;exams:AdminExam[];levels:string[];mapels:MasterItem[];academicYears:MasterItem[];kategoris:MasterItem[];kelas:MasterItem[];siteSettings:SiteSettings|null;defaultQuestionSubject?:string;user:User;busy:boolean;onClose:()=>void;onSubmit:(value:DialogSubmit)=>void}){
  const [options,setOptions] = useState<{key:string;content:string}[]>(dialog.kind==="question"?(dialog.item?.options??[{key:"A",content:""},{key:"B",content:""}]):[]);
  const [pkgExamType,setPkgExamType] = useState<AdminPackage["exam_type"]>(dialog.kind==="package"?(dialog.item?.exam_type??dialog.defaultExamType??"sell"):"sell");
  const [cbtToken,setCbtToken] = useState<string>(dialog.kind==="package"?(dialog.item?.cbt_token??""):"");
  const [pkgPrice,setPkgPrice] = useState<string>(dialog.kind==="package"?String(dialog.item?.price??0):"0");
  const [pkgStartDate,setPkgStartDate] = useState<string>(dialog.kind==="package"?(dialog.item?.start_date?.slice(0,10)??""):"");
  const [pkgEndDate,setPkgEndDate] = useState<string>(dialog.kind==="package"?(dialog.item?.end_date?.slice(0,10)??""):"");
  function generateCBTToken(){const alphabet="ABCDEFGHJKLMNPQRSTUVWXYZ23456789";let out="";for(let index=0;index<6;index++)out+=alphabet[Math.floor(Math.random()*alphabet.length)];return out;}
  const bundleRef = useRef<{getValues:()=>ExamEntry}>(null);
  const questionExam = dialog.kind==="question" ? exams.find((item)=>item.id===dialog.item?.exam_id)??exams[0] : undefined;
  const questionSubject = dialog.kind==="question" ? mapels.find((item)=>item.id===questionExam?.mapel_id)?.nama??dialog.item?.subject_name??defaultQuestionSubject??"" : "";
  const missingQuestionContext = dialog.kind==="question"&&(!questionExam||!questionSubject);
  function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    if(dialog.kind==="question") form.set("options_json", JSON.stringify(options));
    if(dialog.kind==="package"&&bundleRef.current){const exam=bundleRef.current.getValues();const isCbt=pkgExamType==="cbt";const pkg={title:String(form.get("title")),kode:isCbt?"":String(form.get("kode")??""),description:String(form.get("description")),price:isCbt?0:Number(form.get("price")),validity_days:isCbt?0:Number(form.get("validity_days")),status:isCbt?"active":(String(form.get("status")??"active") as Status),jenjang:String(form.get("jenjang")) as User["school_level"],exam_type:String(form.get("exam_type")) as AdminPackage["exam_type"],cbt_token:String(form.get("cbt_token")??""),start_date:isCbt&&pkgStartDate?new Date(pkgStartDate+"T00:00:00Z").toISOString():undefined,end_date:isCbt&&pkgEndDate?new Date(pkgEndDate+"T23:59:59Z").toISOString():undefined,kategori_id:String(form.get("kategori_id")),kelas_id:String(form.get("kelas_id"))};if(!dialog.item)onSubmit({bundle:{package:pkg,exam:{...exam,title:String(form.get("jenjang"))}}});else onSubmit({update:{form,exam}})}
    else onSubmit({form});
  }
  return <div className="modal-backdrop"><form className={`card admin-form ${dialog.kind==="question"?"question-dialog":""}`} onSubmit={submit}><button type="button" className="modal-close" onClick={onClose}>x</button><h2>{"item" in dialog&&dialog.item?"Edit":"Tambah"} {dialog.kind==="user"?"User":dialog.kind==="package"?"Paket Soal":dialog.kind==="master"?masterLabel(dialog.category):"Soal"}</h2>
    {dialog.kind==="user"&&<><label>Email</label><input name="email" type="email" defaultValue={dialog.item?.email} disabled={Boolean(dialog.item)} required/>{!dialog.item&&<><label>Password</label><input name="password" type="password" minLength={8} required/></>}<label>Peran</label><select name="role" defaultValue={dialog.item?.role??"student"} disabled={dialog.item?.id===user.id}><option value="student">Siswa</option><option value="teacher">Guru</option><option value="admin">Operator</option><option value="finance">Finance</option><option value="affiliate">Affiliate</option><option value="owner">Pemilik</option></select>{dialog.item?.id===user.id&&<input type="hidden" name="role" value={dialog.item.role}/>}<label>Jenjang</label><select name="school_level" defaultValue={dialog.item?.school_level??"SMA"}><option>SD</option><option>SMP</option><option value="SMA">SMA / SMK</option></select>{dialog.item&&<p className="form-helper">Email tidak dapat diubah. Gunakan formulir ini untuk memperbarui peran dan jenjang pengguna.</p>}</>}
    {dialog.kind==="package"&&<><label>Nama paket</label><input name="title" defaultValue={dialog.item?.title} required/><label>Jenis ujian</label><select name="exam_type" value={pkgExamType} onChange={(event)=>{const next=event.target.value as AdminPackage["exam_type"];setPkgExamType(next);if(next==="cbt"&&!cbtToken)setCbtToken(generateCBTToken());}} required><option value="sell">Sell</option><option value="cbt">CBT</option></select>{pkgExamType==="cbt"&&<div className="form-grid"><div><label>Kode rahasia (token CBT)</label><input name="cbt_token" value={cbtToken} onChange={(event)=>setCbtToken(event.target.value.toUpperCase())} maxLength={8} placeholder="cth: 9K4PT2" required/><p className="form-helper">Siswa wajib memasukkan kode ini saat memulai ujian CBT. Harga otomatis menjadi Gratis (Rp 0).</p></div><div><label>Acak token</label><button type="button" className="button" onClick={()=>setCbtToken(generateCBTToken())}>Generate ulang</button></div></div>}<label>Kategori tes</label><select name="kategori_id" defaultValue={dialog.item?.kategori_id??kategoris[0]?.id??""} required>{kategoris.map((item)=><option key={item.id} value={item.id}>{item.nama}</option>)}</select>{!kategoris.length&&<p className="form-helper">Belum ada kategori. Tambahkan lewat Master Data - Kategori Tes.</p>}<label>Keterangan kelas</label><select name="kelas_id" defaultValue={dialog.item?.kelas_id??kelas[0]?.id??""} required>{kelas.map((item)=><option key={item.id} value={item.id}>{item.nama}</option>)}</select>{!kelas.length&&<p className="form-helper">Belum ada kelas. Tambahkan lewat Master Data - Kelas.</p>}{pkgExamType==="cbt"&&<div className="form-grid"><div><label>Tanggal mulai aktif</label><input name="start_date" type="date" value={pkgStartDate} onChange={(event)=>setPkgStartDate(event.target.value)} required/><p className="form-helper">Ujian CBT tersedia mulai tanggal ini.</p></div><div><label>Tanggal berakhir aktif</label><input name="end_date" type="date" value={pkgEndDate} onChange={(event)=>setPkgEndDate(event.target.value)} min={pkgStartDate||undefined} required/><p className="form-helper">Setelah tanggal ini paket otomatis nonaktif.</p></div></div>}<div className="form-grid">{pkgExamType!=="cbt"&&<div><label>Kode soal</label><input name="kode" defaultValue={dialog.item?.kode} placeholder="cth: TKA-2026-SMA" maxLength={50} required/></div>}<div><label>Jenjang</label><select name="jenjang" defaultValue={dialog.item?.jenjang??"SMA"} required>{levels.map((level)=><option key={level} value={level}>{level}</option>)}</select></div></div><label>Pembuat paket soal</label><input value={user.email} disabled/><label>Deskripsi</label><textarea name="description" defaultValue={dialog.item?.description}/>{pkgExamType==="cbt"&&<p className="form-helper">Harga otomatis Gratis (Rp 0) dan langsung dipublikasikan aktif tanpa persetujuan. Masa aktif dihitung dari rentang tanggal di atas.</p>}{pkgExamType!=="cbt"&&<div className="form-grid"><div><label>Harga</label><div className="price-input"><span>Rp</span><input name="price" type="number" min="0" step="1000" value={pkgPrice} readOnly={false} onChange={(event)=>{const value=event.target.value;setPkgPrice(value);event.target.setCustomValidity(value!==""&&Number(value)>0&&Number(value)%1000!==0?"Harga harus kelipatan 1.000":"")}} required/></div></div><div><label>Masa aktif (hari)</label><input name="validity_days" type="number" min="1" defaultValue={dialog.item?.validity_days??siteSettings?.default_package_validity_days??7} required/></div></div>}{pkgExamType!=="cbt"&&<><label>Status</label><select name="status" defaultValue={dialog.item?.status??"active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></>}<hr/><PackageExamFields ref={bundleRef} mapels={mapels} academicYears={academicYears} defaults={siteSettings?{duration_minutes:siteSettings.default_exam_duration_minutes,total_questions:siteSettings.default_exam_total_questions,passing_score:siteSettings.default_passing_score}:undefined} defaultExam={dialog.item?exams.find((item)=>item.package_id===dialog.item?.id):undefined}/></>}
    {dialog.kind==="question"&&<><input type="hidden" name="exam_id" value={questionExam?.id??""} readOnly/><input type="hidden" name="subject_name" value={questionSubject} readOnly/>{missingQuestionContext&&<p className="error">Lengkapi mata pelajaran melalui Edit paket sebelum menambah soal.</p>}<TKAQuestionFields item={dialog.item} options={options} setOptions={setOptions}/><div className="form-grid"><div><label>Bobot soal</label><input name="score_weight" type="number" min="0.001" step="0.001" defaultValue={dialog.item?.score_weight??1} required/></div><div><label>Status</label><select name="status" defaultValue={dialog.item?.status??"active"}><option value="active">Aktif</option><option value="inactive">Nonaktif</option></select></div></div><label>Pembahasan</label><textarea name="explanation_text" defaultValue={dialog.item?.explanation_text}/></>}
    {dialog.kind==="master"&&<><label>Nama</label><input name="nama" defaultValue={dialog.item?.nama} required/></>}
    <button className="button full" disabled={busy||missingQuestionContext} type="submit">{busy?"Menyimpan...":"Simpan"}</button>
  </form></div>;
}
function ViewHeader({title,onAdd,templates=false}:{title:string;onAdd:()=>void;templates?:boolean}){return <div className="admin-view-header"><h2>{title}</h2><div className="template-actions">{templates&&<><a className="button secondary" href="/templates/template-bank-soal-tka.docx" download>Template Word</a><a className="button secondary" href="/templates/template-bank-soal-tka.xlsx" download>Template Excel</a></>}<button className="button" onClick={onAdd}>+ Tambah</button></div></div>}
function questionTypeLabel(type:AdminQuestion["question_type"]){return type==="multiple_choice"?"PGK MCMA":type==="category"?"PGK Kategori":type==="essay"?"Esai":"PG Sederhana"}
function Stat({label,value}:{label:string;value:number}) { return <article className="card admin-stat"><span>{label}</span><strong>{value}</strong></article>; }
function AdminTable({headers,children}:{headers:string[];children:React.ReactNode}) { return <div className="card admin-table-card"><div className="ranking-table-wrap"><table className="ranking-table"><thead><tr>{headers.map((header)=><th key={header}>{header}</th>)}</tr></thead><tbody>{children}</tbody></table></div></div>; }
function simpkbLabel(status: TeacherVerification["simpkb_status"]) { return { pending: "Belum dicek", checking: "Sedang dicek", found: "Ditemukan", not_found: "Tidak ditemukan", error: "Gagal dicek" }[status] ?? "—"; }
