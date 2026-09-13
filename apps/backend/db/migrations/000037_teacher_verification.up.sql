-- Verifikasi guru: NIK (KTP), status persetujuan operator/admin, alasan
-- penolakan, dan bukti sanggah (gambar/surat keterangan mengajar).

ALTER TABLE users
    ADD COLUMN teacher_ktp VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN teacher_verification_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    ADD COLUMN teacher_rejection_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN teacher_appeal_image TEXT NOT NULL DEFAULT '',
    ADD COLUMN teacher_verified_at TIMESTAMPTZ;

ALTER TABLE users
    ADD CONSTRAINT users_teacher_verification_status_check
    CHECK (teacher_verification_status IN ('pending', 'approved', 'rejected'));

CREATE INDEX idx_users_teacher_verification
    ON users (teacher_verification_status)
    WHERE role = 'teacher' AND teacher_verification_status IN ('pending', 'rejected');

-- Guru yang sudah terdaftar langsung disetujui agar tidak terkunci.
UPDATE users SET teacher_verification_status = 'approved', teacher_verified_at = NOW()
WHERE role = 'teacher' AND teacher_verification_status = 'pending';