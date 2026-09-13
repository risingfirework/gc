"use client";

import { useEffect, useState } from "react";
import { HeroSlide } from "@/services/api";

export const heroSlides = [
  { id: "adaptive", image_data_url: "", title: "Lebih siap menghadapi ujian." },
  { id: "analytics", image_data_url: "", title: "Pahami kekuatan dan kelemahanmu." },
  { id: "realistic", image_data_url: "", title: "Rasakan simulasi seperti ujian asli." },
];

export default function HeroSlider({ onStart, compact, showPaket, slides=heroSlides, intervalMS=6000 }: { onStart: () => void; compact?: boolean; showPaket?: boolean; slides?:HeroSlide[]; intervalMS?:number }) {
  const [active, setActive] = useState(0);
  useEffect(() => {
    if(slides.length<2)return;
    const timer = setInterval(() => setActive((current) => (current + 1) % slides.length), intervalMS);
    return () => clearInterval(timer);
  }, [slides,intervalMS]);
  const current = slides[Math.min(active, Math.max(0, slides.length - 1))];
  return <section className={`hero-slider${compact ? " compact" : ""}`}>
    <div className={compact ? "" : "container"}>
      <div className="hero-slides">
        {slides.map((slide, index) => <div className="hero-slide" key={slide.id} style={{ display: index === active ? "block" : "none" }}>
          <div className="hero-image" style={slide.image_data_url ? { backgroundImage: `url(${slide.image_data_url})` } : undefined}>
            {!slide.image_data_url && <div className="hero-image-fallback"><strong>Gambar slider</strong><span>Unggah gambar lewat Pengaturan &gt; Slider halaman depan.</span></div>}
          </div>
        </div>)}
      </div>
      {current && <div className="hero-caption">
        <h1 className={compact ? "compact-title" : ""}>{current.title || "Ayo mulai berlatih"}</h1>
        {compact && <div className="actions"><button className="button" onClick={onStart}>Mulai tryout</button>{showPaket && <a className="button secondary" href="#paket">Lihat paket</a>}</div>}
      </div>}
      <div className="hero-dots">{slides.map((slide, index) => <button key={slide.id} className={index === active ? "dot active" : "dot"} aria-label={`Slide ${index + 1}`} onClick={() => setActive(index)}/>)}</div>
    </div>
  </section>;
}