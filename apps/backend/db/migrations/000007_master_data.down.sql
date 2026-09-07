BEGIN;

DROP INDEX exams_tahun_ajaran_idx;
DROP INDEX exams_mapel_idx;

ALTER TABLE exams DROP COLUMN tahun_ajaran_id;
ALTER TABLE exams DROP COLUMN mapel_id;

ALTER TABLE exams ADD COLUMN subject_name VARCHAR(120) NOT NULL DEFAULT '';
CREATE INDEX exams_subject_idx ON exams (subject_name);

DROP TABLE tahun_ajaran;
DROP TABLE mapel;

COMMIT;