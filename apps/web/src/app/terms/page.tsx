import Link from "next/link";

export const metadata = { title: "Syarat Penggunaan" };

export default function TermsPage() {
  return <main className="legal-page"><article className="legal-card">
    <Link href="/">← Kembali ke beranda</Link>
    <h1>Syarat Penggunaan</h1><p className="muted">Terakhir diperbarui: 21 September 2026</p>
    <h2>Akun dan kelayakan</h2><p>Pengguna wajib memberikan informasi yang benar, menjaga keamanan akun, dan segera melaporkan penggunaan tanpa izin. Akun siswa dan guru hanya boleh digunakan oleh pemiliknya.</p>
    <h2>Ujian dan konten</h2><p>Soal, pembahasan, dan hasil belajar digunakan untuk kepentingan pendidikan. Pengguna dilarang menyalin, menyebarkan, mengganggu sistem ujian, atau mencoba memperoleh akses yang tidak diberikan.</p>
    <h2>Pembayaran</h2><p>Harga dan isi paket ditampilkan sebelum transaksi. Status pembayaran mengikuti konfirmasi penyedia pembayaran. Permintaan pengembalian dana ditinjau sesuai status transaksi, akses yang telah digunakan, dan kebijakan operator.</p>
    <h2>Ketersediaan layanan</h2><p>Operator berupaya menjaga layanan tetap tersedia, tetapi pemeliharaan, gangguan jaringan, atau keadaan di luar kendali dapat memengaruhi akses. Jadwal ujian penting sebaiknya disertai rencana operasional cadangan.</p>
    <h2>Penegakan</h2><p>Akses dapat dibatasi bila terjadi penipuan, pelanggaran keamanan, penyalahgunaan konten, atau pelanggaran syarat ini. Pertanyaan dan keberatan dapat diajukan melalui kanal dukungan di beranda.</p>
  </article></main>;
}
