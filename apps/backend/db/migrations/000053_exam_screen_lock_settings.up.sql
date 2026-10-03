BEGIN;

-- Pengaturan blokir layar per ujian CBT.
--
-- screen_lock_enabled menyalakan/mematikan penguncian layar untuk ujian ini.
-- screen_lock_seconds mengatur durasi kunci (detik) tiap pelanggaran.
-- Nilai default mempertahankan perilaku lama: aktif dengan 5 detik.
ALTER TABLE exams
    ADD COLUMN screen_lock_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN screen_lock_seconds INTEGER NOT NULL DEFAULT 5
        CONSTRAINT exams_screen_lock_seconds_check CHECK (screen_lock_seconds BETWEEN 1 AND 300);

COMMIT;
