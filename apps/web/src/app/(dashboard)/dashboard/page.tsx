"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { APIError, api, GlobalRanking, OwnedPackage, Package, PackageExam, SiteSettings, tokenStore, User, UserPricingPolicy } from "@/services/api";
import AdminDashboard from "@/components/AdminDashboard";
import TeacherDashboard from "@/components/TeacherDashboard";
import ProfileEditor from "@/components/ProfileEditor";
import StudentMenu from "@/components/StudentMenu";
import TestimonialSlider from "@/components/TestimonialSlider";
import HeroSlider from "@/components/HeroSlider";
import CatalogStats from "@/components/CatalogStats";
import BrandLogo from "@/components/BrandLogo";

const rupiah = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });
type DashboardView = "belajar" | "katalog" | "ranking" | "profil";
type RankingLevel = NonNullable<User["school_level"]>;
function parseDashboardView(value: string | null): DashboardView | undefined {
  return value === "belajar" || value === "katalog" || value === "ranking" || value === "profil" ? value : undefined;
}
function priceAfterDiscount(price: number, discountPercent: number) {
  return Math.round(price * (100 - discountPercent)) / 100;
}
function PackageGrid({ items, ownedItems, onBuy, policy }: { items: Package[]; ownedItems: OwnedPackage[]; onBuy: (item: Package) => void; policy?: UserPricingPolicy }) {
  const ownedPackageIDs = new Set(ownedItems.map((item) => item.id));
  return <div className="grid">{items.map((item) => {
    const isOwned = ownedPackageIDs.has(item.id);
    const hasDiscount = item.price > 0 && (policy?.discount_percent ?? 0) > 0;
    const finalPrice = priceAfterDiscount(item.price, policy?.discount_percent ?? 0);
    return <article className={`card package-card ${isOwned ? "catalog-owned" : ""}`} key={item.id}><span className="discount">{isOwned ? "SUDAH DIMILIKI" : item.price === 0 ? "GRATIS" : hasDiscount ? `DISKON ${policy?.discount_percent}%` : `AKTIF ${item.validity_days} HARI`}</span><h3>{item.title}</h3><p>{item.description}</p><CatalogStats sales={item.sales_count} views={item.view_count}/><div className={`price ${hasDiscount ? "price-discounted" : ""}`}>{hasDiscount && <small>{rupiah.format(item.price)}</small>}{item.price === 0 ? "Gratis" : rupiah.format(finalPrice)}</div><button className="button full" disabled={isOwned || policy?.account_active === false} onClick={() => onBuy(item)}>{isOwned ? "Paket sudah dimiliki" : item.price === 0 ? "Ambil Gratis" : policy?.account_active === false ? "Pembelian ditahan" : "Beli paket"}</button></article>;
  })}</div>;
}
function OwnedPackageGrid({ items, examsByPackage, onStart, onViewResult }: { items: OwnedPackage[]; examsByPackage: Record<string, PackageExam[] | undefined>; onStart: (examID: string) => void; onViewResult: (attemptID: string) => void }) {
  return <div className="grid owned-grid">{items.map((item) => {
    const exams = examsByPackage[item.id];
    return <article className="card package-card owned" key={item.id}><span className="discount">DIMILIKI</span><h3>{item.title}</h3><p>{item.description}</p><div className="package-meta"><span>Berlaku hingga {new Date(item.expired_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</span><span>Dibeli {new Date(item.paid_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</span></div>{exams && exams.length > 0 ? <div className="package-exams">{exams.map((exam) => <div className="exam-row" key={exam.id}><div className="exam-info"><strong>{exam.title}</strong><span>{exam.total_questions} soal · {exam.duration_minutes} menit</span></div>{exam.user_exam_id ? (
  <div>
    <span className="exam-score">Nilai: {exam.total_score?.toFixed(0) ?? "—"} · {(exam.total_score ?? 0) >= exam.passing_score ? "Lulus" : "Belum lulus"} · {exam.finished_at ? new Date(exam.finished_at).toLocaleDateString("id-ID") : "—"}</span>
    <button className="button secondary" onClick={() => onViewResult(exam.user_exam_id!)}>Lihat analitik</button>
  </div>
) : (
  <button className="button" onClick={() => onStart(exam.id)}>Mulai ujian</button>
)}</div>)}</div> : <p className="empty-state">{exams === undefined ? "Memuat ujian..." : "Belum ada ujian di paket ini."}</p>}</article>;
  })}</div>;
}
function formatDuration(seconds: number) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${ss}` : `${m}:${ss}`;
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
  const [selected, setSelected] = useState<Package>();
  const [pricingPolicy, setPricingPolicy] = useState<UserPricingPolicy>();
  const [method, setMethod] = useState<"qris" | "virtual_account" | "e_wallet">("qris");
  const [loading, setLoading] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [error, setError] = useState("");
  const [siteSettings,setSiteSettings]=useState<SiteSettings>();
  const catalogPackages = user ? packages.filter((item) => item.jenjang === user.school_level) : [];

  useEffect(() => {
    if (!tokenStore.hasToken()) {
      router.replace("/login");
      return;
    }
    api.currentUser().then((current) => { setUser(current); setRankingLevel(current.school_level); }).catch(() => undefined);
    api.packages().then((response) => setPackages(response.data)).catch((reason) => setError(reason instanceof Error ? reason.message : "Katalog gagal dimuat."));
    api.packagesMine().then(setMyPackages).catch(() => undefined);
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
    api.globalRanking(rankingLevel).then(setRanking).catch((reason) => setError(reason instanceof Error ? reason.message : "Ranking gagal dimuat."));
  }, [router, rankingLevel]);

  async function logout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try { await api.logout(); } finally { router.replace("/"); }
  }

  async function checkout() {
    if (!selected || loading) return;
    setLoading(true);
    setError("");
    const paymentWindow = window.open("about:blank", "tka-payment", "popup,width=520,height=760");
    try {
      const response = await api.checkout(selected.id, method, `${crypto.randomUUID()}-${selected.id}`);
      if (paymentWindow) paymentWindow.location.replace(response.payment_url);
      else window.location.assign(response.payment_url);
      setSelected(undefined);
    } catch (reason) {
      paymentWindow?.close();
      setError(reason instanceof APIError ? reason.message : "Checkout gagal dibuat.");
    } finally { setLoading(false); }
  }

  function openPackage(item:Package) {
    setSelected(item);
    void api.trackPackageView(item.id).then((viewCount)=>setPackages((current)=>current.map((pkg)=>pkg.id===item.id?{...pkg,view_count:viewCount}:pkg))).catch(()=>undefined);
  }

  async function claimFree() {
    if (!selected || loading) return;
    setLoading(true);
    setError("");
    try {
      await api.claimFreePackage(selected.id);
      setSelected(undefined);
      api.packagesMine().then(setMyPackages).catch(() => undefined);
    } catch (reason) {
      setError(reason instanceof APIError ? reason.message : "Gagal mengambil paket gratis.");
    } finally { setLoading(false); }
  }

  if (user?.role === "admin") {
    return <AdminDashboard user={user} onLogout={() => void logout()} loggingOut={loggingOut} onUserUpdate={setUser}/>;
  }
  if (user?.role === "teacher") {
    return <TeacherDashboard user={user} onLogout={() => void logout()} loggingOut={loggingOut} onUserUpdate={setUser}/>;
  }

  return <main><div className="container">
    <nav className="dashboard-nav">
      <Link className="brand" href="/dashboard"><BrandLogo logoDataURL={siteSettings?.logo_data_url}/>{siteSettings?.platform_name??"TKA Juara"}</Link>
      <StudentMenu active={activeView} loggingOut={loggingOut} onLogout={() => void logout()} />
    </nav>

    <section className="dashboard">
      {error && <p className="error">{error}</p>}

      {activeView === "belajar" && <>
        <div className="panel-hero"><HeroSlider onStart={() => router.push("/dashboard?view=katalog")} compact slides={siteSettings?.hero_slides} intervalMS={siteSettings?.hero_interval_ms}/></div>
        <section className="dashboard-section panel-view">
          <div className="section-heading"><div><p className="eyebrow">Belajar saya</p><h2>Paket belajar aktif</h2></div></div>
          {myPackages.length === 0 ? <div className="card empty-state">Kamu belum memiliki paket belajar. Jelajahi katalog untuk mulai berlatih.</div> : <OwnedPackageGrid items={myPackages} examsByPackage={examsByPackage} onStart={(examID) => router.push(`/cbt/${examID}`)} onViewResult={(attemptID) => router.push(`/dashboard/results/${attemptID}`)} />}
        </section>
      </>}

      {activeView === "katalog" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Daftar katalog</p><h2>Paket jenjang {user?.school_level === "SMA" ? "SMA / SMK" : user?.school_level ?? "siswa"}</h2></div></div>
        {pricingPolicy?.account_active === false && <p className="error">Pembelian akun ini sedang ditahan oleh admin. Hubungi pengelola untuk informasi lebih lanjut.</p>}
        {!user
          ? <div className="card empty-state">Memuat katalog sesuai jenjang...</div>
          : catalogPackages.length === 0
            ? <div className="card empty-state">Belum ada paket aktif untuk jenjang {user.school_level === "SMA" ? "SMA / SMK" : user.school_level}.</div>
            : <PackageGrid items={catalogPackages} ownedItems={myPackages} onBuy={openPackage} policy={pricingPolicy} />}
      </section>}

      {activeView === "ranking" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Ranking global</p><h2>Nilai terbaik jenjang {rankingLevel}</h2></div><span className="muted">Maksimal 100 siswa</span></div>
        <div className="content-subtabs">{(["SD", "SMP", "SMA"] as const).map((tab) => <button key={tab} className={rankingLevel === tab ? "active" : ""} onClick={() => setRankingLevel(tab)}>{tab}</button>)}</div>
        <div className="card ranking-card">
          {ranking.length === 0 ? <p className="empty-state">Belum ada hasil ujian yang dapat diperingkat.</p> : <div className="ranking-table-wrap"><table className="ranking-table"><thead><tr><th>Peringkat</th><th>Nama pengguna</th><th>Jenjang</th><th>Nilai</th><th>Waktu</th><th>Tanggal</th></tr></thead><tbody>{ranking.map((entry) => <tr key={`${entry.rank}-${entry.display_name}`} className={entry.is_current_user ? "current-user" : ""}><td><span className={`rank-badge rank-${entry.rank}`}>{entry.rank}</span></td><td>{entry.display_name}{entry.is_current_user && <small> Anda</small>}</td><td>{entry.school_level}</td><td><strong>{entry.score.toFixed(2)}</strong></td><td>{formatDuration(entry.duration_seconds)}</td><td>{new Date(entry.finished_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })}</td></tr>)}</tbody></table></div>}
        </div>
      </section>}

      {activeView === "profil" && <section className="dashboard-section panel-view">
        <div className="section-heading"><div><p className="eyebrow">Profil</p><h2>Informasi akun</h2></div></div>
        {user && <ProfileEditor user={user} onUpdated={setUser} />}
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
        <label htmlFor="payment-method">Metode pembayaran</label><select id="payment-method" value={method} onChange={(event) => setMethod(event.target.value as typeof method)}><option value="qris">QRIS</option><option value="virtual_account">Virtual Account</option><option value="e_wallet">E-Wallet</option></select>
        <p className="muted">Jendela pembayaran akan dibuka setelah invoice dibuat.</p><button className="button full" disabled={loading} onClick={() => void checkout()}>{loading ? "Membuat invoice..." : "Lanjut ke pembayaran"}</button>
      </section></div>)}
  </div></main>;
}
