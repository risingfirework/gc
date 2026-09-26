import Link from "next/link";

export const metadata = { title: "Kebijakan Privasi" };

export default function PrivacyPage() {
  return <main className="legal-page"><article className="legal-card">
    <Link href="/">← Kembali ke beranda</Link>
    <h1>Kebijakan Privasi</h1><p className="muted">Terakhir diperbarui: 21 September 2026</p>
    <h2>Data yang diproses</h2><p>Platform memproses data akun, profil pendidikan, aktivitas ujian, hasil belajar, transaksi, serta catatan teknis yang diperlukan untuk keamanan dan operasional layanan. Untuk guru, data verifikasi dapat mencakup NIK dan bukti pendukung yang dikirimkan sendiri.</p>
    <h2>Tujuan penggunaan</h2><p>Data digunakan untuk menyediakan akun dan ujian, memproses pembayaran, menampilkan analisis hasil, mencegah penyalahgunaan, memberi dukungan, serta memenuhi kewajiban hukum yang berlaku.</p>
    <h2>Penyedia layanan</h2><p>Data terbatas dapat diproses oleh penyedia pembayaran, infrastruktur, email, pemantauan kesalahan, dan autentikasi yang dikonfigurasi operator. Data tidak dijual untuk periklanan.</p>
    <h2>Penyimpanan dan keamanan</h2><p>Platform menerapkan pembatasan akses, enkripsi untuk rahasia tertentu, pencatatan aktivitas administratif, serta pengamanan sesi. Tidak ada sistem yang sepenuhnya bebas risiko; laporkan dugaan insiden melalui kanal dukungan di beranda.</p>
    <h2>Hak pengguna</h2><p>Pengguna dapat meminta akses, koreksi, atau penghapusan data sesuai ketentuan yang berlaku melalui kanal dukungan. Data tertentu dapat dipertahankan bila diperlukan untuk keamanan, transaksi, sengketa, atau kewajiban hukum.</p>
  </article></main>;
}
