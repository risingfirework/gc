BEGIN;

DROP INDEX packages_kode_idx;
ALTER TABLE packages DROP CONSTRAINT packages_jenjang_check;
ALTER TABLE packages DROP COLUMN jenjang;
ALTER TABLE packages DROP COLUMN kode;

COMMIT;