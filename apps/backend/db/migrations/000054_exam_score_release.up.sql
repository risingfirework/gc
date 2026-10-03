BEGIN;

-- Kapan nilai ujian dirilis ke siswa (khusus mode CBT).
--   after_finish    : nilai muncul begitu siswa selesai atau waktu habis.
--   with_pembahasan : nilai muncul bersamaan saat pembahasan dipublikasikan.
-- Pembahasan/analitik rinci tetap dikendalikan kolom publish_pembahasan.
ALTER TABLE exams
    ADD COLUMN score_release VARCHAR(20) NOT NULL DEFAULT 'after_finish'
    CONSTRAINT exams_score_release_check CHECK (score_release IN ('after_finish', 'with_pembahasan'));

COMMIT;
