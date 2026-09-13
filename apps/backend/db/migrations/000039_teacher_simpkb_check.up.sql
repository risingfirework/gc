-- Hasil pengecekan otomatis NIK guru ke portal SIMPKB (portal.simpkb.id/cari).
-- teacher_simpkb_status: pending | found | not_found | error
-- teacher_simpkb_image: screenshot hasil pencarian (data-URI PNG) sebagai bukti.
ALTER TABLE users
    ADD COLUMN teacher_simpkb_status    TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN teacher_simpkb_image     TEXT NOT NULL DEFAULT '',
    ADD COLUMN teacher_simpkb_checked_at TIMESTAMPTZ;