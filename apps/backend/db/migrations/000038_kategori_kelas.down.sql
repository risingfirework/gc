BEGIN;

DROP INDEX IF EXISTS packages_kelas_idx;
DROP INDEX IF EXISTS packages_kategori_idx;

ALTER TABLE packages DROP CONSTRAINT IF EXISTS packages_exam_type_check;
ALTER TABLE packages DROP COLUMN IF EXISTS kelas_id;
ALTER TABLE packages DROP COLUMN IF EXISTS kategori_id;
ALTER TABLE packages DROP COLUMN IF EXISTS exam_type;

DROP TABLE IF EXISTS kelas;
DROP TABLE IF EXISTS kategori;

COMMIT;