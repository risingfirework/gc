# Bukti kesiapan peluncuran

Tanggal pemeriksaan terakhir: 21 September 2026.

## Gerbang otomatis yang lulus

- Backend: `go test -count=1 ./...`.
- Migrasi dan repository PostgreSQL: `go test -count=1 -tags integration ./internal/repository/postgres/ -v`.
- Pengacakan soal: urutan soal, pilihan, snapshot peserta, serta fallback snapshot parsial teruji.
- Xendit: pembuatan Payment Session, refund, autentikasi webhook, dan parsing event teruji dengan fake server.
- Security Go: `govulncheck` memakai Go 1.26.6 menghasilkan **0 reachable vulnerabilities**.
- Web: `npm audit --omit=dev --audit-level=moderate`, lint, typecheck, dan production build lulus.
- SIMPKB: lockfile reproducible, `npm audit` 0 vulnerability, `node --check server.js`, serta build image lulus. Runtime Chromium sehat, berjalan sebagai user `node` non-root, dan menolak NIK/body/route invalid dengan HTTP 400/413/404.
- Mobile: `flutter pub get`, `flutter analyze`, dan `flutter test` lulus memakai Flutter 3.44.6/Dart 3.12.2.
- Android release: Gradle hanya menandatangani release bila `key.properties` tersedia; workflow manual memvalidasi input dan secret, lalu membangun AAB bertanda tangan tanpa menyimpan keystore di repository.
- Image backend dan web berhasil dibangun. Binary backend terbukti memakai Go 1.26.6.
- Runtime image web mengembalikan HTTP 200 dan CSP nonce, `strict-dynamic`, `frame-ancestors 'none'`, `DENY`, serta `nosniff`.
- `docker compose ... config --quiet` dan sintaks `scripts/preflight-production.sh` lulus.
- Seluruh workflow YAML dapat diparse dan skrip preflight/backup/restore/bootstrap lulus ShellCheck.
- Preflight terbukti fail-closed: konfigurasi contoh ditolak karena domain placeholder.

## Gerbang wajib sebelum keputusan GO

Item berikut memerlukan infrastruktur/kredensial milik operator dan tidak dapat digantikan
oleh nilai palsu dalam repository:

- [ ] Isi identitas badan usaha/operator dan minta peninjauan legal atas Syarat Penggunaan serta Kebijakan Privasi.
- [ ] Pasang domain, DNS, sertifikat TLS nyata, seluruh secret production, SMTP, dan Sentry.
- [ ] Jalankan `sh scripts/preflight-production.sh` sampai seluruh pemeriksaan PASS.
- [ ] Jalankan satu pembayaran sukses, gagal/kedaluwarsa, webhook terlambat/duplikat, dan refund di Xendit test mode.
- [ ] Verifikasi token webhook yang dikonfigurasi di dashboard Xendit sama dengan secret deployment.
- [ ] Bangun dan tanda tangani Android App Bundle/APK dengan upload keystore production serta API URL nyata.
- [ ] Uji pemeriksaan SIMPKB end-to-end terhadap akun/situs yang sah; selector dan perilaku portal eksternal dapat berubah.
- [ ] Jalankan load test di staging setara produksi dan penuhi threshold p95/error pada README.
- [ ] Lakukan backup lalu restore drill ke database terpisah, dan catat waktu pemulihan.
- [ ] Jalankan `LIVE_CHECK=1 sh scripts/preflight-production.sh` setelah deploy staging.
- [ ] Uji perjalanan siswa dan guru: daftar, verifikasi, beli paket, mulai ujian, autosave, submit, hasil, dan pembahasan.

Status rilis saat dokumen ini dibuat: **CONDITIONAL NO-GO**. Kode dan gerbang otomatis
utama sudah lulus, tetapi peluncuran publik baru boleh berstatus GO setelah semua kotak di
atas ditandai selesai dengan bukti dari environment production/staging.
