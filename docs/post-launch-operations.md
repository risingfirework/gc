# Post-Launch Operational Playbook — TKA CBT

Dokumen ini adalah runbook production. Semua waktu operasional memakai UTC. Jangan melakukan
restore, failover, migrasi destruktif, atau cleanup manual tanpa Incident Commander (IC) dan
pencatatan timeline insiden.

## Target reliabilitas

| Area | Target | Mekanisme |
|---|---:|---|
| API availability selama tryout | 99,9% | ≥2 replika API, health check, Nginx/load balancer |
| API p95 | <150 ms | Prometheus histogram dan alert 10 menit |
| API 5xx | <1% | Alert critical setelah 5 menit |
| RPO normal | ≤24 jam | `pg_dump` harian terenkripsi |
| RPO Tryout Akbar | ≤5 menit | managed PITR/WAL archive + hot standby lintas AZ |
| RTO Tryout Akbar | ≤15 menit | failover standby; restore dump hanya jalur terakhir |

Backup harian saja tidak cukup untuk event besar. Aktifkan automated snapshot, continuous WAL
archiving/PITR, dan satu read replica yang dapat dipromosikan pada layanan PostgreSQL terkelola.
Redis bukan source of truth permanen, tetapi selama ujian ia menyimpan jawaban terbaru; jangan
flush atau rebuild Redis ketika insiden berlangsung.

## 1. Backup PostgreSQL

### Persiapan host

Pasang PostgreSQL client 16, `aws` CLI atau Google Cloud CLI, `flock`, dan `sha256sum`. Buat
service account dengan hak minimum: upload/read hanya pada prefix backup. Pada S3 aktifkan
versioning, default SSE-KMS, block public access, dan lifecycle (contoh: 35 hari hot, 12 bulan
archive). Pada GCS gunakan uniform bucket-level access, CMEK, versioning, dan lifecycle setara.

```bash
sudo install -d -m 0750 /opt/tka /etc/tka /var/backups/tka /var/log/tka
sudo install -d -m 0755 /var/lib/node_exporter/textfile_collector
sudo chmod 0750 /opt/tka/scripts/backup-db.sh /opt/tka/scripts/restore-db.sh \
  /opt/tka/scripts/maintenance.sh /opt/tka/scripts/deploy-blue-green.sh
sudo cp deploy/cron/ops.env.example /etc/tka/ops.env
sudo chmod 0600 /etc/tka/ops.env
sudo cp deploy/cron/tka-operations.cron /etc/cron.d/tka-operations
sudo chmod 0644 /etc/cron.d/tka-operations
```

Perintah di atas mengasumsikan checkout/deployment project berada di `/opt/tka`; sesuaikan path
cron dan `PROD_ENV_FILE` jika lokasi server berbeda.

Sesuaikan `/etc/tka/ops.env`. Mode default `compose` menjalankan `pg_dump` di container PostgreSQL,
sehingga port database tetap tidak dibuka ke host. Mode `url` tersedia untuk managed PostgreSQL.
Cron menjalankan backup pukul 02:15 UTC. Script memakai lock,
custom-format dump terkompresi Zstandard, validasi `pg_restore --list`, checksum SHA-256,
server-side encryption, serta menulis metrik keberhasilan untuk node exporter.

Uji secara manual:

```bash
sudo OPS_ENV_FILE=/etc/tka/ops.env /opt/tka/scripts/backup-db.sh
aws s3 ls s3://BUCKET/tka-production/postgres/ --recursive | tail
```

Jangan menganggap upload sukses sebagai backup teruji. Setiap bulan, restore objek terbaru ke
database kosong dan jalankan smoke test login/start/sync/submit pada staging.

### Restore terkontrol

`restore-db.sh` menolak overwrite URL production dan hanya bekerja setelah guard eksplisit:

```bash
export BACKUP_OBJECT=s3://BUCKET/tka-production/postgres/2026/09/06/tka-TIMESTAMP.dump
export RESTORE_DATABASE_URL='postgres://tka:PASSWORD@recovery-db:5432/tka_recovery?sslmode=require'
export RESTORE_CONFIRM=restore-to-new-empty-database
sudo -E /opt/tka/scripts/restore-db.sh
```

Setelah restore: periksa jumlah user/ujian/attempt, constraint, aplikasi read-only, dan rekonsiliasi
transaction/payment webhook sebelum mengalihkan traffic.

## 2. Disaster recovery saat Tryout Akbar

### Triage 0–5 menit

1. IC mendeklarasikan insiden dan menunjuk Database Lead serta Communications Lead.
2. Bekukan deployment dan maintenance cron. Jangan hapus key Redis atau menjalankan seed.
3. Pastikan apakah kegagalan berada di API, connection pool, primary PostgreSQL, storage, atau AZ.
4. Simpan dashboard, log, timeline, `pg_stat_activity`, replication lag, dan status Redis Cluster.
5. Jika primary tidak pulih dalam 5 menit, pilih failover standby—jangan menunggu restore dump.

### Failover standby/PITR 5–15 menit

1. Pastikan replica lag memenuhi RPO, lalu promote melalui control-plane provider.
2. Uji koneksi/TLS dan query read/write kecil pada endpoint baru.
3. Tulis URL baru ke file secret sementara, `fsync`, lalu rename secara atomik menjadi
   `database_url.txt`; jangan mengedit file aktif secara parsial.
4. Recreate replika backend bertahap dan tunggu `/healthz` serta `/metrics` sehat.
5. Pantau 5xx, p95, connection saturation, queue webhook, dan submit ujian selama 15 menit.
6. Jangan failback pada saat event. Rekonsiliasi dan failback dilakukan setelah traffic sepi.

Jika standby tidak valid, restore PITR ke waktu terakhir sebelum korupsi dalam cluster baru. Dump
harian adalah jalur terakhir dengan RPO maksimum 24 jam. Selama PostgreSQL pulih, pertahankan
Redis: answer hash memiliki grace retention 24 jam sehingga submit dapat dilanjutkan setelah DB
tersedia. Catat attempt yang gagal submit untuk replay/reconciliation.

## 3. Monitoring, dashboard, dan alert

API Go mengekspos `/metrics` hanya di network internal. Metrik utama:

- `tka_http_requests_total{method,route,status}` untuk throughput dan error ratio.
- `tka_http_request_duration_seconds` untuk p50/p95/p99.
- `tka_http_in_flight_requests`, `go_*`, dan `process_*` untuk concurrency, CPU, RAM, GC.
- Redis exporter untuk CPU, memory, clients, commands, evictions, dan role master/replica.
- cAdvisor dan node exporter untuk container/host.

Buat `secrets/redis_exporter_passwords.json` dengan permission `0600`:

```json
{
  "redis://redis-node-1:6379": "REDIS_PASSWORD",
  "redis://redis-node-2:6379": "REDIS_PASSWORD",
  "redis://redis-node-3:6379": "REDIS_PASSWORD",
  "redis://redis-node-4:6379": "REDIS_PASSWORD",
  "redis://redis-node-5:6379": "REDIS_PASSWORD",
  "redis://redis-node-6:6379": "REDIS_PASSWORD"
}
```

Isi secret Grafana, Slack, dan Telegram, lalu jalankan overlay:

```bash
docker compose --env-file .env.production \
  -f docker-compose.prod.yml -f docker-compose.observability.yml config
docker compose --env-file .env.production \
  -f docker-compose.prod.yml -f docker-compose.observability.yml up -d
```

UI hanya bind ke loopback. Akses melalui SSH tunnel:

```bash
ssh -L 3001:127.0.0.1:3001 -L 9090:127.0.0.1:9090 ops@production-host
```

Grafana: `http://localhost:3001`; Prometheus: `http://localhost:9090`; Alertmanager:
`http://localhost:9093`. Kirim test alert sebelum event dan pastikan resolved notification sampai
ke Slack dan Telegram. Ganti placeholder `runbook_url` pada alert rules dengan URL internal.

Sentry aktif jika `sentry_dsn.txt` berisi DSN. Set `APP_RELEASE` ke image tag/Git SHA dan mulai
dengan `SENTRY_TRACES_SAMPLE_RATE=0.05`; naikkan sementara ketika investigasi, bukan menjadi
`1.0` pada traffic puncak tanpa memeriksa quota. Data pribadi tidak dikirim secara default.

## 4. Log dan Redis maintenance

Docker memakai driver `local` dengan rotasi 20 MB × 5 file per container. Aplikasi harus tetap
menulis structured JSON ke stdout/stderr; jangan menyimpan log aktif di filesystem container.
`maintenance.sh` menghapus file log host lebih tua dari 14 hari hanya di `/var/log/tka`, lalu
melakukan incremental `SCAN` pada master Redis dan `UNLINK` sesi ujian yang sudah melewati
deadline + grace 24 jam. Redis TTL tetap mekanisme cleanup utama.

Saat event berlangsung, nonaktifkan maintenance cron. Jangan memakai `KEYS`, `FLUSHDB`, atau
`FLUSHALL`. Batasi `REDIS_CLEANUP_MAX_KEYS` dan jalankan hanya saat traffic rendah.

## 5. Zero-downtime backend deployment

Prasyarat wajib:

1. Image immutable sudah lulus CI, integration test, dan canary load test.
2. Migration mengikuti pola expand/contract. Tambah kolom/table/index kompatibel terlebih dahulu;
   penghapusan/rename dilakukan pada release terpisah setelah semua instance lama hilang.
3. Sedikitnya dua replika aktif, kapasitas tersisa ≥50%, backup/PITR sehat, dan rollback image ada.

Jalankan blue/green:

```bash
export RELEASE_IMAGE=registry.example.com/tka/backend@sha256:DIGEST
export RELEASE_TAG=$(git rev-parse HEAD)
sh scripts/deploy-blue-green.sh
```

Script memilih slot tidak aktif, menjalankan migration kompatibel, menunggu container healthy,
menjalankan smoke healthcheck, mengganti upstream Nginx secara atomik, `nginx -t`, graceful
reload, menunggu connection drain, lalu menghentikan slot lama. Existing requests tetap selesai
melalui keepalive lama; request baru masuk ke slot baru.

Rollback cepat:

1. Start kembali slot/image sebelumnya.
2. Ubah `deploy/nginx/runtime/backend-target.conf` ke slot lama.
3. Jalankan `nginx -t` lalu `nginx -s reload`.
4. Jangan rollback schema setelah migration expand. Perbaiki melalui forward migration.

Untuk banyak host, gunakan rolling update Kubernetes/Swarm atau load balancer provider dengan
`maxUnavailable=0`, readiness probe, PodDisruptionBudget, dan connection draining. Script Compose
di atas adalah blue/green untuk satu host dan tidak memberi high availability terhadap kegagalan
host.

## Checklist H-1 event

- Backup terakhir <26 jam dan restore drill terbaru berhasil.
- Replication lag/PITR, Redis cluster state, disk, certificate, dan DNS sehat.
- Slack/Telegram/Sentry test event diterima oleh petugas on-call.
- Autoscaling/capacity dan limit PostgreSQL connection sudah diuji K6.
- Deployment freeze aktif; cleanup cron dihentikan.
- Daftar kontak provider, status page, template komunikasi, dan decision authority tersedia.
