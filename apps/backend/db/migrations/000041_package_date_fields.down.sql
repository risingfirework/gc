DROP INDEX IF EXISTS packages_kode_idx;
CREATE UNIQUE INDEX packages_kode_idx ON packages(kode);
ALTER TABLE packages ALTER COLUMN kode SET NOT NULL;
ALTER TABLE packages DROP COLUMN IF EXISTS start_date;
ALTER TABLE packages DROP COLUMN IF EXISTS end_date;