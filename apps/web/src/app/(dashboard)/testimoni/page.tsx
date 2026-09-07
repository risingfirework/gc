"use client";

import Link from "next/link";
import BrandLogo from "@/components/BrandLogo";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { APIError, api, Testimonial, tokenStore } from "@/services/api";
import StudentMenu from "@/components/StudentMenu";

export default function TestimoniInputPage() {
  const router = useRouter();
  const [myTestimonial, setMyTestimonial] = useState<Testimonial | null>(null);
  const [quote, setQuote] = useState("");
  const [saving, setSaving] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!tokenStore.hasToken()) {
      router.replace("/login");
      return;
    }
    api
      .currentUser()
      .then((current) => {
        if (current.role !== "student") {
          router.replace("/dashboard");
          return null;
        }
        return api.getMyTestimonial();
      })
      .then((t) => { if (t) { setMyTestimonial(t); setQuote(t.quote); } })
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, [router]);

  async function logout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try { await api.logout(); } finally { router.replace("/"); }
  }

  async function submit() {
    if (!quote.trim() || saving) return;
    setSaving(true);
    setError("");
    try {
      const t = await api.submitTestimonial(quote.trim());
      setMyTestimonial(t);
    } catch (e) {
      setError(e instanceof APIError ? e.message : "Gagal mengirim testimoni.");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return <main><div className="container testimoni-page"><div className="exam-loader"/><p>Memuat...</p></div></main>;
  }

  return (
    <main>
      <div className="container testimoni-page">
        <nav className="dashboard-nav">
          <Link className="brand" href="/dashboard"><BrandLogo/>TKA Juara</Link>
          <StudentMenu active="testimoni" loggingOut={loggingOut} onLogout={() => void logout()} />
        </nav>
        <section className="dashboard">
          <section className="dashboard-section panel-view">
            <div className="section-heading"><div><p className="eyebrow">Testimoni</p><h2>Ceritakan pengalamanmu</h2></div></div>
            <div className="card">
              {error && <p className="error">{error}</p>}
              {myTestimonial ? (
                <div className="testimonial-submitted">
                  <h3>Terima kasih!</h3>
                  <p className="muted">Testimoni Anda sudah kami terima dan sedang dipertimbangkan untuk ditampilkan di halaman utama.</p>
                  <blockquote>&ldquo;{myTestimonial.quote}&rdquo;</blockquote>
                </div>
              ) : (
                <div>
                  <p className="muted">Tulis pengalamanmu menggunakan TKA Juara. Testimoni terbaik akan ditampilkan di halaman utama.</p>
                  <textarea value={quote} onChange={(event) => setQuote(event.target.value)} placeholder="Tulis testimoni Anda di sini..." style={{ minHeight: 140, width: "100%" }} />
                  <div className="form-actions"><button className="button" disabled={saving || !quote.trim()} onClick={() => void submit()}>{saving ? "Mengirim..." : "Kirim testimoni"}</button></div>
                </div>
              )}
            </div>
          </section>
        </section>
      </div>
    </main>
  );
}
