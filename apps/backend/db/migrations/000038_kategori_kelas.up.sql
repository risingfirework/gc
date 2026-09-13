BEGIN;

CREATE TABLE kategori (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  nama TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE kelas (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  nama TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO kategori (nama) VALUES
  ('Tryout'), ('Ulangan Harian'), ('UTS'), ('UAS'), ('PAT')
  ON CONFLICT (nama) DO NOTHING;

INSERT INTO kelas (nama) VALUES
  ('Kelas 1 SD'), ('Kelas 2 SD'), ('Kelas 3 SD'), ('Kelas 4 SD'), ('Kelas 5 SD'), ('Kelas 6 SD'),
  ('Kelas 7 SMP'), ('Kelas 8 SMP'), ('Kelas 9 SMP'),
  ('Kelas 10 SMA'), ('Kelas 11 SMA'), ('Kelas 12 SMA')
  ON CONFLICT (nama) DO NOTHING;

ALTER TABLE packages ADD COLUMN exam_type VARCHAR(8) NOT NULL DEFAULT 'sell';
ALTER TABLE packages ADD CONSTRAINT packages_exam_type_check CHECK (exam_type IN ('sell', 'cbt'));
ALTER TABLE packages ADD COLUMN kategori_id UUID REFERENCES kategori(id) ON DELETE SET NULL;
ALTER TABLE packages ADD COLUMN kelas_id UUID REFERENCES kelas(id) ON DELETE SET NULL;

CREATE INDEX packages_kategori_idx ON packages (kategori_id);
CREATE INDEX packages_kelas_idx ON packages (kelas_id);

COMMIT;