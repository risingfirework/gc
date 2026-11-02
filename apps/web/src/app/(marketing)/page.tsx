"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { api, Package, SiteSettings } from "@/services/api";
import TestimonialSlider, { SliderArrows } from "@/components/TestimonialSlider";
import HeroSlider from "@/components/HeroSlider";
import CatalogStats from "@/components/CatalogStats";
import BrandLogo from "@/components/BrandLogo";
import { VideoSection } from "@/components/YouTubeEmbed";

const rupiah = new Intl.NumberFormat("id-ID", { style:"currency", currency:"IDR", maximumFractionDigits:0 });

export default function LandingPage() {
  const router=useRouter(); const openAuth=()=>router.push("/login"); const [settings,setSettings]=useState<SiteSettings>();
  useEffect(()=>{api.siteSettings().then(setSettings).catch(()=>undefined)},[]);
  return <main className="landing">
    <div className="container"><nav className="nav"><div className="brand"><BrandLogo logoDataURL={settings?.logo_data_url}/><span className="brand-copy"><strong>{settings?.platform_name??""}</strong><small>{settings?.platform_tagline??""}</small></span></div><div className="nav-links"><a href="#keunggulan">Keunggulan</a><a href="#paket">Paket</a><a href="#video">Video</a><a href="#testimoni">Testimoni</a><a href="#faq">FAQ</a><button className="button secondary" onClick={openAuth}>Masuk / Daftar</button></div></nav></div>
    <HeroSlider onStart={openAuth} showPaket slides={settings?.hero_slides} intervalMS={settings?.hero_interval_ms}/>
    <div className="container"><LandingFeatures/><PackagesSlider onStart={openAuth} intervalMS={settings?.catalog_interval_ms}/><HowItWorks/><VideoSection videos={settings?.youtube_videos??[]} id="video" eyebrow="Tonton dulu" title={settings?.platform_name?`Kenali ${settings.platform_name} dalam satu video`:"Kenali kami dalam satu video"} lead="Tonton cara berlatih, analisis hasil, dan alur pembelian paket sebelum memulai." /><TestimonialSlider platformName={settings?.platform_name}/><LandingFaq/><CtaBand onStart={openAuth}/></div>
    <ContactFooter settings={settings}/>
  </main>;
}

const featureItems = [
  {
    title: "Soal terstandar",
    text: "Bank soal disusun per jenjang dan bab, lengkap dengan pembahasan setelah ujian.",
    icon: <><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v17H6.5A2.5 2.5 0 0 0 4 22V5.5Z" /><path d="M20 5.5A2.5 2.5 0 0 0 17.5 3H13v17h4.5A2.5 2.5 0 0 1 20 22V5.5Z" /></>,
  },
  {
    title: "CBT realistis",
    text: "Timer, navigasi soal, dan mode layar terkunci memberi pengalaman ujian yang sesungguhnya.",
    icon: <><rect x="4" y="4" width="16" height="16" rx="3" /><path d="M8 9h8M8 13h5M8 17h3" /></>,
  },
  {
    title: "Analisis per bab",
    text: "Lihat kekuatan dan area yang perlu diulang lewat skor tiap bab setelah setiap tryout.",
    icon: <><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /></>,
  },
  {
    title: "Ranking global",
    text: "Bandingkan persentil dan waktu pengerjaanmu dengan siswa lain di jenjang yang sama.",
    icon: <><path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4Z" /><path d="M7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3" /></>,
  },
];

const howSteps = [
  { title: "Pilih paket", text: "Pilih paket tryout sesuai jenjang, lalu selesaikan pembayaran atau ambil paket gratis." },
  { title: "Kerjakan tryout", text: "Mulai ujian dari dashboard. Jawaban tersimpan otomatis, jadi kamu bisa melanjutkan kapan saja." },
  { title: "Evaluasi hasil", text: "Baca skor, analisis per bab, ranking, dan pembahasan untuk menentukan langkah belajar berikutnya." },
];

const faqItems = [
  { q: "Apakah ada paket yang gratis?", a: "Ada. Paket bertanda GRATIS bisa langsung diambil dari katalog tanpa pembayaran. Kamu tetap perlu membuat akun agar hasil ujian tersimpan." },
  { q: "Bagaimana cara mengikuti ujian CBT?", a: "Masuk ke dashboard, buka menu Ujian CBT, lalu masukkan token dari gurumu. Token hanya berlaku untuk paket CBT yang sedang aktif." },
  { q: "Apakah nilai keluar langsung setelah ujian?", a: "Sebagian besar ujian menampilkan nilai langsung setelah dikumpulkan. Kalau pengelola mengatur rilis nilai, hasil akan muncul setelah waktu rilis tiba." },
  { q: "Bisa dikerjakan dari HP?", a: "Bisa. Tampilan dashboard, katalog, dan ujian sudah responsif untuk HP, tablet, maupun desktop." },
  { q: "Metode pembayaran apa saja yang tersedia?", a: "Pembayaran diproses melalui kanal aman Xendit. Metode aktif seperti QRIS, virtual account, dan e-wallet tampil saat kamu membuat invoice." },
];

function LandingFeatures() {
  return <section className="section landing-features-section" id="keunggulan">
    <div className="section-heading"><div><p className="eyebrow">Kenapa belajar di sini</p><h2>Belajar lebih terarah, hasil lebih terukur</h2></div></div>
    <div className="landing-features">
      {featureItems.map((item) => <article className="feature-card" key={item.title}>
        <span className="feature-icon" aria-hidden="true"><svg viewBox="0 0 24 24">{item.icon}</svg></span>
        <h3>{item.title}</h3>
        <p>{item.text}</p>
      </article>)}
    </div>
  </section>;
}

function HowItWorks() {
  return <section className="section" id="cara-kerja">
    <div className="section-heading"><div><p className="eyebrow">Cara kerja</p><h2>Mulai dalam tiga langkah</h2></div></div>
    <div className="how-steps">
      {howSteps.map((step, index) => <article className="step-card" key={step.title}>
        <span className="step-number" aria-hidden="true">{index + 1}</span>
        <h3>{step.title}</h3>
        <p>{step.text}</p>
      </article>)}
    </div>
  </section>;
}

function LandingFaq() {
  return <section className="section" id="faq">
    <div className="section-heading"><div><p className="eyebrow">Pertanyaan umum</p><h2>Hal yang sering ditanyakan</h2></div></div>
    <div className="faq-list">
      {faqItems.map((item) => <details className="faq-item" key={item.q}>
        <summary>{item.q}</summary>
        <p>{item.a}</p>
      </details>)}
    </div>
  </section>;
}

function CtaBand({ onStart }: { onStart: () => void }) {
  return <section className="section cta-section">
    <div className="cta-band">
      <div>
        <h2>Siap mulai tryout pertamamu?</h2>
        <p>Daftar gratis, pilih paket sesuai jenjang, dan pantau perkembangan belajarmu dalam satu dashboard.</p>
      </div>
      <button className="button" type="button" onClick={onStart}>Daftar sekarang</button>
    </div>
  </section>;
}

function PackagesSlider({onStart,intervalMS=4000}:{onStart:()=>void;intervalMS?:number}) {
  const [packages,setPackages]=useState<Package[]>([]); const [error,setError]=useState(""); const [tab,setTab]=useState("semua"); const [query,setQuery]=useState(""); const trackRef=useRef<HTMLDivElement>(null); const pauseAutoRef=useRef(false);
  useEffect(()=>{api.packages().then((response)=>setPackages(response.data)).catch(()=>setError("Paket tryout belum dapat dimuat."))},[]);
  const q=query.trim().toLowerCase(); const visible=packages.filter((item)=>(tab==="semua"||item.jenjang===tab)&&(!q||item.kode.toLowerCase().includes(q)||item.title.toLowerCase().includes(q)||(item.publisher_email??"").toLowerCase().includes(q)));
  const cardStep=useCallback(()=>{const track=trackRef.current; const card=track?.querySelector<HTMLElement>(".package-card"); return card ? card.offsetWidth+18 : 284},[]);
  const scroll=useCallback((direction:1|-1)=>{const track=trackRef.current;if(!track)return;if(direction===1&&track.scrollLeft+track.clientWidth>=track.scrollWidth-cardStep()/2){track.scrollTo({left:0,behavior:"smooth"});return}if(direction===-1&&track.scrollLeft<=cardStep()/2){track.scrollTo({left:track.scrollWidth,behavior:"smooth"});return}track.scrollBy({left:direction*cardStep(),behavior:"smooth"})},[cardStep]);
  useEffect(()=>{trackRef.current?.scrollTo({left:0});if(visible.length<2)return;const timer=window.setInterval(()=>{if(!pauseAutoRef.current&&!document.hidden)scroll(1)},intervalMS);return()=>window.clearInterval(timer)},[tab,query,visible.length,scroll,intervalMS]);
  return <section className="section" id="paket">
    <div className="section-heading"><div><p className="eyebrow">Paket tryout</p><h2>Pilih paket sesuai targetmu</h2></div><SliderArrows onPrev={()=>scroll(-1)} onNext={()=>scroll(1)}/></div>
    <div className="catalog-toolbar"><input className="catalog-search" type="search" placeholder="Cari kode soal, nama paket, atau pembuat..." value={query} onChange={(event)=>setQuery(event.target.value)}/><div className="content-subtabs">{["semua","SD","SMP","SMA"].map((item)=><button key={item} className={tab===item?"active":""} onClick={()=>setTab(item)}>{item==="semua"?"Semua":item}</button>)}</div></div>
    {error&&<p className="error">{error}</p>}
    <div className="slider-track package-slider-track" ref={trackRef} onMouseEnter={()=>{pauseAutoRef.current=true}} onMouseLeave={()=>{pauseAutoRef.current=false}} onFocusCapture={()=>{pauseAutoRef.current=true}} onBlurCapture={()=>{pauseAutoRef.current=false}}>{visible.length?visible.map((item)=><article className="card package-card slider-item" key={item.id}>
      <div className="admin-package-card-top"><span className="package-code">{item.kode}</span><span className="package-jenjang">{item.jenjang}</span></div><h3>{item.title}</h3><p>{item.description}</p>{item.publisher_email&&<small className="package-publisher">Pembuat: {item.publisher_email}</small>}<CatalogStats sales={item.sales_count} views={item.view_count}/><div className="price">{item.price===0?"Gratis":rupiah.format(item.price)}</div>
      <button className="button full" onClick={()=>{void api.trackPackageView(item.id).then((viewCount)=>setPackages((current)=>current.map((pkg)=>pkg.id===item.id?{...pkg,view_count:viewCount}:pkg))).catch(()=>undefined);onStart()}}>Mulai sekarang</button>
    </article>):!error&&<p className="muted">{packages.length?"Tidak ada paket yang cocok dengan pencarian atau jenjang ini.":"Memuat paket tryout..."}</p>}</div>
  </section>;
}

function SocialIcon({type}:{type:string}){const paths:{[k:string]:string}={WA:"M12.04 2c-5.46 0-9.91 4.45-9.91 9.91 0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38a9.87 9.87 0 0 0 4.74 1.21c5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.9-7.01A9.85 9.85 0 0 0 12.04 2zm0 18.15c-1.48 0-2.93-.4-4.2-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.26 8.26 0 0 1-1.26-4.38c0-4.54 3.7-8.24 8.25-8.24 2.2 0 4.27.86 5.82 2.42a8.18 8.18 0 0 1 2.41 5.83c0 4.54-3.7 8.23-8.23 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.12-.17.25-.64.81-.78.97-.14.17-.29.19-.54.06-.25-.12-1.05-.39-1.99-1.23-.74-.66-1.23-1.47-1.38-1.72-.14-.25-.02-.38.11-.51.11-.11.25-.29.37-.43.12-.14.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.34-.76-1.84-.2-.48-.41-.42-.56-.43h-.48c-.17 0-.43.06-.66.31-.22.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.67-1.18.21-.58.21-1.07.14-1.18-.06-.1-.22-.16-.47-.28z",IG:"M12 2.2c3.2 0 3.6 0 4.9.1 1.2.1 1.8.2 2.2.4.6.2 1 .5 1.4.9.4.4.7.8.9 1.4.1.4.3 1 .4 2.2.1 1.3.1 1.7.1 4.9s0 3.6-.1 4.9c-.1 1.2-.2 1.8-.4 2.2-.2.6-.5 1-.9 1.4-.4.4-.8.7-1.4.9-.4.1-1 .3-2.2.4-1.3.1-1.7.1-4.9.1s-3.6 0-4.9-.1c-1.2-.1-1.8-.2-2.2-.4-.6-.2-1-.5-1.4-.9-.4-.4-.7-.8-.9-1.4-.1-.4-.3-1-.4-2.2-.1-1.3-.1-1.7-.1-4.9s0-3.6.1-4.9c.1-1.2.2-1.8.4-2.2.2-.6.5-1 .9-1.4.4-.4.8-.7 1.4-.9.4-.1 1-.3 2.2-.4 1.3-.1 1.7-.1 4.9-.1zm0 1.8c-3.1 0-3.5 0-4.8.1-1.1 0-1.7.2-2.1.4-.5.2-.9.4-1.3.8-.4.4-.6.8-.8 1.3-.2.4-.3 1-.4 2.1-.1 1.3-.1 1.7-.1 4.8s0 3.5.1 4.8c0 1.1.2 1.7.4 2.1.2.5.4.9.8 1.3.4.4.8.6 1.3.8.4.2 1 .3 2.1.4 1.3.1 1.7.1 4.8.1s3.5 0 4.8-.1c1.1 0 1.7-.2 2.1-.4.5-.2.9-.4 1.3-.8.4-.4.6-.8.8-1.3.2-.4.3-1 .4-2.1.1-1.3.1-1.7.1-4.8s0-3.5-.1-4.8c0-1.1-.2-1.7-.4-2.1-.2-.5-.4-.9-.8-1.3-.4-.4-.8-.6-1.3-.8-.4-.2-1-.3-2.1-.4-1.3-.1-1.7-.1-4.8-.1zm0 3a5 5 0 1 1 0 10 5 5 0 0 1 0-10zm0 8.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4zm6.4-8.4a1.2 1.2 0 1 1-2.4 0 1.2 1.2 0 0 1 2.4 0z",YT:"M23.5 6.2a3 3 0 0 0-2.1-2.1C19.5 3.5 12 3.5 12 3.5s-7.5 0-9.4.6A3 3 0 0 0 .5 6.2 31 31 0 0 0 0 12c0 1.9.2 3.9.5 5.8a3 3 0 0 0 2.1 2.1c1.9.6 9.4.6 9.4.6s7.5 0 9.4-.6a3 3 0 0 0 2.1-2.1c.3-1.9.5-3.9.5-5.8s-.2-3.9-.5-5.8zM9.6 15.6V8.4L15.8 12l-6.2 3.6z",MAIL:"M20 4H4a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2zm0 4-8 5-8-5V6l8 5 8-5v2z"};return <svg viewBox="0 0 24 24" aria-hidden="true"><path d={paths[type]||paths.MAIL}/></svg>}
function ContactFooter({settings}:{settings?:SiteSettings}){const links=[settings?.whatsapp&&{href:`https://wa.me/${settings.whatsapp.replace(/\D/g,"")}`,label:"WhatsApp",icon:"WA"},settings?.instagram_url&&{href:settings.instagram_url,label:"Instagram",icon:"IG"},settings?.youtube_url&&{href:settings.youtube_url,label:"YouTube",icon:"YT"},settings?.support_email&&{href:`mailto:${settings.support_email}`,label:"Email",icon:"MAIL"}].filter(Boolean) as {href:string;label:string;icon:string}[];return <footer className="site-footer"><div className="container footer-inner"><div className="brand"><BrandLogo logoDataURL={settings?.logo_data_url}/><span className="brand-copy"><strong>{settings?.platform_name??""}</strong><small>{settings?.platform_tagline??""}</small></span></div><div className="footer-legal-links"><Link href="/terms">Syarat Penggunaan</Link><Link href="/privacy">Kebijakan Privasi</Link></div><div className="social-links">{links.map((item)=><a key={item.label} href={item.href} target="_blank" rel="noopener noreferrer" aria-label={item.label}><SocialIcon type={item.icon}/></a>)}</div></div><p className="footer-copy">© {new Date().getFullYear()}{settings?.platform_name?` ${settings.platform_name}`:""}. Semua hak dilindungi.</p></footer>}
