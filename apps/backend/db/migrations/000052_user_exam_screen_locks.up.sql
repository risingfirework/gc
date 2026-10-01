BEGIN;

-- Status layar kunci untuk satu attempt ujian CBT.
--
-- Satu baris per user_exams.id. Baris hanya dibuat saat siswa terdeteksi
-- meninggalkan aplikasi/tab. released_at diisi saat guru/admin membuka blokir
-- sehingga clients bisa membedakan "masih terkunci" dari "sudah dilepas".
CREATE TABLE user_exam_screen_locks (
    user_exam_id UUID PRIMARY KEY REFERENCES user_exams(id) ON DELETE CASCADE,
    violation_count INTEGER NOT NULL DEFAULT 1 CHECK (violation_count >= 0),
    locked_at TIMESTAMPTZ NOT NULL,
    unlock_until TIMESTAMPTZ NOT NULL,
    released_at TIMESTAMPTZ,
    last_event VARCHAR(64) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Daftar peserta CBT yang perlu menampilkan indikator terkunci.
CREATE INDEX user_exam_screen_locks_active_idx
    ON user_exam_screen_locks(unlock_until)
    WHERE released_at IS NULL;

COMMIT;