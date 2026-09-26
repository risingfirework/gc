# TKA Backend — Step 2

Implementasi autentikasi menggunakan PostgreSQL, Redis, bcrypt, JWT HS256, dan Chi.

## Menjalankan service lokal

1. Salin `.env.example` di root menjadi `.env`, lalu ganti `JWT_SECRET` dan `PAYMENT_WEBHOOK_SECRET` dengan nilai acak yang berbeda, masing-masing minimal 32 karakter.
2. Jalankan infrastrukturnya:

   ```bash
   docker compose up -d postgres redis
   ```

3. Jalankan migration menggunakan [golang-migrate](https://github.com/golang-migrate/migrate):

   ```bash
   cd apps/backend
   migrate -path db/migrations -database "$DATABASE_URL" up
   ```

   Rollback satu migration:

   ```bash
   migrate -path db/migrations -database "$DATABASE_URL" down 1
   ```

   Di PowerShell, gunakan `$env:DATABASE_URL` sebagai pengganti `$DATABASE_URL` atau jalankan target `make migrate-up` dari shell yang mendukung Make.

4. Jalankan API:

   ```bash
   go run ./cmd/api
   ```

API tersedia di `http://localhost:8080`. Endpoint:

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout` dengan header `Authorization: Bearer <token>`
- `GET /api/v1/exams/{id}/start` untuk mulai atau melanjutkan ujian
- `POST /api/v1/cbt/answers/sync` untuk autosave jawaban
- `POST /api/v1/exams/{user_exam_id}/submit` untuk submit dan scoring
- `GET /api/v1/packages` untuk katalog paket
- `POST /api/v1/transactions/checkout` dengan `Idempotency-Key`
- `POST /api/v1/webhooks/payment` untuk callback gateway
- `GET /api/v1/exams/{user_exam_id}/result` untuk analitik dan pembahasan

Login baru menimpa JTI sebelumnya pada key Redis `tka:auth:session:<user-id>`. Akibatnya, token perangkat sebelumnya langsung ditolak middleware dengan HTTP 401.

Jawaban aktif disimpan pada Redis Hash `EXAM_ANSWERS:{<user_exam_id>}`, sedangkan deadline server berada di `EXAM_TIMER:{<user_exam_id>}`. Hash-tag yang sama menjaga kedua key berada pada slot Redis Cluster yang sama. Submit mengambil lock Redis, menggabungkan fallback answer PostgreSQL, lalu melakukan batch upsert dan perubahan status dalam satu transaksi database. Worker auto-submit berjalan setiap lima detik.

## Webhook dan gateway pembayaran

Backend memakai seam `domain.PaymentGateway` (`CreatePaymentSession`, `CreateRefund`, `VerifyWebhook`, `ParseWebhook`). Default sekarang adalah integrasi **Xendit Payment Sessions**:

- `POST /api/v1/transactions/checkout` memanggil `/sessions` dalam mode `PAYMENT_LINK` dengan Basic Auth `XENDIT_SECRET_KEY`; `reference_id` = nomor invoice dan `expires_at` mengikuti jendela pembayaran transaksi.
- Webhook Payment Session dan Refund diverifikasi lewat header `x-callback-token` yang harus sama dengan `PAYMENT_WEBHOOK_SECRET`. Konfigurasikan kedua jenis webhook Xendit ke endpoint aplikasi yang sama.
- Refund admin memanggil `POST /refunds`; transaksi tetap `refund_pending` sampai webhook `refund.succeeded` diterima, sehingga lisensi dan komisi baru dibalik setelah provider mengonfirmasi hasil akhir.
- Status di-map: `PAID`/`SETTLED` → `paid`, `EXPIRED` → `expired`, `FAILED` → `failed`. Channel di-map: `QR_CODE` → `qris`, `BANK_TRANSFER` → `virtual_account`, `EWALLET` → `e_wallet`.
- Event diproses idempoten terhadap `payment_webhook_events.event_id`.

Karena belum ada credential provider, set `XENDIT_SECRET_KEY` kosong di lingkungan non-produksi; `Checkout` akan menolak invoice berbayar sampai secret key diisi. Untuk uji sandbox, isi `secrets/xendit_secret_key.txt` dan `secrets/payment_webhook_secret.txt` lalu arahkan webhook Xendit ke `https://<domain>/api/v1/webhooks/payment`.
