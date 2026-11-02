import Link from "next/link";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Halaman tidak ditemukan",
  description: "Halaman yang kamu cari tidak tersedia. Kembali ke beranda atau masuk ke akunmu.",
};

export default function NotFound() {
  return <main className="status-page">
    <section className="status-card">
      <span className="status-code" aria-hidden="true">404</span>
      <h1>Halaman tidak ditemukan</h1>
      <p>Alamat yang kamu buka mungkin salah ketik, sudah dipindahkan, atau memang tidak tersedia lagi.</p>
      <div className="status-actions">
        <Link className="button" href="/">Kembali ke beranda</Link>
        <Link className="button secondary" href="/login">Masuk ke akun</Link>
      </div>
      <p className="status-meta">Butuh bantuan? Hubungi pengelola platform melalui kontak yang tersedia di beranda.</p>
    </section>
  </main>;
}
