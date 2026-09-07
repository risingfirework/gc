"use client";

import { useEffect, useRef, useState } from "react";
import { api, Testimonial } from "@/services/api";

export function SliderArrows({ onPrev, onNext }: { onPrev: () => void; onNext: () => void }) {
  return <div className="slider-arrows"><button className="slider-arrow" onClick={onPrev} aria-label="Sebelumnya">‹</button><button className="slider-arrow" onClick={onNext} aria-label="Berikutnya">›</button></div>;
}

export default function TestimonialSlider() {
  const trackRef = useRef<HTMLDivElement>(null);
  const [items, setItems] = useState<Testimonial[]>([]);
  useEffect(() => { api.getPublicTestimonials().then(setItems).catch(() => undefined); }, []);
  function scroll(direction: 1 | -1) { trackRef.current?.scrollBy({ left: direction * 300, behavior: "smooth" }); }
  if (items.length === 0) return null;
  return <section className="section" id="testimoni">
    <div className="section-heading"><div><p className="eyebrow">Testimoni</p><h2>Kata mereka tentang TKA Juara</h2></div><SliderArrows onPrev={() => scroll(-1)} onNext={() => scroll(1)}/></div>
    <div className="slider-track" ref={trackRef}>
      {items.map((item) => <article className="card testimonial-card slider-item" key={item.id}>
        <p className="testimonial-quote">&ldquo;{item.quote}&rdquo;</p>
        <div className="testimonial-author"><span className="profile-avatar">{(item.user_name || item.user_email || "T")[0].toUpperCase()}</span><div><strong>{item.user_name || item.user_email}</strong><small>Siswa TKA Juara</small></div></div>
      </article>)}
    </div>
  </section>;
}