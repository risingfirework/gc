-- Up migration: create master bab_pelajaran
CREATE TABLE bab_pelajaran (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  nama VARCHAR(160) NOT NULL,
  mapel_id UUID NULL REFERENCES mapel(id) ON DELETE SET NULL,
  jenjang VARCHAR(50) NULL,
  urutan INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (nama, COALESCE(mapel_id, '00000000-0000-0000-0000-000000000000'), COALESCE(jenjang, ''))
);

CREATE INDEX idx_bab_pelajaran_mapel ON bab_pelajaran(mapel_id);
CREATE INDEX idx_bab_pelajaran_nama ON bab_pelajaran(nama);

-- Seed 13 bab granular (opsional)
INSERT INTO bab_pelajaran (nama, urutan) VALUES
  ('Bangun Datar', 1),
  ('Bangun Ruang', 2),
  ('Bilangan Berpangkat dan Kuadrat', 3),
  ('Bilangan Bulat', 4),
  ('Bilangan Prima', 5),
  ('Kecepatan, Jarak, dan Waktu', 6),
  ('KPK dan FPB', 7),
  ('Operasi Hitung Bilangan', 8),
  ('Pecahan', 9),
  ('Pengolahan Data', 10),
  ('Pengukuran', 11),
  ('Perbandingan dan Skala', 12),
  ('Persen', 13)
ON CONFLICT DO NOTHING;
