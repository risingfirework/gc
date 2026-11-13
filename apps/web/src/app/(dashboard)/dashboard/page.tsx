"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";
import { APIError, api, CBTLookupPackage, GlobalRanking, MyTransaction, OwnedPackage, Package, PackageExam, PendingTransaction, SiteSettings, tokenStore, User, UserPricingPolicy } from "@/services/api";
import AdminDashboard from "@/components/AdminDashboard";
import TeacherDashboard from "@/components/TeacherDashboard";
import AffiliateDashboard from "@/components/AffiliateDashboard";
import ProfileEditor from "@/components/ProfileEditor";
import StudentMenu from "@/components/StudentMenu";
import NotificationBell from "@/components/NotificationBell";
import TestimonialSlider from "@/components/TestimonialSlider";
import HeroSlider from "@/components/HeroSlider";
import CatalogStats from "@/components/CatalogStats";
import BrandLogo from "@/components/BrandLogo";
import EmptyState from "@/components/EmptyState";
import StudentBottomNav from "@/components/StudentBottomNav";
import Toast, { ToastState } from "@/components/Toast";
import { Skeleton, SkeletonExamRows, SkeletonOwnedGrid, SkeletonPackageGrid } from "@/components/Skeleton";
import { VideoSection } from "@/components/YouTubeEmbed";

const rupiah = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
type DashboardView = "belajar" | "katalog" | "ranking" | "video" | "profil" | "cbt";
type RankingLevel = NonNullable<User["school_level"]>;
function parseDashboardView(value: string | null): DashboardView | undefined {
  return value === "belajar" || value === "katalog" || value === "ranking" || value === "video" || value === "profil" || value === "cbt" ? value : undefined;
}
function priceAfterDiscount(price: number, discountPercent: number) {
  return Math.round(price * (100 - discountPercent)) / 100;
}
function PackageGrid({ items, ownedItems, onBuy, policy }: { items: Package[]; ownedItems: OwnedPackage[]; onBuy: (item: Package) => void; policy?: UserPricingPolicy }) {
  const ownedPackageIDs = new Set(ownedItems.map((item) => item.id));
  return <div className="admin-package-grid student-catalog-grid">{items.map((item) => {
    const isOwned = ownedPackageIDs.has(item.id);
    const hasDiscount = item.price > 0 && (policy?.discount_percent ?? 0) > 0;
    const finalPrice = priceAfterDiscount(item.price, policy?.discount_percent ?? 0);
    return <article className={`card admin-package-card ${isOwned ? "catalog-owned" : ""}`} key={item.id}>
      <div className="admin-package-card-top"><span className="discount">{isOwned ? "SUDAH DIMILIKI" : item.price === 0 ? "GRATIS" : hasDiscount ? `DISKON ${policy?.discount_percent}%` : `AKTIF ${item.validity_days} HARI`}</span><span className="package-code">{item.kode}</span></div>
      <span className="package-jenjang">{item.jenjang}</span>
      <h3>{item.title}</h3>
      <p>{item.description}</p>
      <strong>{item.price === 0 ? "Gratis" : rupiah.format(finalPrice)}</strong>
      <small>{item.exam_count} ujian · {item.question_count} soal</small>
      {item.publisher_email && <small className="package-publisher">Pembuat: {item.publisher_email}</small>}
      <CatalogStats sales={item.sales_count} views={item.view_count}/>
      <button className="button full" disabled={isOwned || policy?.account_active === false} onClick={() => onBuy(item)}>{isOwned ? "Paket sudah dimiliki" : item.price === 0 ? "Ambil Gratis" : policy?.account_active === false ? "Pembelian ditahan" : "Beli paket"}</button>
    </article>;
  })}</div>;
}
function OwnedPackageGrid({ items, examsByPackage, onStart, onViewResult }: { items: OwnedPackage[]; examsByPackage: Record<string, PackageExam[] | undefined>; onStart: (examID: string) => void; onViewResult: (attemptID: string) => void }) {
  return <div className="grid owned-grid">{items.map((item) => {
    const exams = examsByPackage[item.id];
    return <article className="card package-card owned" key={item.id}><span className="discount">DIMILIKI</span><h3>{item.title}</h3><p>{item.description}</p><div className="package-meta"><span>Berlaku hingga {new Date(item.expired_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</span><span>Dibeli {new Date(item.paid_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</span></div>{exams && exams.length > 0 ? <div className="package-exams">{exams.map((exam) => <div className="exam-row" key={exam.id}><div className="exam-info"><strong>{exam.title}</strong><span>{exam.total_questions} soal · {exam.duration_minutes} menit</span></div>{exam.user_exam_id ? (
  <div>
    {exam.score_released === false
      ? <span className="exam-score">Menunggu nilai dirilis</span>
      : <>
        <span className="exam-score">Nilai: {exam.total_score?.toFixed(0) ?? "—"} · {(exam.total_score ?? 0) >= exam.passing_score ? "Lulus" : "Belum lulus"} · {exam.finished_at ? new Date(exam.finished_at).toLocaleDateString("id-ID") : "—"}</span>
        <button className="button secondary" onClick={() => onViewResult(exam.user_exam_id!)}>{exam.review_available === false ? "Lihat nilai" : "Lihat analitik"}</button>
      </>}
  </div>
) : exam.exam_type === "cbt" ? (
  <span className="exam-score">Masukkan token di menu Ujian CBT</span>
) : (
  <button className="button" onClick={() => onStart(exam.id)}>Mulai ujian</button>
)}</div>)}</div> : (exams === undefined
    ? <SkeletonExamRows count={2}/>
    : <EmptyState variant="plain" icon="exam" title="Belum ada ujian" description="Paket ini belum memiliki ujian aktif. Cek kembali nanti atau hubungi pembuat paket."/>)}</article>;
  })}</div>;
}
function findNextExam(packages: OwnedPackage[], examsByPackage: Record<string, PackageExam[] | undefined>) {
  for (const pkg of packages) {
    const exams = examsByPackage[pkg.id];
    const exam = exams?.find((item) => !item.user_exam_id && item.exam_type !== "cbt");
    if (exam) return { pkg, exam };
  }
  return null;
}
function formatDuration(seconds: number) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
}
function CBTLookupPanel({ token, onToken, onLookup }: { token: string; onToken: (value: string) => void; onLookup: () => void }) {
  return <div className="card cbt-lookup-card">
    <p className="eyebrow">Token ujian CBT</p>
    <h3>Mulai ujian dengan token</h3>
    <p className="muted">Masukkan token yang diberikan oleh pembuat paket untuk menemukan ujian milikmu.</p>
    <div className="cbt-token-row">
      <input type="text" value={token} onChange={(event) => onToken(event.target.value.toUpperCase())} placeholder="Contoh: 9K4PT2" maxLength={20} autoComplete="off"/>
      <button className="button" onClick={onLookup}>Cari ujian</button>
    </div>
  </div>;
}
function CBTResultPanel({ packages, onSelect, onReview }: { packages: CBTLookupPackage[]; onSelect: (examID: string) => void; onReview: (attemptID: string) => void }) {
  if (packages.length === 0) return <EmptyState icon="search" title="Token tidak cocok" description="Tidak ada paket ujian aktif yang cocok dengan token tersebut. Periksa kembali token dari gurumu, lalu coba lagi."/>;
  return <div className="package-exams">{packages.map((item) => <article className="card package-card owned cbt-result-card" key={item.id}>
    <span className="discount">PAKET CBT</span><h3>{item.title}</h3><p>{item.kode} · {item.jenjang}{item.price === 0 ? " · Gratis" : ""}</p>
    {item.exams.length === 0
      ? <EmptyState variant="plain" icon="exam" title="Belum ada ujian aktif" description="Paket ini belum memiliki ujian yang bisa dikerjakan."/>
      : <div className="package-exams">{item.exams.map((exam) => <div className="exam-row" key={exam.exam_id}><div className="exam-info"><strong>{exam.title}</strong><span>{exam.total_questions} soal · {exam.duration_minutes} menit · Syarat lulus {exam.passing_score.toFixed(0)}</span></div>{exam.submitted
        ? exam.score_released === false
          ? <span className="exam-score">Menunggu nilai dirilis</span>
          : exam.user_exam_id
            ? <button className="button secondary" onClick={() => onReview(exam.user_exam_id!)}>{exam.review_available === false ? "Lihat nilai" : "Lihat analitik"}</button>
            : <span className="exam-score">Sudah dikerjakan</span>
        : <button className="button" onClick={() => onSelect(exam.exam_id)}>Mulai ujian</button>}</div>)}</div>}
  </article>)}</div>;
}

export default function Dashboard() {
  return <Suspense fallback={<main><div className="container"><div className="exam-loader"/><p>Memuat panel siswa...</p></div></main>}><DashboardContent/></Suspense>;
}

function DashboardContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [user, setUser] = useState<User | null>(null);
  const activeView = parseDashboardView(searchParams.get("view")) ?? "belajar";
  const [packages, setPackages] = useState<Package[]>([]);
  const [myPackages, setMyPackages] = useState<OwnedPackage[]>([]);
  const [examsByPackage, setExamsByPackage] = useState<Record<string, PackageExam[] | undefined>>({});
  const [ranking, setRanking] = useState<GlobalRanking[]>([]);
  const [rankingLevel, setRankingLevel] = useState<RankingLevel>("SD");
  const [rankingMode, setRankingMode] = useState<"score" | "activity">("score");
  const [myTransactions, setMyTransactions] = useState<MyTransaction[]>([]);
  const [pendingPayments, setPendingPayments] = useState<PendingTransaction[]>([]);
  const [selected, setSelected] = useState<Package>();
  const [pricingPolicy, setPricingPolicy] = useState<UserPricingPolicy>();
  const [method, setMethod] = useState<"qris" | "virtual_account" | "e_wallet">("qris");
  const [referralCode, setReferralCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [error, setError] = useState("");
  const [siteSettings,setSiteSettings]=useState<SiteSettings>();
  const [cbtToken, setCbtToken] = useState("");
  const [cbtResults, setCbtResults] = useState<CBTLookupPackage[] | null>(null);
  const [cbtSearching, setCbtSearching] = useState(false);
  const [userLoading, setUserLoading] = useState(true);
  const [packagesLoading, setPackagesLoading] = useState(true);
  const [myPackagesLoading, setMyPackagesLoading] = useState(true);
  const [toast, setToast] = useState<ToastState>(null);
  const closeToast = useCallback(() => setToast(null), []);
  const catalogPackages = user ? packages.filter((item) => item.jenjang === user.school_level && item.exam_type !== "cbt") : [];
  const nextExam = findNextExam(myPackages, examsByPackage);

  useEffect(() => {
    if (!tokenStore.hasToken()) {
      router.replace("/login");
      return;
    }
    api.currentUser().then((current) => { setUser(current); setRankingLevel(current.school_level); }).catch(() => undefined).finally(() => setUserLoading(false));
    api.packages().then((response) => setPackages(response.data)).catch((reason) => setError(reason instanceof Error ? reason.message : "Katalog gagal dimuat.")).finally(() => setPackagesLoading(false));
    api.packagesMine().then(setMyPackages).catch(() => undefined).finally(() => setMyPackagesLoading(false));
    api.pricingPolicy().then(setPricingPolicy).catch(() => undefined);
    api.siteSettings().then(setSiteSettings).catch(()=>undefined);
  }, [router]);

  useEffect(() => {
    if (!tokenStore.hasToken() || myPackages.length === 0) return;
    let cancelled = false;
    Promise.allSettled(myPackages.map(async (pkg) => ({ id: pkg.id, exams: await api.packageExams(pkg.id) }))).then((results) => {
      if (cancelled) return;
      const next: Record<string, PackageExam[] | undefined> = {};
      for (const result of results) if (result.status === "fulfilled") next[result.value.id] = result.value.exams;
      setExamsByPackage(next);
    });
    return () => { cancelled = true; };
  }, [myPackages]);

  useEffect(() => {
    if (!tokenStore.hasToken()) return;
    api.globalRanking(rankingLevel, 1, 50, rankingMode === "activity" ? "activity" : undefined).then((result) => setRanking(result.items)).catch((reason) => setError(reason instanceof Error ? reason.message : "Ranking gagal dimuat."));
  }, [router, rankingLevel, rankingMode]);

  useEffect(() => {
    if (!tokenStore.hasToken()) return;
    api.myTransactions().then(setMyTransactions).catch(() => undefined);
    api.pendingTransactions().then(setPendingPayments).catch(() => undefined);
  }, [router]);

  async function logout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try { await api.logout(); } finally { router.replace("/"); }
  }

  async function downloadInvoice(transactionID: string) {
    try {
      const blob = await api.downloadInvoice(transactionID);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = "invoice.pdf";
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      setToast({ message: "Invoice berhasil diunduh.", tone: "success" });
    } catch (reason) {
      setToast({ message: reason instanceof APIError ? reason.message : "Invoice gagal diunduh.", tone: "error" });
    }
  }

  async function checkout() {
    if (!selected || loading) return;
    setLoading(true);
    setError("");
    const paymentWindow = window.open("about:blank", "tka-payment", "popup,width=520,height=760");
    try {
      const response = await api.checkout(selected.id, method, `${crypto.randomUUID()}-${selected.id}`, referralCode.trim() || undefined);
      if (paymentWindow) paymentWindow.location.replace(response.payment_url);
      else window.location.assign(response.payment_url);
      setSelected(undefined);
      api.pendingTransactions().then(setPendingPayments).catch(() => undefined);
    } catch (reason) {
      paymentWindow?.close();
      setToast({ message: reason instanceof APIError ? reason.message : "Checkout gagal dibuat.", tone: "error" });
    } finally { setLoading(false); }
  }

  function resumePayment(item: PendingTransaction) {
    const paymentWindow = window.open(item.payment_url, "tka-payment", "popup,width=520,height=760");
    if (!paymentWindow) window.location.assign(item.payment_url);
  }

  function openPackage(item:Package) {
    setSelected(item);
    setReferralCode("");
    void api.trackPackageView(item.id).then((viewCount)=>setPackages((current)=>current.map((pkg)=>pkg.id===item.id?{...pkg,view_count:viewCount}:pkg))).catch(()=>undefined);
  }

  function lookupCBT() {
    if (cbtToken.trim().length < 4 || cbtSearching) return;
    setCbtSearching(true);
    api.cbtLookup(cbtToken.trim()).then((results) => { setCbtResults(results); setError(""); }).catch((reason) => setToast({ message: reason instanceof Error ? reason.message : "Pencarian token gagal.", tone: "error" })).finally(() => setCbtSearching(false));
  }

  async function claimFree() {
    if (!selected || loading) return;
    setLoading(true);
    setError("");
    try {
      await api.claimFreePackage(selected.id);
      setSelected(undefined);
      setToast({ message: "Paket gratis berhasil diambil. Selamat belajar!", tone: "success" });
      api.packagesMine().then(setMyPackages).catch(() => undefined);
    } catch (reason) {
      setToast({ message: reason instanceof APIError ? reason.message : "Gagal mengambil paket gratis.", tone: "error" });
    } finally { setLoading(false); }
  }

  if (userLoading) {
    return <main className="student-shell"><div className="container"><section className="dashboard" aria-busy="true" aria-label="Memuat dashboard">
      <div className="dashboard-skeleton-head"><Skeleton className="skeleton-title"/><Skeleton className="skeleton-subtitle"/></div>
      <Skeleton className="skeleton-hero"/>
      <div className="section-heading"><div><Skeleton className="skeleton-eyebrow"/><Skeleton className="skeleton-title"/></div></div>
      <SkeletonOwnedGrid count={2}/>
    </section></div></main>;
  }
  if (user?.role === "admin" || user?.role === "owner" || user?.role === "finance") {
    return <AdminDashboard user={user} onLogout={() => void logout()} loggingOut={loggingOut} onUserUpdate={setUser}/>;
  }
  if (user?.role === "teacher") {
    return <TeacherDashboard user={user} onLogout={() => void logout()} loggingOut={loggingOut} onUserUpdate={setUser}/>;
  }
  if (user?.role === "affiliate") {
    return <AffiliateDashboard user={user} onLogout={() => void logout()} loggingOut={loggingOut} onUserUpdate={setUser}/>;
  }

  return <main><div className="container">
    <nav className="dashboard-nav">
      <Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/>{siteSettings?.platform_name??""}</Link>
      <span className="nav-actions"><NotificationBell/><StudentMenu active={activeView} loggingOut={loggingOut} onLogout={() => void logout()} /></span>
    </nav>

    <section className="dashboard">
      {error && <p className="error">{error}</p>}

      {activeView === "belajar" && <>
        <div className="panel-hero"><HeroSlider onStart={() => router.push("/dashboard?view=katalog")} compact slides={siteSettings?.hero_slides} intervalMS={siteSettings?.hero_interval_ms}/></div>
        {nextExam && <section className="resume-banner" aria-label="Ujian berikutnya">
          <span className="resume-banner-icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M5 4h14v17H5z"/><path d="M9 4.5V3h6v1.5M9 9h6M9 13h6"/><path d="m10.5 17 1.8 1.8L15.5 15"/></svg></span>
          <div className="resume-banner-copy"><p className="eyebrow">Ujian berikutnya</p><strong>{nextExam.exam.title}</strong><span>{nextExam.pkg.title} · {nextExam.exam.total_questions} soal · {nextExam.exam.duration_minutes} menit</span></div>
          <button className="button" type="button" onClick={() => router.push(`/cbt/${nextExam.exam.id}`)}>Lanjutkan ujian</button>
        </section>}
        <section className="dashboard-section panel-view">
          <div className="section-heading"><div><p className="eyebrow">Belajar saya</p><h2>Paket belajar aktif</h2></div></div>
          {myPackagesLoading && myPackages.length === 0
            ? <SkeletonOwnedGrid count={2}/>
            : myPackages.length === 0
              ? <EmptyState icon="book" title="Belum ada paket belajar" description="Kamu belum memiliki paket belajar. Jelajahi katalog untuk mulai berlatih dengan soal dan pembahasan." action={{ label: "Jelajahi katalog", href: "/dashboard?view=katalog" }}/>
              : <OwnedPackageGrid items={myPackages} examsByPackage={examsByPackage} onStart={(examID) => router.push(`/cbt/${examID}`)} onViewResult={(attemptID) => router.push(`/dashboard/results/${attemptID}`)} />}
        </section>
        <VideoSection videos={siteSettings?.youtube_videos ?? []} eyebrow="Video" title={siteSettings?.platform_name?`Tonton panduan belajar ${siteSettings.platform_name}`:"Tonton panduan belajar"} lead="Pelajari alur tryout, analisis nilai, dan tips biar makin siap."/>
        <section className="dashboard-section panel-view">
          <div className="section-heading"><div><p className="eyebrow">Pembelian</p><h2>Riwayat transaksi</h2></div></div>
          {myTransactions.length === 0 && pendingPayments.length === 0
            ? <EmptyState icon="cart" title="Belum ada transaksi" description="Pembelian paket dan invoice akan muncul di sini setelah kamu menyelesaikan pembayaran." action={{ label: "Lihat katalog", href: "/dashboard?view=katalog" }}/>
            : <div className="card transactions-table-card"><div className="transactions-table-wrap"><table className="transactions-table"><thead><tr><th>Invoice</th><th>Paket</th><th>Tanggal</th><th>Total</th><th>Status</th><th></th></tr></thead><tbody>
              {pendingPayments.map((item) => <tr key={item.id}><td><strong>{item.invoice_number}</strong></td><td>{item.package_title}</td><td>{item.expires_at ? `Sampai ${new Date(item.expires_at).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" })}` : "Menunggu"}</td><td>{rupiah.format(item.amount)}</td><td><span className="status-pill pending">Menunggu pembayaran</span></td><td><button type="button" className="button small-btn" onClick={() => resumePayment(item)}>Lanjutkan bayar</button></td></tr>)}
              {myTransactions.map((item) => <tr key={item.id}><td><strong>{item.invoice_number}</strong></td><td>{item.package_title}</td><td>{new Date(item.created_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</td><td>{rupiah.format(item.amount)}</td><td><span className={`status-pill ${item.status}`}>{item.status === "paid" ? "Lunas" : "Direfund"}</span></td><td><button type="button" className="button secondary small-btn" onClick={() => void downloadInvoice(item.id)}>Unduh invoice</button></td></tr>)}
            </tbody></table></div></div>}
        </section>
      </>}

      {activeView === "katalog" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Daftar katalog</p><h2>Paket jenjang {user?.school_level === "SMA" ? "SMA / SMK" : user?.school_level ?? "siswa"}</h2></div></div>
        {pricingPolicy?.account_active === false && <p className="error">Pembelian akun ini sedang ditahan oleh admin. Hubungi pengelola untuk informasi lebih lanjut.</p>}
        {!user
          ? <SkeletonPackageGrid count={3}/>
          : packagesLoading && catalogPackages.length === 0
            ? <SkeletonPackageGrid count={3}/>
            : catalogPackages.length === 0
              ? <EmptyState icon="catalog" title={`Belum ada paket aktif untuk jenjang ${user.school_level === "SMA" ? "SMA / SMK" : user.school_level}`} description="Paket baru akan tampil otomatis di sini setelah dipublikasikan oleh pengelola." action={{ label: "Lihat ranking", href: "/dashboard?view=ranking" }}/>
              : <PackageGrid items={catalogPackages} ownedItems={myPackages} onBuy={openPackage} policy={pricingPolicy} />}
      </section>}

      {activeView === "ranking" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Ranking global</p><h2>Peringkat persentil jenjang {rankingLevel}</h2></div><span className="muted">Maksimal 100 siswa</span></div>
        <div className="content-subtabs" style={{display:"flex",gap:12,alignItems:"center",flexWrap:"wrap"}}>
          <span style={{fontWeight:700,color:"var(--ink)"}}>Jenjang</span>
          {(["SD", "SMP", "SMA"] as const).map((tab) => <button key={tab} className={rankingLevel === tab ? "active" : ""} onClick={() => setRankingLevel(tab)}>{tab}</button>)}
          <span style={{width:1,height:20,background:"var(--border)",margin:"0 4px"}}/>
          <span style={{fontWeight:700,color:"var(--ink)"}}>Urutkan</span>
          <button className={rankingMode === "score" ? "active" : ""} onClick={() => setRankingMode("score")}>Persentil</button>
          <button className={rankingMode === "activity" ? "active" : ""} onClick={() => setRankingMode("activity")}>Aktivitas</button>
        </div>
        <div className="card ranking-card">
          {ranking.length === 0 ? <EmptyState variant="plain" icon="ranking" title="Ranking belum tersedia" description={`Belum ada hasil ujian jenjang ${rankingLevel} yang bisa diperingkat. Selesaikan satu ujian untuk masuk peringkat.`}/> : <div className="ranking-table-wrap"><table className="ranking-table"><thead><tr><th>Peringkat</th><th>Nama pengguna</th><th>Jenjang</th><th>Ujian Selesai</th><th>Persentil</th><th>Terbaik</th><th>Waktu</th><th>Tanggal</th></tr></thead><tbody>{ranking.map((entry) => <tr key={`${entry.rank}-${entry.display_name}`} className={entry.is_current_user ? "current-user" : ""}><td><span className={`rank-badge rank-${entry.rank}`}>{entry.rank}</span></td><td>{entry.display_name}{entry.is_current_user && <small> Anda</small>}</td><td>{entry.school_level}</td><td><strong>{entry.exams_done}</strong></td><td><strong>{entry.score.toFixed(1)}%</strong></td><td>{entry.best_percentile.toFixed(1)}%</td><td>{formatDuration(entry.duration_seconds)}</td><td>{new Date(entry.finished_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</td></tr>)}</tbody></table></div>}
        </div>
      </section>}

      {activeView === "video" && ((siteSettings?.youtube_videos ?? []).length === 0
          ? <section className="dashboard-section panel-view"><div className="section-heading"><div><p className="eyebrow">Video</p><h2>Video panduan & promosi</h2></div></div><EmptyState icon="video" title="Belum ada video" description="Pengelola belum mempublikasikan video panduan atau promosi. Coba cek lagi nanti."/></section>
          : <section className="dashboard-section panel-view"><VideoSection videos={siteSettings?.youtube_videos ?? []} eyebrow="Video" title="Video panduan & promosi" lead="Pelajari alur tryout, analisis nilai, dan tips biar makin siap."/></section>)}

      {activeView === "profil" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div>
        {user && <ProfileEditor user={user} onUpdated={setUser} />}
      </section>}

      {activeView === "cbt" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Ujian CBT</p><h2>Masukkan token ujian</h2></div></div>
        <CBTLookupPanel token={cbtToken} onToken={setCbtToken} onLookup={lookupCBT}/>
        {cbtSearching && <div className="card"><SkeletonExamRows count={3}/></div>}
        {cbtResults !== null && <CBTResultPanel packages={cbtResults} onSelect={(examID) => router.push(`/cbt/${examID}?token=${cbtToken.trim()}`)} onReview={(attemptID) => router.push(`/dashboard/results/${attemptID}`)}/>}
      </section>}

      <TestimonialSlider/>
    </section>

    {selected && (selected.price === 0
      ? <div className="modal-backdrop" role="presentation" onMouseDown={() => !loading && setSelected(undefined)}><section className="card checkout-modal" role="dialog" aria-modal="true" aria-labelledby="checkout-title" onMouseDown={(event) => event.stopPropagation()}>
        <button className="modal-close" aria-label="Tutup" onClick={() => setSelected(undefined)}>x</button><p className="eyebrow">Paket gratis</p><h2 id="checkout-title">{selected.title}</h2><p>{selected.description}</p><div className="checkout-total"><span>Total</span><strong>Gratis</strong></div>
        <p className="muted">Paket ini gratis dan akan langsung menjadi milikmu setelah dikonfirmasi.</p><button className="button full" disabled={loading} onClick={() => void claimFree()}>{loading ? "Memproses..." : "Ambil Gratis"}</button>
      </section></div>
      : <div className="modal-backdrop" role="presentation" onMouseDown={() => !loading && setSelected(undefined)}><section className="card checkout-modal" role="dialog" aria-modal="true" aria-labelledby="checkout-title" onMouseDown={(event) => event.stopPropagation()}>
        <button className="modal-close" aria-label="Tutup" onClick={() => setSelected(undefined)}>x</button><p className="eyebrow">Checkout aman</p><h2 id="checkout-title">{selected.title}</h2><div className="checkout-total"><span>Total pembayaran{(pricingPolicy?.discount_percent ?? 0) > 0 && <small>Diskon akun {pricingPolicy?.discount_percent}%</small>}</span><strong>{rupiah.format(priceAfterDiscount(selected.price, pricingPolicy?.discount_percent ?? 0))}</strong></div>
        <label htmlFor="payment-method">Preferensi pembayaran</label><select id="payment-method" value={method} onChange={(event) => setMethod(event.target.value as typeof method)}><option value="qris">QRIS</option><option value="virtual_account">Virtual Account</option><option value="e_wallet">E-Wallet</option></select><small className="muted">Metode yang tersedia tetap mengikuti kanal aktif pada halaman aman Xendit.</small>
        <label htmlFor="referral-code">Kode referral <small>(opsional)</small></label><input id="referral-code" type="text" value={referralCode} onChange={(event) => setReferralCode(event.target.value.toUpperCase())} maxLength={20} placeholder="Contoh: MITRA-XXXX" autoComplete="off"/>
        <p className="muted">Punya kode rujukan dari teman? Masukkan di sini agar temanmu mendapat komisi.</p>
        <p className="muted">Jendela pembayaran akan dibuka setelah invoice dibuat.</p><button className="button full" disabled={loading} onClick={() => void checkout()}>{loading ? "Membuat invoice..." : "Lanjut ke pembayaran"}</button>
      </section></div>)}
  </div>
  <StudentBottomNav active={activeView}/>
  <Toast toast={toast} onClose={closeToast}/>
  </main>;
}
