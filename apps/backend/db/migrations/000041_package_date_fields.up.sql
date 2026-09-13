ALTER TABLE packages ADD COLUMN IF NOT EXISTS start_date TIMESTAMPTZ;
ALTER TABLE packages ADD COLUMN IF NOT EXISTS end_date TIMESTAMPTZ;

-- CBT packages omit kode entirely; store NULL and keep uniqueness only for
-- non-empty codes so multiple CBT packages can coexist.
ALTER TABLE packages ALTER COLUMN kode DROP NOT NULL;
DROP INDEX IF EXISTS packages_kode_idx;
CREATE UNIQUE INDEX packages_kode_idx ON packages(kode) WHERE kode IS NOT NULL AND kode <> '';