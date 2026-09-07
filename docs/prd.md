# Product Requirement Document — Platform CBT Tryout TKA

## Ringkasan

Platform B2C untuk siswa SD, SMP, SMA/SMK dan orang tua. Produk menyediakan katalog paket, ujian CBT lintas Web/Android, penyimpanan jawaban otomatis, kontrol anti-kecurangan, pembayaran, serta hasil dan pembahasan.

## Sasaran teknis

- API Go dengan target respons transaksi ujian di bawah 150 ms.
- PostgreSQL sebagai sumber data utama dan Redis untuk sesi, rate limit, serta state jawaban sementara.
- Next.js untuk landing/katalog SSR dan CBT CSR.
- Flutter untuk Android, termasuk `FLAG_SECURE` dan pencatatan aplikasi yang masuk ke background.
- Target kapasitas 20.000 peserta ujian aktif dan ketersediaan 99,9% saat event.

## Kebutuhan fungsional

1. Login email/password dan Google OAuth, profil serta jenjang pendidikan.
2. Satu perangkat aktif per akun, dengan pilihan force logout perangkat lama.
3. Katalog paket satuan/bundel, diskon dan masa berlaku.
4. Webhook pembayaran mengaktifkan lisensi secara idempoten.
5. Jawaban disimpan langsung ke Redis dan dipersistenkan asinkron ke PostgreSQL.
6. Timer memakai waktu server dan submit otomatis ketika waktu berakhir.
7. Konten mendukung LaTeX/KaTeX, gambar, bacaan panjang dan video pembahasan.
8. Web mencatat perpindahan tab dan memblokir interaksi terlarang; Android memblokir screenshot/rekaman layar dan mencatat background event.
9. Nilai otomatis, status lulus, breakdown mata pelajaran dan kesiapan untuk model IRT.

## Batasan starter

Implementasi awal di repositori ini memakai penyimpanan in-memory agar demo dapat dijalankan tanpa dependensi Go eksternal. Skema PostgreSQL dan layanan Redis telah disiapkan. Adapter produksi, Google OAuth, payment provider nyata, object storage, worker sinkronisasi jawaban dan estimasi parameter IRT merupakan tahap integrasi berikutnya.
