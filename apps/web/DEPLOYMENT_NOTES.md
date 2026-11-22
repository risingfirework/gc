# Catatan deploy

Workflow `Deploy to VPS` hanya menunggu workflow CI yang ikut berjalan untuk commit yang sama.
`Backend CI` berjalan hanya bila ada perubahan di `apps/backend/**`, sehingga commit yang
hanya menyentuh `apps/web/**` tetap bisa deploy (perubahan backend yang sudah ada di `main`
ikut terpasang).

Perbaikan backend pada commit `2150b87` (rate-limit refresh per hash token dan
`forgot-password` per email) sudah diverifikasi lokal dengan `go build`, `go vet`,
dan `go test ./...`. Backend CI untuk commit tersebut gagal pada langkah yang sudah
gagal pada commit-commit sebelumnya (pemeriksaan `gofmt` dan tes integrasi Postgres),
bukan pada kode yang diubah. Perbaikan CI tersebut menyusul setelah ujian.
