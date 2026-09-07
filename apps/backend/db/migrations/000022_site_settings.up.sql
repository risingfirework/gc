CREATE TABLE site_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    platform_name VARCHAR(80) NOT NULL DEFAULT 'TKA Juara',
    platform_tagline VARCHAR(180) NOT NULL DEFAULT 'Platform latihan TKA terpercaya',
    support_email VARCHAR(160) NOT NULL DEFAULT '',
    whatsapp VARCHAR(32) NOT NULL DEFAULT '',
    instagram_url VARCHAR(300) NOT NULL DEFAULT '',
    youtube_url VARCHAR(300) NOT NULL DEFAULT '',
    hero_interval_ms INTEGER NOT NULL DEFAULT 6000 CHECK (hero_interval_ms BETWEEN 2000 AND 30000),
    catalog_interval_ms INTEGER NOT NULL DEFAULT 4000 CHECK (catalog_interval_ms BETWEEN 2000 AND 30000),
    default_package_validity_days INTEGER NOT NULL DEFAULT 7 CHECK (default_package_validity_days BETWEEN 1 AND 3650),
    default_exam_duration_minutes INTEGER NOT NULL DEFAULT 60 CHECK (default_exam_duration_minutes BETWEEN 1 AND 1440),
    default_exam_total_questions INTEGER NOT NULL DEFAULT 20 CHECK (default_exam_total_questions BETWEEN 1 AND 1000),
    default_passing_score NUMERIC(5,2) NOT NULL DEFAULT 50 CHECK (default_passing_score BETWEEN 0 AND 100),
    hero_slides JSONB NOT NULL DEFAULT '[
      {"id":"adaptive","eyebrow":"Tryout adaptif & terukur","title":"Lebih siap menghadapi TKA.","lead":"Latihan seperti ujian sebenarnya, simpan jawaban otomatis, dan pahami kemampuanmu lewat analisis hasil per mata pelajaran.","stats":[["24/7","Akses latihan"],["Auto","Simpan jawaban"],["IRT","Analisis nilai"],["Aman","Kontrol ujian"]]},
      {"id":"analytics","eyebrow":"Analisis mendalam","title":"Pahami kekuatan dan kelemahanmu.","lead":"Skor per mata pelajaran, riwayat pengerjaan, dan rekomendasi belajar yang lebih terarah untuk setiap siswa.","stats":[["Detail","Skor per mapel"],["Riwayat","Semua percobaan"],["Ranking","Peringkat global"],["Insight","Rekomendasi belajar"]]},
      {"id":"realistic","eyebrow":"Ujian realistis","title":"Rasakan simulasi seperti ujian asli.","lead":"Timer berjalan di server, autosave setiap jawaban, dan deteksi perpindahan tab menjaga hasil tetap valid dan adil.","stats":[["Server","Timer akurat"],["Live","Autosave jawaban"],["Deteksi","Anti kecurangan"],["Valid","Hasil terpercaya"]]}
    ]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO site_settings(singleton) VALUES(TRUE);
