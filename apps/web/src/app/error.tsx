"use client";

import Link from "next/link";
import { useEffect } from "react";

export default function ErrorPage({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error("Terjadi kesalahan saat memuat halaman:", error);
  }, [error]);

  const digest = error && typeof error === "object" && "digest" in error
    ? String((error as { digest?: string }).digest ?? "")
    : "";

  return <main className="status-page">
    <section className="status-card">
      <span className="status-code status-code-error" aria-hidden="true">500</span>
      <h1>Ada yang tidak berjalan lancar</h1>
      <p>Kami gagal memuat halaman ini. Coba muat ulang sebentar lagi. Kalau masih terjadi, kembali ke beranda atau hubungi pengelola.</p>
      <div className="status-actions">
        <button className="button" type="button" onClick={() => retry()}>Coba lagi</button>
        <Link className="button secondary" href="/">Kembali ke beranda</Link>
      </div>
      {digest && <details className="status-details">
        <summary>Detail teknis</summary>
        <code>{digest}</code>
      </details>}
    </section>
  </main>;
}
