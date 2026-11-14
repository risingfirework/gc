-- Perangkat + jeda waktu CBT
ALTER TABLE user_exams
  ADD COLUMN IF NOT EXISTS device_id VARCHAR(128),
  ADD COLUMN IF NOT EXISTS paused_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS paused_seconds INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS user_exams_paused_idx ON user_exams (status, paused_at) WHERE status = 'ongoing';

-- Publisher ID pada exam (denormalisasi untuk akses cepat) - opsional, tidak wajib.
-- Tidak menambah kolom; publisher_id tetap di packages.