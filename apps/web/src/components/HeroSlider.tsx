"use client";

import { useEffect, useState } from "react";
import { HeroSlide } from "@/services/api";

export const heroSlides = [
  {
    id: "adaptive",
    eyebrow: "Tryout adaptif & terukur",
    title: "Lebih siap menghadapi TKA.",
    lead: "Latihan seperti ujian sebenarnya, simpan jawaban otomatis, dan pahami kemampuanmu lewat analisis hasil per mata pelajaran.",
    stats: [["24/7", "Akses latihan"], ["Auto", "Simpan jawaban"], ["IRT", "Analisis nilai"], ["Aman", "Kontrol ujian"]],
  },
  {
    id: "analytics",
    eyebrow: "Analisis mendalam",
    title: "Pahami kekuatan dan kelemahanmu.",
    lead: "Skor per mata pelajaran, riwayat pengerjaan, dan rekomendasi belajar yang lebih terarah untuk setiap siswa.",
    stats: [["Detail", "Skor per mapel"], ["Riwayat", "Semua percobaan"], ["Ranking", "Peringkat global"], ["Insight", "Rekomendasi belajar"]],
  },
  {
    id: "realistic",
    eyebrow: "Ujian realistis",
    title: "Rasakan simulasi seperti ujian asli.",
    lead: "Timer berjalan di server, autosave setiap jawaban, dan deteksi perpindahan tab menjaga hasil tetap valid dan adil.",
    stats: [["Server", "Timer akurat"], ["Live", "Autosave jawaban"], ["Deteksi", "Anti kecurangan"], ["Valid", "Hasil terpercaya"]],
  },
];

export default function HeroSlider({ onStart, compact, showPaket, slides=heroSlides, intervalMS=6000 }: { onStart: () => void; compact?: boolean; showPaket?: boolean; slides?:HeroSlide[]; intervalMS?:number }) {
  const [active, setActive] = useState(0);
  useEffect(() => {
    if(slides.length<2)return;
    const timer = setInterval(() => setActive((current) => (current + 1) % slides.length), intervalMS);
    return () => clearInterval(timer);
  }, [slides,intervalMS]);
  return <section className={`hero-slider${compact ? " compact" : ""}`}>
    <div className={compact ? "" : "container"}>
      {slides.map((slide, index) => <div className="hero" key={slide.id} style={{ display: index === active ? "grid" : "none" }}>
        <div>
          <div className="eyebrow">{slide.eyebrow}</div>
          <h1 className={compact ? "compact-title" : ""}>{slide.title}</h1>
          <p className="lead">{slide.lead}</p>
          <div className="actions"><button className="button" onClick={onStart}>Mulai tryout</button>{showPaket && <a className="button secondary" href="#paket">Lihat paket</a>}</div>
        </div>
        <div className="card">
          <p className="eyebrow">Dirancang untuk fokus</p>
          <h2>Satu tempat untuk belajar dan mengukur progres.</h2>
          <div className="stat-grid">{slide.stats.map(([value, label]) => <div className="stat" key={label}><strong>{value}</strong><span>{label}</span></div>)}</div>
        </div>
      </div>)}
      <div className="hero-dots">{slides.map((slide, index) => <button key={slide.id} className={index === active ? "dot active" : "dot"} aria-label={`Slide ${index + 1}`} onClick={() => setActive(index)}/>)}</div>
    </div>
  </section>;
}
