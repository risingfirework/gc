BEGIN;

ALTER TABLE exams ADD COLUMN subject_name VARCHAR(120) NOT NULL DEFAULT '';

CREATE INDEX exams_subject_idx ON exams (subject_name);

COMMIT;