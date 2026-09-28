# Deploy TKA ke VPS kosong (Ubuntu 24.04 LTS)

Runbook langkah demi langkah untuk server yang benar-benar kosong.
Contoh memakai: domain `siap-siap.com`, IP publik `43.173.33.115`, user `ubuntu`.

> Ganti `siap-siap.com` dan `43.173.33.115` dengan nilai Anda bila berbeda.

## Sebelum mulai (di laptop / panel VPS)

1. **DNS**: tambah/ubah record A di penyedia domain Anda.
   ```
   siap-siap.com  A  43.173.33.115
   ```
   Tunggu propagasi, verifikasi dari laptop:
   ```
   Resolve-DnsName siap-siap.com
   ```
2. **Firewall VPS**: pastikan port berikut terbuka dari internet:
   - `22/tcp` (SSH)
   - `80/tcp` (HTTP, untuk certbot)
   - `443/tcp` (HTTPS)
   Di panel cloud VPS Anda, ini biasanya *Security Group* atau *Firewall Rules*.
3. Siapkan **password/SSH key** untuk `ubuntu@43.173.33.115`.

## Langkah 1 — SSH masuk ke VPS

Buka terminal lokalan Anda lalu:

```
ssh ubuntu@43.173.33.115
```

Anda akan diminta password (atau sudah pakai SSH key). Setelah masuk, prompt menjadi
`ubuntu@<hostname>:~$`.

## Langkah 2 — Update sistem (opsional tapi disarankan)

```
sudo apt-get update
sudo apt-get upgrade -y
```

## Langkah 3 — Clone repo

```
git clone https://github.com/risingfirework/gc.git ~/tka
cd ~/tka
```

## Langkah 4 — Bootstrap (install docker, certbot, generate secret, ambil TLS)

Jalankan sekali. Skrip akan:

- install Docker Engine + Compose plugin
- install certbot
- generate `secrets/*.txt` acak (postgres, redis, jwt, webhook, database_url)
- buat `.env.production` (domain + release diisi otomatis)
- ambil sertifikat TLS via certbot (gunakan port 80)

```
APP_DOMAIN=siap-siap.com sh scripts/bootstrap-vps.sh
```

Selama proses, certbot akan bertanya beberapa hal — ikuti prompt. Bila domain belum
mengarah ke IP, certbot gagal; perbaiki DNS lalu ulangi langkah ini.

> Setelah skrip selesai, **logout lalu login ulang** (`exit`, lalu `ssh` lagi) agar
> grup `docker` aktif untuk user `ubuntu`.

## Langkah 5 — Isi kredensial nyata

File berikut sudah dibuat oleh bootstrap tetapi nilainya masih kosong/placeholder.
Isi hanya yang memang Anda miliki. **Jangan kosongkan** (preflight menolak file kosong);
bila fitur belum dipakai, isi nilai penanda seperti `disabled` di file dan kosongkan
variabel terkait di `.env.production`.

```
nano secrets/xendit_secret_key.txt      # xnd_... (dari dashboard Xendit)
nano secrets/sentry_dsn.txt             # DSN Sentry, atau "disabled"
nano secrets/smtp_password.txt          # password SMTP, atau "disabled"
nano .env.production
```

Di `.env.production`, periksa dan sesuaikan:

```
SMTP_HOST=mail.anda.com       # kosongkan bila SMTP nonaktif
SMTP_PORT=587
SMTP_USERNAME=no-reply@anda.com
SMTP_FROM=no-reply@anda.com
XENDIT_BASE_URL=https://api.xendit.co
APP_RELEASE=2026.09.26-1
```

Catatan tentang perilaku runtime:
- `SentryDSN` kosong → Sentry tidak diinisialisasi (aman).
- `SMTPHost` kosong → outbound email nonaktif (aman).
- `xendit_secret_key.txt` berisi `disabled` → pembayaran tidak aktif sampai owner
  mengisi konfigurasi lewat panel admin (halaman Integrasi Pembayaran), yang tersimpan
  terenkripsi di database.

## Langkah 6 — Preflight

```
cd ~/tka
sh scripts/preflight-production.sh
```

Harus tampil `PASS` untuk semua baris. Bila ada `FAIL`, ikuti pesannya
(umumnya: DNS belum propagate, cert kedaluwarsa, atau format secret salah).

## Langkah 7 — Migrasi & start layanan

Siapkan folder untuk data Postgres/Redis (opsional, disarankan di luar repo):

```
sudo mkdir -p /var/lib/tka
```

Mulai layanan (migrasi dijalankan dulu, sekali):

```
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Perintah kedua akan build image backend/web di VPS (butuh beberapa menit).
Cek status:

```
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

Semua service harus `Up` dan `healthy`.

## Langkah 8 — Smoke test live

```
cd ~/tka
LIVE_CHECK=1 sh scripts/preflight-production.sh
```

Memeriksa: `https://siap-siap.com/`, `/privacy`, `/terms`, `/api/v1/packages`,
serta header keamanan HSTS/CSP.

## Langkah 9 — Buat akun owner pertama

```
cd ~/tka
CREATE_OWNER_PASSWORD='GANTI_DENGAN_PASSWORD_KUAT' scripts/create-owner.sh admin@siap-siap.com
```

## Langkah 10 — Periksa log & backup

```
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
```

Pasang backup harian (opsional, lihat `scripts/backup-db.sh` dan
`deploy/cron/tka-operations.cron`).

## Referensi cepat beberapa perintah

- Stop: `docker compose --env-file .env.production -f docker-compose.prod.yml down`
- Restart nginx: `docker compose --env-file .env.production -f docker-compose.prod.yml restart nginx`
- Log: `... logs -f backend|web|nginx`
- Update kode: `git pull --ff-only origin main`, lalu ulangi Langkah 7.
- Renew certbot: `sudo certbot renew` (atau pasang cron).