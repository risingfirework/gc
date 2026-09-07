BEGIN;

ALTER TABLE packages ADD COLUMN kode VARCHAR(50);
ALTER TABLE packages ADD COLUMN jenjang VARCHAR(8);

UPDATE packages SET
  kode = sub.kode,
  jenjang = 'SMA'
FROM (
  SELECT id, 'TKA-' || lpad(row_number() OVER (ORDER BY created_at, id)::text, 4, '0') AS kode
  FROM packages
) sub
WHERE packages.id = sub.id;

ALTER TABLE packages ALTER COLUMN kode SET NOT NULL;
ALTER TABLE packages ALTER COLUMN jenjang SET NOT NULL;

ALTER TABLE packages ADD CONSTRAINT packages_jenjang_check CHECK (jenjang IN ('SD', 'SMP', 'SMA'));
CREATE UNIQUE INDEX packages_kode_idx ON packages (kode);

COMMIT;