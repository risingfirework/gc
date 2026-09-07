BEGIN;

CREATE TABLE mapel (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  nama TEXT NOT NULL UNIQUE
);

CREATE TABLE tahun_ajaran (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  nama TEXT NOT NULL UNIQUE
);

INSERT INTO mapel (nama) VALUES
  ('Matematika'), ('Bahasa Indonesia'), ('Bahasa Inggris'), ('IPA'), ('IPS'), ('PJOK')
  ON CONFLICT (nama) DO NOTHING;

INSERT INTO tahun_ajaran (nama) VALUES ('2024/2025'), ('2025/2026')
  ON CONFLICT (nama) DO NOTHING;

DROP INDEX exams_subject_idx;

ALTER TABLE exams DROP COLUMN subject_name;

ALTER TABLE exams ADD COLUMN mapel_id UUID REFERENCES mapel(id) ON DELETE SET NULL;
ALTER TABLE exams ADD COLUMN tahun_ajaran_id UUID REFERENCES tahun_ajaran(id) ON DELETE SET NULL;

CREATE INDEX exams_mapel_idx ON exams (mapel_id);
CREATE INDEX exams_tahun_ajaran_idx ON exams (tahun_ajaran_id);

COMMIT;