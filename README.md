# TKA Juara

Monorepo starter untuk platform CBT Tryout TKA berbasis Go, Next.js, Flutter, PostgreSQL, dan Redis.

## Yang sudah tersedia

- API Step 2 yang dapat dijalankan: register, login, logout, bcrypt, JWT, serta single-device session lock melalui JTI Redis.
- Web responsif: landing page, login, dashboard/katalog, CBT, autosave, server timer, deteksi perpindahan tab dan hasil.
- Flutter starter: login, katalog, ujian, autosave, deteksi background dan Android `FLAG_SECURE`.
- Commerce checkout idempoten, webhook HMAC, aktivasi lisensi otomatis, scoring standar/IRT 2PL, serta analitik dan pembahasan lintas web/mobile.
- Skema PostgreSQL, seed data, Redis/PostgreSQL via Docker Compose, OpenAPI, dan CI.

## Menjalankan lokal

Cara termudah di Windows hanya memerlukan Docker Desktop:

1. Pastikan Docker Desktop sudah aktif.
2. Klik dua kali `MULAI-TKA.cmd` di root project.
3. Tunggu browser terbuka otomatis ke `http://localhost:3000`.

Migrasi dan seed demo dijalankan otomatis. Gunakan akun berikut:

```text
Email    : siswa.sma1@tka.local
Password : 12345678
```

Untuk menghentikan aplikasi, klik dua kali `HENTIKAN-TKA.cmd`. Data PostgreSQL dan
Redis tetap tersimpan. Menjalankan `docker compose down -v` akan menghapus seluruh data
lokal dan hanya boleh digunakan jika memang ingin melakukan reset.

### Otomatis aktif saat laptop menyala

Klik dua kali `PASANG-OTOMATIS.cmd` satu kali saja. Windows akan membuat Scheduled Task
`TKA Local - Auto Start` yang menjalankan aplikasi secara tersembunyi setiap kali pengguna
login. Docker Desktop akan dibuka otomatis bila belum aktif. Log startup tersimpan di
`.cache/autostart.log`.

Di laptop ini aplikasi dapat dibuka melalui `http://localhost:3000`. Perangkat lain yang
terhubung ke Wi-Fi/LAN yang sama dapat menggunakan `http://IP-LAPTOP:3000`; alamat IP LAN
ditampilkan oleh `MULAI-TKA.cmd`. Jika perangkat lain belum bisa tersambung, izinkan inbound
TCP port 3000 pada Windows Firewall untuk jaringan Private. PostgreSQL dan Redis hanya
tersedia di jaringan internal Docker dan tidak dibuka ke host maupun jaringan LAN.

Alternatif terminal (Windows PowerShell):

```powershell
.\scripts\start-local.ps1
```

Gunakan `-NoBrowser` jika browser tidak perlu dibuka otomatis. Untuk macOS/Linux:

```bash
cp .env.example .env
docker compose up -d --build --force-recreate
```

Seluruh aplikasi tersedia melalui `http://localhost:3000`. Permintaan `/api/v1/*`
diteruskan otomatis oleh Next.js ke backend di jaringan internal Docker, sehingga tidak
memerlukan port API tambahan dan tidak bentrok dengan aplikasi lokal lain.

Tambahkan `LOAD_TEST_SEED=1` saat menjalankan `seed-db.sh` untuk membuat 20.000 akun
K6 (`loadtest+1@tka.local` sampai `loadtest+20000@tka.local`). Semua akun demo
memakai `12345678`; akun beban memakai `TkaLoad123!`.

Isi `NEXT_PUBLIC_DEMO_EXAM_ID` di `.env` dengan ID ujian hasil seed/database. Untuk Android emulator, jalankan:

```bash
cd apps/mobile
flutter pub get
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080/api/v1 --dart-define=DEMO_EXAM_ID=<UUID_UJIAN>
```

Untuk perangkat Android fisik, ganti `10.0.2.2` dengan alamat IP LAN komputer backend. Folder `android` memuat integrasi native `FLAG_SECURE`; jangan menimpa `MainActivity.kt` ketika meregenerasi platform Flutter.

## Catatan arsitektur

Auth memakai adapter PostgreSQL (`pgxpool`) dan Redis (`go-redis`). Detail migration dan cara menjalankan backend tersedia di `apps/backend/README.md`.

Nilai demo menggunakan persentase jawaban benar. Kolom `difficulty` dan `discrimination` pada skema soal merupakan fondasi IRT, tetapi kalibrasi parameter dan model 2PL/3PL perlu dataset respons yang memadai serta validasi psikometrik sebelum dipakai untuk keputusan kelulusan.

## Stress test K6

Jalankan smoke test dahulu dari mesin terpisah, lalu naikkan beban bertahap. Target 20.000
VU membutuhkan load generator yang memadai dan bukan konfigurasi yang cocok untuk laptop biasa.

```bash
LOAD_TEST_SEED=1 sh scripts/seed-db.sh
k6 run -e BASE_URL=https://tka.example.com/api/v1 -e MAX_VUS=100 scripts/load-test.js
k6 run -e BASE_URL=https://tka.example.com/api/v1 -e MAX_VUS=20000 scripts/load-test.js
```

Durasi dapat diatur melalui `RAMP_UP`, `STEADY`, `RAMP_DOWN`, dan
`ANSWER_INTERVAL_SECONDS`. Threshold global adalah p95 kurang dari 150 ms dan error HTTP
kurang dari 0,1%. Hasil hanya representatif jika dijalankan pada staging yang setara produksi.

## Deployment production

1. Salin `.env.production.example` menjadi `.env.production`.
2. Buat semua file pada `secrets/README.md`, lalu pasang sertifikat TLS pada path yang
   dikonfigurasi. Jangan commit nilai rahasia.
3. Validasi dan jalankan stack:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml config
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

Migration dijalankan oleh service one-shot `migrate`. Untuk menambah replika API di satu
host gunakan `--scale backend=4`. PostgreSQL dan Redis tidak membuka port ke host; trafik
publik hanya masuk melalui Nginx HTTPS. Compose ini merupakan baseline single-host. Untuk
high availability lintas node, gunakan managed PostgreSQL/Redis atau orkestrator seperti
Kubernetes/Swarm, external load balancer, backup teruji, dan monitoring.

Panduan backup, disaster recovery, monitoring, alerting, cleanup, dan blue/green deployment
tersedia di `docs/post-launch-operations.md`.

## Struktur

- `apps/backend` — REST API Go
- `apps/web` — Next.js App Router
- `apps/mobile` — Flutter Android
- `docs/api-spec.yaml` — kontrak OpenAPI
- `apps/backend/db` — migrasi dan seed PostgreSQL
- `scripts` — helper pengembangan
