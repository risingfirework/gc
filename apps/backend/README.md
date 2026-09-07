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

## Kontrak signature webhook

Gateway mengirim `X-Payment-Timestamp` berupa Unix timestamp dan `X-Payment-Signature` berupa hex HMAC-SHA256 dari string:

```text
<timestamp>.<raw-json-body>
```

Signature memakai `PAYMENT_WEBHOOK_SECRET` dan dibandingkan secara constant-time. Timestamp di luar toleransi lima menit ditolak. Payload harus membawa `event_id`, `invoice_number`, `payment_status`, dan `amount`; event diproses idempoten. Set `PAYMENT_CHECKOUT_URL` ke endpoint checkout adapter/provider yang kompatibel sebelum produksi.
