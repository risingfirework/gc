# Prosedur validasi sebelum publik

Dokumen ini menghasilkan bukti yang harus dilampirkan pada keputusan GO. Gunakan akun dan
environment **test/staging**; jangan menyalin secret ke tiket, chat, log, atau repository.

## 1. Xendit test mode

1. Isi Xendit test secret key dan webhook verification token pada secret staging.
2. Atur callback dashboard ke `https://STAGING_DOMAIN/api/v1/webhooks/payment`.
3. Buat transaksi baru dari akun siswa dan simpan ID transaksi internal, `reference_id`,
   `payment_session_id`, serta `payment_request_id`.
4. Jalankan skenario berhasil, gagal, kedaluwarsa, webhook yang dikirim ulang, dan refund.
5. Untuk setiap skenario, cocokkan status database, dashboard Xendit, lisensi paket, dan audit log.
6. Webhook duplikat wajib menghasilkan status akhir yang sama tanpa lisensi atau refund ganda.
7. Refund awal boleh berstatus `refund_pending`; status akhir hanya berubah dari webhook
   `refund.succeeded` atau `refund.failed`.

Bukti minimum: timestamp UTC, ID internal/provider, status HTTP callback, status sebelum/sesudah,
dan screenshot dashboard yang sudah menyamarkan data pribadi. Jangan mengaktifkan live key sebelum
seluruh skenario lulus.

## 2. Smoke test staging

```bash
ENV_FILE=.env.staging LIVE_CHECK=1 sh scripts/preflight-production.sh
```

Perintah memeriksa secret/TLS/Compose, halaman utama, privasi, syarat penggunaan, katalog API,
HSTS, dan CSP. Simpan output terminal sebagai artefak release.

## 3. Load test bertahap

Jalankan dari mesin terpisah dan mulai dari 100 VU. Hentikan eskalasi bila error rate mencapai
0,1%, p95 mencapai 150 ms, database/Redis mengalami saturation, atau autosave gagal.

```bash
k6 run -e BASE_URL=https://STAGING_DOMAIN/api/v1 -e MAX_VUS=100 scripts/load-test.js
k6 run -e BASE_URL=https://STAGING_DOMAIN/api/v1 -e MAX_VUS=1000 scripts/load-test.js
k6 run -e BASE_URL=https://STAGING_DOMAIN/api/v1 -e MAX_VUS=5000 scripts/load-test.js
k6 run -e BASE_URL=https://STAGING_DOMAIN/api/v1 -e MAX_VUS=20000 scripts/load-test.js
```

Simpan ringkasan K6, dashboard CPU/RAM, pool PostgreSQL, Redis latency/eviction, p95/p99, dan error
rate. Angka 20.000 hanya dinyatakan lulus bila staging mempunyai kapasitas setara produksi.

## 4. Backup dan restore drill

1. Jalankan `scripts/backup-db.sh` dan pastikan dump serta SHA-256 berada di object storage.
2. Buat database recovery baru dan kosong; jangan memakai URL production.
3. Jalankan `scripts/restore-db.sh` dengan guard `RESTORE_CONFIRM=restore-to-new-empty-database`.
4. Jalankan smoke test login, mulai ujian, autosave, submit, hasil, serta rekonsiliasi transaksi.
5. Catat waktu backup, ukuran, waktu restore, RPO aktual, RTO aktual, dan orang yang menyetujui.

## 5. Android signed release

Tambahkan empat secret pada GitHub Environment `production`:

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_STORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

Jalankan workflow **Mobile signed release**, masukkan URL API produksi, version name, dan version
code. Workflow menolak secret kosong, menjalankan analyze/test, membuat AAB bertanda tangan, dan
menyimpan artefak selama 14 hari. Upload ke Play Console internal testing sebelum production.

## 6. Keputusan GO

GO hanya dapat diberikan bila preflight dan seluruh skenario di atas lulus, backup dapat dipulihkan,
alert test diterima petugas on-call, pemilik bisnis menyetujui dokumen legal, dan rollback image
lama masih tersedia.
