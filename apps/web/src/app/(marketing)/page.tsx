"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { api, Package, SiteSettings } from "@/services/api";
import TestimonialSlider, { SliderArrows } from "@/components/TestimonialSlider";
import HeroSlider from "@/components/HeroSlider";
import CatalogStats from "@/components/CatalogStats";
import BrandLogo from "@/components/BrandLogo";

const rupiah = new Intl.NumberFormat("id-ID", { style:"currency", currency:"IDR", maximumFractionDigits:0 });

export default function LandingPage() {
  const router=useRouter(); const openAuth=()=>router.push("/login"); const [settings,setSettings]=useState<SiteSettings>();
  useEffect(()=>{api.siteSettings().then(setSettings).catch(()=>undefined)},[]);
  return <main className="landing">
    <div className="container"><nav className="nav"><div className="brand"><BrandLogo logoDataURL={settings?.logo_data_url}/><span className="brand-copy"><strong>{settings?.platform_name??"TKA Juara"}</strong><small>{settings?.platform_tagline??"Platform latihan TKA terpercaya"}</small></span></div><button className="button secondary" onClick={openAuth}>Masuk</button></nav></div>
    <HeroSlider onStart={openAuth} showPaket slides={settings?.hero_slides} intervalMS={settings?.hero_interval_ms}/>
    <div className="container"><PackagesSlider onStart={openAuth} intervalMS={settings?.catalog_interval_ms}/><TestimonialSlider/></div>
    <ContactFooter settings={settings}/>
  </main>;
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

function ContactFooter({settings}:{settings?:SiteSettings}){const links=[settings?.whatsapp&&{href:`https://wa.me/${settings.whatsapp.replace(/\D/g,"")}`,label:"WhatsApp",icon:"WA"},settings?.instagram_url&&{href:settings.instagram_url,label:"Instagram",icon:"IG"},settings?.youtube_url&&{href:settings.youtube_url,label:"YouTube",icon:"YT"},settings?.support_email&&{href:`mailto:${settings.support_email}`,label:"Email",icon:"@"}].filter(Boolean) as {href:string;label:string;icon:string}[];return <footer className="site-footer"><div className="container footer-inner"><div className="brand"><BrandLogo logoDataURL={settings?.logo_data_url}/><span className="brand-copy"><strong>{settings?.platform_name??"TKA Juara"}</strong><small>{settings?.platform_tagline??"Platform latihan TKA terpercaya"}</small></span></div><div className="social-links">{links.map((item)=><a key={item.label} href={item.href} target="_blank" rel="noopener noreferrer" aria-label={item.label}>{item.icon}</a>)}</div></div><p className="footer-copy">© {new Date().getFullYear()} {settings?.platform_name??"TKA Juara"}. Semua hak dilindungi.</p></footer>}
