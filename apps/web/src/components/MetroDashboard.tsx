"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState, useSyncExternalStore, type KeyboardEvent as ReactKeyboardEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { api, Package, SiteSettings, Testimonial, tokenStore } from "@/services/api";
import { youtubeVideoID } from "@/components/YouTubeEmbed";
import styles from "./MetroDashboard.module.css";

type IconName =
  | "rocket"
  | "play"
  | "tag"
  | "quote"
  | "help"
  | "chart"
  | "user"
  | "headset"
  | "chat"
  | "search"
  | "arrow"
  | "settings"
  | "bolt"
  | "monitor"
  | "brain"
  | "shield"
  | "whatsapp"
  | "mail"
  | "instagram"
  | "youtube";

const iconPaths: Record<IconName, ReactNode> = {
  rocket: <><path d="M5 15c-1.5 1.5-2 5-2 5s3.5-.5 5-2c.8-.8.8-2.2 0-3-.8-.8-2.2-.8-3 0Z" /><path d="M14.5 4.5C17 2 20 2.5 20 2.5s.5 3-2 5.5l-6.5 6.5-3.5-3.5 6.5-6.5Z" /><path d="m9 15-1.5 1.5M13 6l5 5" /></>,
  play: <><circle cx="12" cy="12" r="9" /><path d="M10 8.5v7l6-3.5-6-3.5Z" /></>,
  tag: <><path d="M3 12V5a2 2 0 0 1 2-2h7l9 9-9 9-9-9Z" /><circle cx="7.5" cy="7.5" r="1.2" /></>,
  quote: <><path d="M10 7H6a2 2 0 0 0-2 2v3a2 2 0 0 0 2 2h2v1a3 3 0 0 1-3 3" /><path d="M20 7h-4a2 2 0 0 0-2 2v3a2 2 0 0 0 2 2h2v1a3 3 0 0 1-3 3" /></>,
  help: <><circle cx="12" cy="12" r="9" /><path d="M9.5 9a2.5 2.5 0 1 1 3.5 2.3c-.7.4-1 1-1 1.7v.5" /><circle cx="12" cy="17" r="0.7" fill="currentColor" stroke="none" /></>,
  chart: <><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /></>,
  user: <><circle cx="12" cy="8" r="3.5" /><path d="M5 20a7 7 0 0 1 14 0" /></>,
  headset: <><path d="M4 13a8 8 0 0 1 16 0" /><rect x="3" y="13" width="4" height="6" rx="1.5" /><rect x="17" y="13" width="4" height="6" rx="1.5" /><path d="M19 19a3 3 0 0 1-3 3h-3" /></>,
  chat: <><path d="M21 11.5a8.5 8.5 0 0 1-12.6 7.5L3 21l2-5.4A8.5 8.5 0 1 1 21 11.5Z" /><path d="M8.5 10h7M8.5 13.5h4" /></>,
  search: <><circle cx="11" cy="11" r="7" /><path d="m20 20-3.2-3.2" /></>,
  arrow: <><path d="M5 12h14M13 6l6 6-6 6" /></>,
  settings: <><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.8.3H9a1.6 1.6 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.6 1.6 0 0 0 1 1.5 1.6 1.6 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.8V9a1.6 1.6 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.6 1.6 0 0 0-1.5 1Z" /></>,
  bolt: <><path d="M13 2 4 14h6l-1 8 9-12h-6l1-8Z" /></>,
  monitor: <><rect x="3" y="4" width="18" height="12" rx="2" /><path d="M8 20h8M12 16v4" /></>,
  brain: <><path d="M9.5 4.5A3 3 0 0 0 6 7.4 3 3 0 0 0 4.5 13 3 3 0 0 0 6 18.5a3 3 0 0 0 5.5 1.5V5.8a3 3 0 0 0-2-1.3Z" /><path d="M14.5 4.5A3 3 0 0 1 18 7.4 3 3 0 0 1 19.5 13 3 3 0 0 1 18 18.5a3 3 0 0 1-5.5 1.5V5.8a3 3 0 0 1 2-1.3Z" /></>,
  shield: <><path d="M12 3 4.5 6v5c0 5 3.2 8 7.5 10 4.3-2 7.5-5 7.5-10V6L12 3Z" /><path d="m9 12 2 2 4-4" /></>,
  whatsapp: <><path d="M21 11.5a8.5 8.5 0 0 1-12.6 7.5L3 21l2-5.4A8.5 8.5 0 1 1 21 11.5Z" /><path d="M8.8 9.2c.2 2.8 2.2 4.8 5 5" /></>,
  mail: <><rect x="3" y="5" width="18" height="14" rx="2" /><path d="m3 7 9 6 9-6" /></>,
  instagram: <><rect x="3" y="3" width="18" height="18" rx="5" /><circle cx="12" cy="12" r="4" /><circle cx="17.5" cy="6.5" r="0.8" fill="currentColor" stroke="none" /></>,
  youtube: <><rect x="2.5" y="6" width="19" height="12" rx="4" /><path d="m10 9.5 5 2.5-5 2.5v-5Z" /></>,
};

function Icon({ name }: { name: IconName }) {
  return <svg viewBox="0 0 24 24" aria-hidden="true">{iconPaths[name]}</svg>;
}

function PlayGlyph() {
  return <svg viewBox="0 0 24 24" aria-hidden="true" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>;
}

const solusiSlides = [
  { tag: "SD · SMP · SMA", title: "Tryout terstandar", text: "Bank soal disusun per jenjang dan bab, lengkap dengan pembahasan setelah ujian." },
  { tag: "CBT realistis", title: "Rasakan ujian sebenarnya", text: "Timer, navigasi soal, dan mode layar terkunci memberi pengalaman CBT yang sesungguhnya." },
  { tag: "Analitik per bab", title: "Belajar lebih terarah", text: "Lihat kekuatan dan area yang perlu diulang lewat skor tiap bab setelah setiap tryout." },
  { tag: "Bank soal & media", title: "Soal lebih hidup", text: "Guru dapat mengunggah gambar, audio, dan video langsung ke server tanpa hosting eksternal." },
];

const solutionCards: { icon: IconName; title: string; text: string; points: string[]; cta: string }[] = [
  {
    icon: "monitor",
    title: "Tryout & CBT",
    text: "Simulasi ujian lengkap dengan timer, token, autosave, dan mode layar terkunci.",
    points: ["Soal per jenjang & bab", "Token ujian aman", "Autosave jawaban"],
    cta: "Mulai tryout",
  },
  {
    icon: "brain",
    title: "Analitik belajar",
    text: "Pahami performa lewat skor per bab, ranking global, dan pembahasan soal.",
    points: ["Skor per bab", "Ranking persentil", "Pembahasan terarah"],
    cta: "Lihat analitik",
  },
  {
    icon: "shield",
    title: "Bank soal guru",
    text: "Guru dapat menyusun paket, ujian, dan soal media dengan alur yang terkontrol.",
    points: ["Upload gambar, audio, video", "Verifikasi guru", "Kelola peserta CBT"],
    cta: "Masuk sebagai guru",
  },
];

const faqItems = [
  { q: "Apakah ada paket yang gratis?", a: "Ada. Paket bertanda GRATIS bisa langsung diambil dari katalog tanpa pembayaran. Kamu tetap perlu membuat akun agar hasil ujian tersimpan." },
  { q: "Bagaimana cara mengikuti ujian CBT?", a: "Masuk ke dashboard, buka menu Ujian CBT, lalu masukkan token dari gurumu. Token hanya berlaku untuk paket CBT yang sedang aktif." },
  { q: "Apakah nilai keluar langsung setelah ujian?", a: "Sebagian besar ujian menampilkan nilai langsung setelah dikumpulkan. Jika pengelola mengatur rilis nilai, hasil akan muncul setelah waktu rilis tiba." },
  { q: "Bisa dikerjakan dari HP?", a: "Bisa. Tampilan dashboard, katalog, dan ujian sudah responsif untuk HP, tablet, maupun desktop." },
  { q: "Metode pembayaran apa saja yang tersedia?", a: "Pembayaran diproses melalui kanal aman Xendit. Metode aktif seperti QRIS, virtual account, dan e-wallet tampil saat kamu membuat invoice." },
  { q: "Bagaimana guru mengunggah soal bergambar atau video?", a: "Guru dapat mengunggah media langsung ke server melalui editor soal. Media disajikan dengan URL aman dan tidak memerlukan layanan hosting eksternal." },
];

const rupiah = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 });

const subscribeNoop = () => () => {};

type ModalKey = "solusi" | "video" | "paket" | "testimoni" | "faq" | "stats" | "kontak" | null;

function truncate(text: string, length: number) {
  return text.length > length ? `${text.slice(0, length).trimEnd()}…` : text;
}

function activate(fn: () => void) {
  return (event: ReactKeyboardEvent) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      fn();
    }
  };
}

function Modal({
  open,
  title,
  subtitle,
  accent,
  icon,
  narrow,
  onClose,
  children,
}: {
  open: boolean;
  title: string;
  subtitle?: string;
  accent: string;
  icon: IconName;
  narrow?: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!open) return null;
  return createPortal(
    <div className={styles.overlay} role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className={`${styles.modal} ${narrow ? styles.modalNarrow : ""}`} role="dialog" aria-modal="true" aria-labelledby="metro-modal-title" onMouseDown={(event) => event.stopPropagation()}>
        <header className={styles.modalHeader} style={{ background: accent }}>
          <div className={styles.modalHeading}>
            <span className={styles.modalIcon}><Icon name={icon} /></span>
            <div>
              <h2 id="metro-modal-title">{title}</h2>
              {subtitle && <p>{subtitle}</p>}
            </div>
          </div>
          <button type="button" className={styles.closeButton} onClick={onClose} aria-label="Tutup">×</button>
        </header>
        <div className={styles.modalBody}>{children}</div>
      </section>
    </div>,
    document.body,
  );
}

export default function MetroDashboard() {
  const router = useRouter();
  const [settings, setSettings] = useState<SiteSettings>();
  const [packages, setPackages] = useState<Package[]>([]);
  const [testimonials, setTestimonials] = useState<Testimonial[]>([]);
  const [openModal, setOpenModal] = useState<ModalKey>(null);
  const [query, setQuery] = useState("");
  const [now, setNow] = useState<Date>(() => new Date());
  const isClient = useSyncExternalStore(subscribeNoop, () => true, () => false);
  const authed = useSyncExternalStore(subscribeNoop, () => tokenStore.hasToken(), () => false);
  const [solusiIndex, setSolusiIndex] = useState(0);
  const [videoIndex, setVideoIndex] = useState(0);
  const [packageTab, setPackageTab] = useState("semua");
  const [compact, setCompact] = useState(false);

  useEffect(() => {
    api.siteSettings().then(setSettings).catch(() => undefined);
    api.packages().then((response) => setPackages(response.data)).catch(() => undefined);
    api.getPublicTestimonials().then((list) => setTestimonials(list ?? [])).catch(() => undefined);
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => setSolusiIndex((current) => (current + 1) % solusiSlides.length), 5200);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!openModal) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setOpenModal(null); };
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", onKey);
    };
  }, [openModal]);

  const platformName = settings?.platform_name || "SIAP TKA";
  const tagline = settings?.platform_tagline || "Platform Tryout & CBT";
  const videos = settings?.youtube_videos ?? [];
  const activeVideo = videos.length ? videos[Math.min(videoIndex, videos.length - 1)] : undefined;
  const activeVideoID = activeVideo ? youtubeVideoID(activeVideo.url) : "";

  const totalSales = packages.reduce((sum, item) => sum + (item.sales_count || 0), 0);
  const totalViews = packages.reduce((sum, item) => sum + (item.view_count || 0), 0);
  const paidPrices = packages.filter((item) => item.price > 0).map((item) => item.price);
  const minPriceLabel = paidPrices.length ? `Mulai ${rupiah.format(Math.min(...paidPrices))}` : "Lihat paket";

  const filteredPackages = useMemo(
    () => packages.filter((item) => packageTab === "semua" || item.jenjang === packageTab),
    [packages, packageTab],
  );

  const matches = (keywords: string) => {
    const term = query.trim().toLowerCase();
    return !term || keywords.toLowerCase().includes(term);
  };
  const tileClass = (base: string, keywords: string) => `${styles.tile} ${base} ${matches(keywords) ? "" : styles.tileHidden}`;

  const timeLabel = isClient ? now.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" }) : "--:--";
  const dateLabel = isClient ? now.toLocaleDateString("id-ID", { day: "2-digit", month: "short" }) : "";

  const goAuth = () => router.push(authed ? "/dashboard" : "/login");
  const closeModal = () => setOpenModal(null);
  const slide = solusiSlides[solusiIndex];

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.headerInner}>
          <button type="button" className={styles.brand} onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })} aria-label={`Beranda ${platformName}`}>
            <span className={styles.brandMark}>TKA</span>
            <span className={styles.brandText}>
              <strong>{platformName}</strong>
              <span>{tagline}</span>
            </span>
          </button>

          <label className={styles.search}>
            <Icon name="search" />
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Cari fitur, paket, atau bantuan..."
              aria-label="Cari fitur, paket, atau bantuan"
            />
          </label>

          <div className={styles.headerActions}>
            <div className={styles.clock} aria-hidden="true">
              <strong>{timeLabel}</strong>
              <span>{dateLabel}</span>
            </div>
            <button type="button" className={styles.authButton} onClick={goAuth}>
              <Icon name="user" />
              {authed ? "Dashboard" : "Masuk / Daftar"}
            </button>
          </div>
        </div>
      </header>

      <main className={styles.main}>
        <div className={styles.welcome}>
          <div>
            <h1>Selamat datang di {platformName}</h1>
            <p>Tryout terstandar, CBT realistis, dan analitik belajar dalam satu platform.</p>
          </div>
          <span className={styles.live}><i className={styles.liveDot} />Sistem operasional · 99,9% uptime</span>
        </div>

        <div className={`${styles.grid} ${compact ? styles.gridCompact : ""}`}>
          {/* Solusi digital */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.solusi, "solusi digital tryout cbt bank soal analitik media")}
            onClick={() => setOpenModal("solusi")}
            onKeyDown={activate(() => setOpenModal("solusi"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileBadge}><Icon name="rocket" />Solusi Digital</span>
              <span className={styles.tileBadge}>Utama</span>
            </div>
            <div className={styles.solusiBody}>
              <div className={styles.solusiSlide} key={solusiIndex}>
                <span className={styles.solusiTag}>{slide.tag}</span>
                <h2>{slide.title}</h2>
                <p>{slide.text}</p>
              </div>
            </div>
            <div className={styles.tileFoot}>
              <div className={styles.dots}>
                {solusiSlides.map((item, index) => (
                  <button
                    type="button"
                    key={item.title}
                    className={`${styles.dot} ${index === solusiIndex ? styles.dotActive : ""}`}
                    aria-label={`Slide ${index + 1}`}
                    onClick={(event) => { event.stopPropagation(); setSolusiIndex(index); }}
                  />
                ))}
              </div>
              <span className={styles.tileArrow}>Jelajahi <Icon name="arrow" /></span>
            </div>
          </div>

          {/* Video showcase */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.video, "video showcase panduan promosi tonton")}
            onClick={() => setOpenModal("video")}
            onKeyDown={activate(() => setOpenModal("video"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileBadge}><span className={styles.liveTag}><i />Video</span></span>
              <span className={styles.tileIcon}><Icon name="play" /></span>
            </div>
            <div className={styles.videoRow}>
              <span
                className={styles.videoThumb}
                style={activeVideoID ? { backgroundImage: `url(https://i.ytimg.com/vi/${activeVideoID}/mqdefault.jpg)` } : undefined}
              >
                <PlayGlyph />
              </span>
              <div className={styles.videoMeta}>
                <strong>{activeVideo?.title || "Video panduan"}</strong>
                <span>{videos.length ? `${videos.length} video tersedia` : "Belum ada video"}</span>
              </div>
            </div>
            <div className={styles.tileFoot}><span>Putar video</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* Paket & harga */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.paket, "paket harga tryout katalog beli gratis")}
            onClick={() => setOpenModal("paket")}
            onKeyDown={activate(() => setOpenModal("paket"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileIcon}><Icon name="tag" /></span>
              <span className={styles.tileBadge}>{packages.length ? `${packages.length} paket` : "Katalog"}</span>
            </div>
            <div className={styles.tileBody}>
              <h3>Paket & Harga</h3>
              <p className={styles.tileDesc}>Pilih paket sesuai jenjang. Ada paket gratis untuk mulai.</p>
            </div>
            <div className={styles.tileFoot}><span>{minPriceLabel}</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* Testimoni */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.testimoni, "testimoni ulasan review siswa")}
            onClick={() => setOpenModal("testimoni")}
            onKeyDown={activate(() => setOpenModal("testimoni"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileBadge}><Icon name="quote" />Testimoni</span>
              <span className={styles.tileIcon}><Icon name="quote" /></span>
            </div>
            <div className={styles.tileBody}>
              <h3>{testimonials[0]?.user_name || "Kata mereka"}</h3>
              <p className={styles.tileDesc}>{testimonials[0]?.quote ? `“${truncate(testimonials[0].quote, 76)}”` : "Ulasan siswa yang sudah berlatih di platform ini."}</p>
            </div>
            <div className={styles.tileFoot}><span>Lihat semua ulasan</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* FAQ */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.faq, "faq bantuan pertanyaan panduan")}
            onClick={() => setOpenModal("faq")}
            onKeyDown={activate(() => setOpenModal("faq"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileIcon}><Icon name="help" /></span>
              <span className={styles.tileBadge}>Tanya jawab</span>
            </div>
            <div className={styles.tileBody}>
              <h3>Bantuan & FAQ</h3>
              <p className={styles.tileDesc}>Pertanyaan umum seputar akun, ujian, dan pembayaran.</p>
            </div>
            <div className={styles.tileFoot}><span>Buka FAQ</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* Statistik */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.stats, "statistik performa data laporan")}
            onClick={() => setOpenModal("stats")}
            onKeyDown={activate(() => setOpenModal("stats"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileBadge}>Performa</span>
              <span className={styles.tileIcon}><Icon name="chart" /></span>
            </div>
            <div className={styles.statsGrid}>
              <div><strong>{totalSales.toLocaleString("id-ID")}</strong><span>Paket terjual</span></div>
              <div><strong>{totalViews.toLocaleString("id-ID")}</strong><span>Kunjungan</span></div>
            </div>
            <div className={styles.tileFoot}><span>Detail laporan</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* Masuk / daftar */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.auth, "masuk daftar login akun portal")}
            onClick={goAuth}
            onKeyDown={activate(goAuth)}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileIcon}><Icon name="user" /></span>
              <span className={styles.tileBadge}>{authed ? "Akun" : "Portal"}</span>
            </div>
            <div className={styles.tileBody}>
              <h3>{authed ? "Dashboard Saya" : "Masuk / Daftar"}</h3>
              <p className={styles.tileDesc}>Akses dashboard, paket belajar, dan hasil tryout.</p>
            </div>
            <div className={styles.tileFoot}><span>{authed ? "Buka dashboard" : "Login sekarang"}</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>

          {/* Kontak */}
          <div
            role="button"
            tabIndex={0}
            className={tileClass(styles.kontak, "kontak hubungi whatsapp email bantuan support")}
            onClick={() => setOpenModal("kontak")}
            onKeyDown={activate(() => setOpenModal("kontak"))}
          >
            <div className={styles.tileTop}>
              <span className={styles.tileIcon}><Icon name="headset" /></span>
              <span className={styles.tileBadge}>Online</span>
            </div>
            <div className={styles.tileBody}>
              <h3>Hubungi Kami</h3>
              <p className={styles.tileDesc}>Konsultasi gratis lewat WhatsApp atau email.</p>
            </div>
            <div className={styles.tileFoot}><span>Kirim pesan</span><span className={styles.tileArrow}><Icon name="arrow" /></span></div>
          </div>
        </div>

        <div className={styles.commandBar}>
          <div className={styles.commandCopy}>
            <span className={styles.commandIcon}><Icon name="headset" /></span>
            <div>
              <strong>Butuh bantuan memulai?</strong>
              <span>Tim kami siap membantu memilih paket dan alur tryout yang tepat.</span>
            </div>
          </div>
          <div className={styles.commandActions}>
            <button type="button" className={styles.primaryButton} onClick={goAuth}><Icon name="bolt" />{authed ? "Buka Dashboard" : "Mulai Tryout"}</button>
            <button type="button" className={styles.ghostButton} onClick={() => setOpenModal("kontak")}><Icon name="chat" />Konsultasi Gratis</button>
            <button type="button" className={styles.ghostButton} onClick={() => setCompact((value) => !value)}><Icon name="settings" />{compact ? "Tampilan Normal" : "Pengaturan Tampilan"}</button>
          </div>
        </div>
      </main>

      <footer className={styles.footer}>
        <span>© {now ? now.getFullYear() : new Date().getFullYear()} {platformName}. Semua hak dilindungi.</span>
        <div className={styles.footerLinks}>
          <Link href="/terms">Syarat Penggunaan</Link>
          <Link href="/privacy">Kebijakan Privasi</Link>
          {settings?.support_email && <a href={`mailto:${settings.support_email}`}>Email</a>}
        </div>
      </footer>

      <Modal
        open={openModal === "solusi"}
        title="Solusi Digital"
        subtitle="Ekosistem tryout, CBT, dan bank soal dalam satu platform"
        accent="linear-gradient(120deg, #0078d7, #0b3f8a)"
        icon="rocket"
        onClose={closeModal}
      >
        <div className={styles.cardGrid}>
          {solutionCards.map((card) => (
            <article className={styles.cardItem} key={card.title}>
              <span className={styles.cardIcon}><Icon name={card.icon} /></span>
              <h3>{card.title}</h3>
              <p>{card.text}</p>
              <ul className={styles.checkList}>{card.points.map((point) => <li key={point}>{point}</li>)}</ul>
              <button type="button" className={styles.packageButton} onClick={goAuth}>{card.cta}</button>
            </article>
          ))}
        </div>
      </Modal>

      <Modal
        open={openModal === "video"}
        title="Video Showcase"
        subtitle="Tonton panduan dan promosi platform"
        accent="linear-gradient(120deg, #7b2ff7, #4a148c)"
        icon="play"
        onClose={closeModal}
      >
        {activeVideoID ? (
          <div className={styles.videoPlayer}>
            <iframe
              src={`https://www.youtube-nocookie.com/embed/${activeVideoID}?rel=0`}
              title={activeVideo?.title || "Video SIAP TKA"}
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullScreen
            />
          </div>
        ) : (
          <div className={styles.videoEmpty}><Icon name="play" /><span>Belum ada video yang dipublikasikan. Video panduan dan promosi akan tampil di sini.</span></div>
        )}
        {videos.length > 0 && (
          <div className={styles.videoList}>
            {videos.map((video, index) => {
              const id = youtubeVideoID(video.url);
              return (
                <button
                  type="button"
                  key={video.id}
                  className={`${styles.videoPick} ${index === videoIndex ? styles.videoPickActive : ""}`}
                  onClick={() => setVideoIndex(index)}
                >
                  <span className={styles.videoPickThumb} style={id ? { backgroundImage: `url(https://i.ytimg.com/vi/${id}/mqdefault.jpg)` } : undefined}><PlayGlyph /></span>
                  <span>{video.title || `Video ${index + 1}`}</span>
                </button>
              );
            })}
          </div>
        )}
      </Modal>

      <Modal
        open={openModal === "paket"}
        title="Paket & Harga"
        subtitle="Pilih paket tryout sesuai jenjang"
        accent="linear-gradient(120deg, #10b981, #047857)"
        icon="tag"
        onClose={closeModal}
      >
        <div className={styles.tabs}>
          {["semua", "SD", "SMP", "SMA"].map((tab) => (
            <button
              type="button"
              key={tab}
              className={`${styles.tab} ${packageTab === tab ? styles.tabActive : ""}`}
              onClick={() => setPackageTab(tab)}
            >
              {tab === "semua" ? "Semua" : tab}
            </button>
          ))}
        </div>
        {filteredPackages.length === 0 ? (
          <div className={styles.emptyNote}>Paket untuk jenjang ini belum tersedia. Coba pilih jenjang lain.</div>
        ) : (
          <div className={styles.packageGrid}>
            {filteredPackages.map((pkg) => (
              <article className={styles.packageCard} key={pkg.id}>
                <div className={styles.packageTop}>
                  <span className={styles.packageCode}>{pkg.kode}</span>
                  <span className={styles.packageLevel}>{pkg.jenjang}{pkg.exam_type === "cbt" ? " · CBT" : ""}</span>
                </div>
                <h3>{pkg.title}</h3>
                <p>{pkg.description}</p>
                <div className={styles.packagePrice}>{pkg.price === 0 ? "Gratis" : rupiah.format(pkg.price)}</div>
                <div className={styles.packageMeta}>
                  <span>{pkg.exam_count} ujian</span>
                  <span>{pkg.question_count} soal</span>
                  <span>Aktif {pkg.validity_days} hari</span>
                </div>
                <button type="button" className={styles.packageButton} onClick={goAuth}>{pkg.price === 0 ? "Ambil gratis" : "Beli paket"}</button>
              </article>
            ))}
          </div>
        )}
      </Modal>

      <Modal
        open={openModal === "testimoni"}
        title="Testimoni"
        subtitle="Apa kata siswa yang sudah berlatih"
        accent="linear-gradient(120deg, #f59e0b, #b45309)"
        icon="quote"
        narrow
        onClose={closeModal}
      >
        {testimonials.length === 0 ? (
          <div className={styles.emptyNote}>Belum ada testimoni yang dipublikasikan.</div>
        ) : (
          <div className={styles.testimonialList}>
            {testimonials.map((item) => (
              <article className={styles.testimonialItem} key={item.id}>
                <span className={styles.testimonialAvatar}>{(item.user_name || item.user_email || "T")[0].toUpperCase()}</span>
                <div>
                  <h3>{item.user_name || item.user_email}</h3>
                  <small>Siswa {platformName}</small>
                  <p>“{item.quote}”</p>
                </div>
              </article>
            ))}
          </div>
        )}
      </Modal>

      <Modal
        open={openModal === "faq"}
        title="Bantuan & FAQ"
        subtitle="Pertanyaan yang sering diajukan"
        accent="linear-gradient(120deg, #f43f5e, #be123c)"
        icon="help"
        narrow
        onClose={closeModal}
      >
        <div className={styles.faqList}>
          {faqItems.map((item) => (
            <details className={styles.faqItem} key={item.q}>
              <summary>{item.q}</summary>
              <p>{item.a}</p>
            </details>
          ))}
        </div>
      </Modal>

      <Modal
        open={openModal === "stats"}
        title="Statistik Platform"
        subtitle="Ringkasan performa katalog dan sistem"
        accent="linear-gradient(120deg, #2563eb, #1e3a8a)"
        icon="chart"
        onClose={closeModal}
      >
        <div className={styles.statCards}>
          <div className={styles.statCard}><strong>{packages.length}</strong><span>Paket tryout</span></div>
          <div className={styles.statCard}><strong>{totalSales.toLocaleString("id-ID")}</strong><span>Paket terjual</span></div>
          <div className={styles.statCard}><strong>{totalViews.toLocaleString("id-ID")}</strong><span>Kunjungan</span></div>
          <div className={styles.statCard}><strong>99,9%</strong><span>Uptime</span></div>
        </div>
        <p className={styles.contactNote}>Statistik katalog diambil langsung dari data paket dan diperbarui setiap halaman dimuat. Uptime mengacu pada target layanan produksi.</p>
        <div className={styles.statBar}><i style={{ width: "99.9%" }} /></div>
      </Modal>

      <Modal
        open={openModal === "kontak"}
        title="Hubungi Kami"
        subtitle="Konsultasi gratis dengan tim SIAP TKA"
        accent="linear-gradient(120deg, #06b6d4, #0e7490)"
        icon="headset"
        narrow
        onClose={closeModal}
      >
        <div className={styles.contactGrid}>
          {settings?.whatsapp && (
            <a className={styles.contactCard} href={`https://wa.me/${settings.whatsapp.replace(/\D/g, "")}`} target="_blank" rel="noopener noreferrer">
              <Icon name="whatsapp" />
              <div><strong>WhatsApp</strong><span>{settings.whatsapp}</span></div>
            </a>
          )}
          {settings?.support_email && (
            <a className={styles.contactCard} href={`mailto:${settings.support_email}`}>
              <Icon name="mail" />
              <div><strong>Email</strong><span>{settings.support_email}</span></div>
            </a>
          )}
          {settings?.instagram_url && (
            <a className={styles.contactCard} href={settings.instagram_url} target="_blank" rel="noopener noreferrer">
              <Icon name="instagram" />
              <div><strong>Instagram</strong><span>Lihat profil</span></div>
            </a>
          )}
          {settings?.youtube_url && (
            <a className={styles.contactCard} href={settings.youtube_url} target="_blank" rel="noopener noreferrer">
              <Icon name="youtube" />
              <div><strong>YouTube</strong><span>Lihat video</span></div>
            </a>
          )}
        </div>
        <p className={styles.contactNote}>Konsultasi gratis untuk memilih paket, membuat soal, atau mengatur ujian CBT. Tim kami biasanya membalas pada jam kerja.</p>
        <button type="button" className={styles.primaryButton} onClick={goAuth} style={{ marginTop: 14 }}><Icon name="user" />{authed ? "Buka Dashboard" : "Masuk / Daftar"}</button>
      </Modal>
    </div>
  );
}
