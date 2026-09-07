BEGIN;

DROP INDEX exams_subject_idx;

ALTER TABLE exams DROP COLUMN subject_name;

COMMIT;