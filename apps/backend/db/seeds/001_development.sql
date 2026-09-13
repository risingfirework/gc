BEGIN;

-- ============================================================
-- SEED DEMO LOKAL
-- Semua akun demo menggunakan password: 12345678
--
-- Konten demo (masing-masing 3):
--   * 3 akun siswa      : siswa.sd1 / siswa.smp1 / siswa.sma1 (SD, SMP, SMA)
--   * 9 bank soal (paket): 3 SD, 3 SMP, 3 SMA
--   * 9 ujian            : 1 per paket/bank soal
--   * 3 transaksi        : pembelian paket berbayar
--   * 3 hasil ujian      : data menu Ranking Global
--   * 3 testimoni        : 2 disetujui + 1 menunggu
--   * 3 komisi guru      : biaya unggah + 2 bonus penjualan
-- Ditambah akun staff (owner/finance/operator).
-- Guru: guru.sma1 (sudah diverifikasi, pemilik bank soal SMA) + 1 akun guru
-- baru yang masih belum diverifikasi (guru.smp1, status pending).
-- ============================================================

-- ---------- AKUN ----------
WITH password AS MATERIALIZED (SELECT crypt('12345678', gen_salt('bf', 10)) AS hash)
INSERT INTO users (id, email, password_hash, role, school_level, name, teacher_ktp, teacher_verification_status)
SELECT data.id::uuid, data.email, password.hash, data.role, data.school_level, data.name, data.teacher_ktp, data.teacher_verification_status
FROM password CROSS JOIN (VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin@tka.local', 'owner', 'SMA', 'Pemilik Platform', '', 'pending'),
    ('10000000-0000-0000-0000-000000000002', 'finance@tka.local', 'finance', 'SMA', 'Finance Platform', '', 'pending'),
    ('10000000-0000-0000-0000-000000000003', 'operator@tka.local', 'admin', 'SMA', 'Operator Platform', '', 'pending'),
    ('10000000-0000-0000-0000-000000000201', 'guru.sma1@tka.local', 'teacher', 'SMA', 'Rina Kusuma', '3273012345678920', 'approved'),
    ('10000000-0000-0000-0000-000000000202', 'guru.smp1@tka.local', 'teacher', 'SMP', 'Budi Santoso', '3273054321098765', 'pending'),
    ('10000000-0000-0000-0000-000000000101', 'siswa.sd1@tka.local', 'student', 'SD', 'Raka Pratama', '', 'pending'),
    ('10000000-0000-0000-0000-000000000102', 'siswa.smp1@tka.local', 'student', 'SMP', 'Bimo Ardiansyah', '', 'pending'),
    ('10000000-0000-0000-0000-000000000103', 'siswa.sma1@tka.local', 'student', 'SMA', 'Dewa Mahendra', '', 'pending')
) AS data(id, email, role, school_level, name, teacher_ktp, teacher_verification_status)
ON CONFLICT (id) DO UPDATE SET email=EXCLUDED.email, password_hash=EXCLUDED.password_hash,
role=EXCLUDED.role, school_level=EXCLUDED.school_level, name=EXCLUDED.name,
teacher_ktp=EXCLUDED.teacher_ktp, teacher_verification_status=EXCLUDED.teacher_verification_status,
teacher_rejection_reason='', teacher_appeal_image='',
teacher_verified_at=CASE WHEN EXCLUDED.teacher_verification_status='approved' THEN NOW() ELSE NULL END,
updated_at=NOW();

-- ---------- KATEGORI TES & KELAS ----------
INSERT INTO kategori (nama) VALUES ('Tryout'), ('Ulangan Harian'), ('UTS'), ('UAS'), ('PAT')
ON CONFLICT (nama) DO NOTHING;

INSERT INTO kelas (nama) VALUES
('Kelas 1 SD'), ('Kelas 2 SD'), ('Kelas 3 SD'), ('Kelas 4 SD'), ('Kelas 5 SD'), ('Kelas 6 SD'),
('Kelas 7 SMP'), ('Kelas 8 SMP'), ('Kelas 9 SMP'),
('Kelas 10 SMA'), ('Kelas 11 SMA'), ('Kelas 12 SMA')
ON CONFLICT (nama) DO NOTHING;

-- ---------- BANK SOAL (PAKET), 3 BUAH ----------
INSERT INTO packages (id, title, description, price, validity_days, kode, jenjang, exam_type, cbt_token, kategori_id, kelas_id) VALUES
('20000000-0000-0000-0000-000000000001','Tryout TKA Eceran','Satu simulasi lengkap dengan pembahasan.',0,7,'TKA-0001','SMA','cbt','9K4PT2',(SELECT id FROM kategori WHERE nama='Tryout'),(SELECT id FROM kelas WHERE nama='Kelas 12 SMA')),
('20000000-0000-0000-0000-000000000003','Tryout Gratis TKA','Simulasi tryout gratis untuk mencoba pengalaman ujian penuh.',0,7,'TKA-0008','SMA','cbt',NULL,(SELECT id FROM kategori WHERE nama='Tryout'),(SELECT id FROM kelas WHERE nama='Kelas 12 SMA'))
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price,
validity_days=EXCLUDED.validity_days, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang,
exam_type=EXCLUDED.exam_type, cbt_token=EXCLUDED.cbt_token, kategori_id=EXCLUDED.kategori_id, kelas_id=EXCLUDED.kelas_id;

-- Bank soal milik guru demo (akun guru.sma1@tka.local / 12345678).
INSERT INTO packages (id, title, description, price, validity_days, publisher_id, kode, jenjang, exam_type, cbt_token, kategori_id, kelas_id) VALUES
('20000000-0000-0000-0000-000000000002','Bank Soal Matematika Bu Rina','Kumpulan soal latihan Matematika karya guru bersertifikat.',15000,14,'10000000-0000-0000-0000-000000000201','TKA-0003','SMA','sell',NULL,(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 11 SMA')),
('20000000-0000-0000-0000-000000000004','Tryout CBT Matematika Bu Rina','Tryout CBT Matematika untuk kelas 12 karya guru bersertifikat.',0,14,'10000000-0000-0000-0000-000000000201','TKA-CBT1','SMA','cbt','SMA2TKA',(SELECT id FROM kategori WHERE nama='Tryout'),(SELECT id FROM kelas WHERE nama='Kelas 12 SMA'))
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price,
validity_days=EXCLUDED.validity_days, publisher_id=EXCLUDED.publisher_id, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang,
exam_type=EXCLUDED.exam_type, cbt_token=EXCLUDED.cbt_token, kategori_id=EXCLUDED.kategori_id, kelas_id=EXCLUDED.kelas_id;

-- ---------- UJIAN, 4 BUAH ----------
INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method) VALUES
('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','Simulasi TKA Matematika dan Literasi',30,10,70,'standard'),
('30000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000002','Latihan Matematika Bab Aljabar',20,5,60,'standard'),
('30000000-0000-0000-0000-000000000003','20000000-0000-0000-0000-000000000003','Tryout Percobaan Gratis',10,2,50,'standard'),
('30000000-0000-0000-0000-000000000004','20000000-0000-0000-0000-000000000004','Tryout CBT Matematika',30,8,70,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title,
duration_minutes=EXCLUDED.duration_minutes, total_questions=EXCLUDED.total_questions,
passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

-- ---------- SOAL, 17 BUAH ----------
INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
-- Ujian 1: Simulasi TKA (5 Matematika + 5 Literasi)
('40000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Matematika','Jika $2x + 6 = 16$, nilai $x$ adalah ...','[{"key":"A","content":"3"},{"key":"B","content":"4"},{"key":"C","content":"5"},{"key":"D","content":"6"}]','C',1,'Kurangi kedua ruas dengan 6, sehingga $2x=10$ dan $x=5$.',-1.2,1.0,NULL),
('40000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000001','Matematika','Nilai dari $3^2 + 4^2$ adalah ...','[{"key":"A","content":"7"},{"key":"B","content":"12"},{"key":"C","content":"25"},{"key":"D","content":"49"}]','C',1,'$3^2+4^2=9+16=25$.',-1.0,1.1,NULL),
('40000000-0000-0000-0000-000000000003','30000000-0000-0000-0000-000000000001','Matematika','Luas persegi dengan sisi 8 cm adalah ...','[{"key":"A","content":"16 cm²"},{"key":"B","content":"32 cm²"},{"key":"C","content":"64 cm²"},{"key":"D","content":"80 cm²"}]','C',1,'Luas persegi $L=s^2=8^2=64$ cm².',-0.9,1.0,NULL),
('40000000-0000-0000-0000-000000000004','30000000-0000-0000-0000-000000000001','Matematika','Hasil $\frac{3}{4}+\frac{1}{8}$ adalah ...','[{"key":"A","content":"$\\frac{1}{2}$"},{"key":"B","content":"$\\frac{5}{8}$"},{"key":"C","content":"$\\frac{7}{8}$"},{"key":"D","content":"1"}]','C',1,'Samakan penyebut: $\frac{6}{8}+\frac{1}{8}=\frac{7}{8}$.',-0.4,1.2,NULL),
('40000000-0000-0000-0000-000000000005','30000000-0000-0000-0000-000000000001','Matematika','Rata-rata dari 6, 8, 10, dan 12 adalah ...','[{"key":"A","content":"8"},{"key":"B","content":"9"},{"key":"C","content":"10"},{"key":"D","content":"11"}]','B',1,'Jumlah data 36 dibagi 4 menghasilkan 9.',-0.2,1.0,NULL),
('40000000-0000-0000-0000-000000000006','30000000-0000-0000-0000-000000000001','Literasi','Kalimat utama dalam paragraf umumnya memuat ...','[{"key":"A","content":"Ide pokok"},{"key":"B","content":"Contoh tambahan"},{"key":"C","content":"Nama penulis"},{"key":"D","content":"Daftar pustaka"}]','A',1,'Kalimat utama menyatakan ide pokok.',-0.8,0.9,NULL),
('40000000-0000-0000-0000-000000000007','30000000-0000-0000-0000-000000000001','Literasi','Sinonim kata "cermat" adalah ...','[{"key":"A","content":"Ceroboh"},{"key":"B","content":"Teliti"},{"key":"C","content":"Lambat"},{"key":"D","content":"Keras"}]','B',1,'Cermat dan teliti bermakna saksama.',-0.6,1.0,NULL),
('40000000-0000-0000-0000-000000000008','30000000-0000-0000-0000-000000000001','Literasi','Penulisan kata baku yang tepat adalah ...','[{"key":"A","content":"Aktifitas"},{"key":"B","content":"Resiko"},{"key":"C","content":"Analisis"},{"key":"D","content":"Ijin"}]','C',1,'Bentuk bakunya: aktivitas, risiko, analisis, dan izin.',-0.3,1.1,NULL),
('40000000-0000-0000-0000-000000000009','30000000-0000-0000-0000-000000000001','Literasi','Teks yang menjelaskan proses terjadinya fenomena disebut teks ...','[{"key":"A","content":"Narasi"},{"key":"B","content":"Eksplanasi"},{"key":"C","content":"Negosiasi"},{"key":"D","content":"Anekdot"}]','B',1,'Teks eksplanasi menerangkan sebab-akibat fenomena.',-0.1,1.0,NULL),
('40000000-0000-0000-0000-000000000010','30000000-0000-0000-0000-000000000001','Literasi','Kata hubung yang menyatakan pertentangan adalah ...','[{"key":"A","content":"Karena"},{"key":"B","content":"Sehingga"},{"key":"C","content":"Tetapi"},{"key":"D","content":"Kemudian"}]','C',1,'"Tetapi" menghubungkan gagasan yang bertentangan.',0.0,0.9,NULL),
-- Ujian 2: Bank Soal Aljabar milik guru (5 Matematika)
('40000000-0000-0000-0000-000000000011','30000000-0000-0000-0000-000000000002','Matematika','Jika $a=3$ dan $b=4$, nilai $2a+b$ adalah ...','[{"key":"A","content":"9"},{"key":"B","content":"10"},{"key":"C","content":"11"},{"key":"D","content":"14"}]','B',1,'$2(3)+4=6+4=10$.',-0.5,1.0,NULL),
('40000000-0000-0000-0000-000000000012','30000000-0000-0000-0000-000000000002','Matematika','Faktor dari $x^2-9$ adalah ...','[{"key":"A","content":"$(x-3)^2$"},{"key":"B","content":"$(x-3)(x+3)$"},{"key":"C","content":"$(x-9)(x+1)$"},{"key":"D","content":"$(x+9)(x-1)$"}]','B',1,'Selisih kuadrat: $x^2-9=(x-3)(x+3)$.',0.1,1.1,NULL),
('40000000-0000-0000-0000-000000000013','30000000-0000-0000-0000-000000000002','Matematika','Penyelesaian dari $x + 7 = 12$ adalah ...','[{"key":"A","content":"3"},{"key":"B","content":"5"},{"key":"C","content":"7"},{"key":"D","content":"19"}]','B',1,'Kurangi kedua ruas dengan 7: $x=5$.',-0.3,1.0,NULL),
('40000000-0000-0000-0000-000000000014','30000000-0000-0000-0000-000000000002','Matematika','Bentuk sederhana dari $3x + 2x$ adalah ...','[{"key":"A","content":"$5x$"},{"key":"B","content":"$6x$"},{"key":"C","content":"$5x^2$"},{"key":"D","content":"$6x^2$"}]','A',1,'Jumlahkan koefisien: $3x+2x=(3+2)x=5x$.',0.2,1.0,NULL),
('40000000-0000-0000-0000-000000000015','30000000-0000-0000-0000-000000000002','Matematika','Jika $y=2x-1$ dan $x=3$, maka $y$ adalah ...','[{"key":"A","content":"3"},{"key":"B","content":"4"},{"key":"C","content":"5"},{"key":"D","content":"6"}]','C',1,'Substitusi $x=3$: $y=2(3)-1=5$.',0.4,1.2,NULL),
-- Ujian 3: Tryout Gratis (2 soal)
('40000000-0000-0000-0000-000000000021','30000000-0000-0000-0000-000000000003','Matematika','Hasil dari $25 \times 4$ adalah ...','[{"key":"A","content":"80"},{"key":"B","content":"90"},{"key":"C","content":"100"},{"key":"D","content":"120"}]','C',1,'$25 \times 4 = 100$.',-0.8,1.0,NULL),
('40000000-0000-0000-0000-000000000022','30000000-0000-0000-0000-000000000003','Literasi','Sinonim dari kata "cerdas" adalah ...','[{"key":"A","content":"bodoh"},{"key":"B","content":"pintar"},{"key":"C","content":"malas"},{"key":"D","content":"lemah"}]','B',1,'Pintar memiliki makna yang sama dengan cerdas.',0.0,1.0,NULL),
-- Ujian 4: Tryout CBT Matematika milik guru (3 soal)
('40000000-0000-0000-0000-000000000031','30000000-0000-0000-0000-000000000004','Matematika','Jika $f(x)=2x^2+1$, nilai $f(3)$ adalah ...','[{"key":"A","content":"17"},{"key":"B","content":"19"},{"key":"C","content":"21"},{"key":"D","content":"37"}]','B',1,'$f(3)=2(9)+1=19$.',0.1,1.0,NULL),
('40000000-0000-0000-0000-000000000032','30000000-0000-0000-0000-000000000004','Matematika','Gradien garis yang melalui $(0,0)$ dan $(2,6)$ adalah ...','[{"key":"A","content":"2"},{"key":"B","content":"3"},{"key":"C","content":"4"},{"key":"D","content":"6"}]','B',1,'Gradien $m=\frac{6-0}{2-0}=3$.',0.3,1.1,NULL),
('40000000-0000-0000-0000-000000000033','30000000-0000-0000-0000-000000000004','Matematika','Diketahui deret $2,6,10,14,\dots$ Suku ke-20 adalah ...','[{"key":"A","content":"74"},{"key":"B","content":"76"},{"key":"C","content":"78"},{"key":"D","content":"80"}]','C',1,'Barisan aritmetika $a=2$, $b=4$: $U_{20}=2+19\cdot4=78$.',0.6,1.2,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name,
content_text=EXCLUDED.content_text, options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer,
score_weight=EXCLUDED.score_weight, explanation_text=EXCLUDED.explanation_text,
difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

-- ---------- BANK SOAL SD, 3 BUAH ----------
INSERT INTO packages (id, title, description, price, validity_days, kode, jenjang, exam_type, kategori_id, kelas_id) VALUES
('20000000-0000-0000-0000-000000000011','Bank Soal Matematika SD','Latihan dasar aritmetika dan bilangan untuk jenjang SD dengan pembahasan.',10000,14,'TKA-SD1','SD','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 5 SD')),
('20000000-0000-0000-0000-000000000012','Bank Soal Bahasa Indonesia SD','Latihan kosakata, ejaan, dan pemahaman bacaan untuk jenjang SD.',10000,14,'TKA-SD2','SD','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 4 SD')),
('20000000-0000-0000-0000-000000000013','Bank Soal IPA SD','Latihan pengenalan alam sekitar, makhluk hidup, dan gejala alam untuk SD.',10000,14,'TKA-SD3','SD','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 4 SD'))
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price,
validity_days=EXCLUDED.validity_days, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang,
exam_type=EXCLUDED.exam_type, kategori_id=EXCLUDED.kategori_id, kelas_id=EXCLUDED.kelas_id;

-- ---------- BANK SOAL SMP, 3 BUAH ----------
INSERT INTO packages (id, title, description, price, validity_days, kode, jenjang, exam_type, kategori_id, kelas_id) VALUES
('20000000-0000-0000-0000-000000000021','Bank Soal Matematika SMP','Latihan aljabar, persamaan, dan geometri untuk jenjang SMP.',12000,14,'TKA-SMP1','SMP','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 8 SMP')),
('20000000-0000-0000-0000-000000000022','Bank Soal Bahasa Inggris SMP','Latihan grammar, vocabulary, dan reading comprehension untuk SMP.',12000,14,'TKA-SMP2','SMP','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 7 SMP')),
('20000000-0000-0000-0000-000000000023','Bank Soal IPA SMP','Latihan IPA terpadu: fisika, kimia, dan biologi untuk jenjang SMP.',12000,14,'TKA-SMP3','SMP','sell',(SELECT id FROM kategori WHERE nama='Ulangan Harian'),(SELECT id FROM kelas WHERE nama='Kelas 8 SMP'))
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price,
validity_days=EXCLUDED.validity_days, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang,
exam_type=EXCLUDED.exam_type, kategori_id=EXCLUDED.kategori_id, kelas_id=EXCLUDED.kelas_id;

-- ---------- UJIAN BANK SOAL SD, 3 BUAH ----------
INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method) VALUES
('30000000-0000-0000-0000-000000000011','20000000-0000-0000-0000-000000000011','Latihan Matematika Dasar',15,4,60,'standard'),
('30000000-0000-0000-0000-000000000012','20000000-0000-0000-0000-000000000012','Latihan Bahasa Indonesia Dasar',15,4,60,'standard'),
('30000000-0000-0000-0000-000000000013','20000000-0000-0000-0000-000000000013','Latihan IPA Dasar',15,4,60,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title,
duration_minutes=EXCLUDED.duration_minutes, total_questions=EXCLUDED.total_questions,
passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

-- ---------- SOAL BANK SOAL SD, 12 BUAH ----------
INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
('40000000-0000-0000-0000-000000000101','30000000-0000-0000-0000-000000000011','Matematika','Hasil dari $7 + 8$ adalah ...','[{"key":"A","content":"13"},{"key":"B","content":"15"},{"key":"C","content":"16"},{"key":"D","content":"17"}]','B',1,'$7+8=15$.',-1,1,NULL),
('40000000-0000-0000-0000-000000000102','30000000-0000-0000-0000-000000000011','Matematika','Hasil dari $5 \times 6$ adalah ...','[{"key":"A","content":"25"},{"key":"B","content":"28"},{"key":"C","content":"30"},{"key":"D","content":"36"}]','C',1,'$5 \times 6 = 30$.',-0.8,1,NULL),
('40000000-0000-0000-0000-000000000103','30000000-0000-0000-0000-000000000011','Matematika','Hasil dari $100 - 37$ adalah ...','[{"key":"A","content":"57"},{"key":"B","content":"63"},{"key":"C","content":"67"},{"key":"D","content":"73"}]','B',1,'$100-37=63$.',-0.5,1,NULL),
('40000000-0000-0000-0000-000000000104','30000000-0000-0000-0000-000000000011','Matematika','Setengah dari 10 adalah ...','[{"key":"A","content":"4"},{"key":"B","content":"5"},{"key":"C","content":"6"},{"key":"D","content":"8"}]','B',1,'$\frac{10}{2}=5$.',-0.3,1,NULL),
('40000000-0000-0000-0000-000000000111','30000000-0000-0000-0000-000000000012','Bahasa Indonesia','Lawan kata (antonim) dari "besar" adalah ...','[{"key":"A","content":"Kecil"},{"key":"B","content":"Luas"},{"key":"C","content":"Tinggi"},{"key":"D","content":"Panjang"}]','A',1,'Antonim "besar" adalah "kecil".',-0.7,1,NULL),
('40000000-0000-0000-0000-000000000112','30000000-0000-0000-0000-000000000012','Bahasa Indonesia','Kalimat "Ibu pergi ke pasar" memiliki subjek ...','[{"key":"A","content":"pergi"},{"key":"B","content":"Ibu"},{"key":"C","content":"ke pasar"},{"key":"D","content":"pasar"}]','B',1,'Subjek kalimat adalah "Ibu".',-0.4,1,NULL),
('40000000-0000-0000-0000-000000000113','30000000-0000-0000-0000-000000000012','Bahasa Indonesia','Huruf kapital dipakai pada ...','[{"key":"A","content":"Semua kata dalam kalimat"},{"key":"B","content":"Awal kalimat dan nama orang"},{"key":"C","content":"Akhir kalimat"},{"key":"D","content":"Setiap kata kerja"}]','B',1,'Huruf kapital digunakan di awal kalimat dan nama orang.',0,1,NULL),
('40000000-0000-0000-0000-000000000114','30000000-0000-0000-0000-000000000012','Bahasa Indonesia','Sinonim dari kata "pandai" adalah ...','[{"key":"A","content":"Cepat"},{"key":"B","content":"Pintar"},{"key":"C","content":"Rajin"},{"key":"D","content":"Teliti"}]','B',1,'"Pandai" berarti "pintar".',0.1,1,NULL),
('40000000-0000-0000-0000-000000000121','30000000-0000-0000-0000-000000000013','IPA','Bagian tumbuhan tempat terjadinya fotosintesis adalah ...','[{"key":"A","content":"Akar"},{"key":"B","content":"Batang"},{"key":"C","content":"Daun"},{"key":"D","content":"Bunga"}]','C',1,'Fotosintesis terjadi di daun yang mengandung klorofil.',-0.6,1,NULL),
('40000000-0000-0000-0000-000000000122','30000000-0000-0000-0000-000000000013','IPA','Alat pernapasan utama manusia adalah ...','[{"key":"A","content":"Jantung"},{"key":"B","content":"Paru-paru"},{"key":"C","content":"Hati"},{"key":"D","content":"Ginjal"}]','B',1,'Manusia bernapas dengan paru-paru.',-0.5,1,NULL),
('40000000-0000-0000-0000-000000000123','30000000-0000-0000-0000-000000000013','IPA','Air yang dipanaskan akan berubah menjadi ...','[{"key":"A","content":"Es"},{"key":"B","content":"Uap air"},{"key":"C","content":"Salju"},{"key":"D","content":"Tanah"}]','B',1,'Air yang dipanaskan menguap menjadi uap air.',0,1,NULL),
('40000000-0000-0000-0000-000000000124','30000000-0000-0000-0000-000000000013','IPA','Hewan berikut yang termasuk mamalia adalah ...','[{"key":"A","content":"Ayam"},{"key":"B","content":"Kucing"},{"key":"C","content":"Buaya"},{"key":"D","content":"Katak"}]','B',1,'Kucing menyusui anaknya, termasuk mamalia.',0.2,1,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name,
content_text=EXCLUDED.content_text, options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer,
score_weight=EXCLUDED.score_weight, explanation_text=EXCLUDED.explanation_text,
difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

-- ---------- UJIAN BANK SOAL SMP, 3 BUAH ----------
INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method) VALUES
('30000000-0000-0000-0000-000000000021','20000000-0000-0000-0000-000000000021','Latihan Aljabar SMP',15,4,60,'standard'),
('30000000-0000-0000-0000-000000000022','20000000-0000-0000-0000-000000000022','Latihan Reading Comprehension',15,4,60,'standard'),
('30000000-0000-0000-0000-000000000023','20000000-0000-0000-0000-000000000023','Latihan IPA Terpadu',15,4,60,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title,
duration_minutes=EXCLUDED.duration_minutes, total_questions=EXCLUDED.total_questions,
passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

-- ---------- SOAL BANK SOAL SMP, 12 BUAH ----------
INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
('40000000-0000-0000-0000-000000000201','30000000-0000-0000-0000-000000000021','Matematika','Penyelesaian dari $2x = 8$ adalah ...','[{"key":"A","content":"2"},{"key":"B","content":"4"},{"key":"C","content":"6"},{"key":"D","content":"16"}]','B',1,'Kedua ruas dibagi 2: $x=4$.',-0.6,1,NULL),
('40000000-0000-0000-0000-000000000202','30000000-0000-0000-0000-000000000021','Matematika','Jika $a=5$, nilai dari $3a-2$ adalah ...','[{"key":"A","content":"10"},{"key":"B","content":"12"},{"key":"C","content":"13"},{"key":"D","content":"15"}]','C',1,'$3(5)-2=15-2=13$.',-0.3,1,NULL),
('40000000-0000-0000-0000-000000000203','30000000-0000-0000-0000-000000000021','Matematika','Akar positif dari $x^2=36$ adalah ...','[{"key":"A","content":"4"},{"key":"B","content":"5"},{"key":"C","content":"6"},{"key":"D","content":"7"}]','C',1,'$x=6$ karena $6^2=36$.',0,1,NULL),
('40000000-0000-0000-0000-000000000204','30000000-0000-0000-0000-000000000021','Matematika','Hasil dari $2(3+4)$ adalah ...','[{"key":"A","content":"10"},{"key":"B","content":"12"},{"key":"C","content":"13"},{"key":"D","content":"14"}]','D',1,'$2(3+4)=2 \times 7=14$.',0.2,1,NULL),
('40000000-0000-0000-0000-000000000211','30000000-0000-0000-0000-000000000022','Bahasa Inggris','She ___ to school every day.','[{"key":"A","content":"go"},{"key":"B","content":"going"},{"key":"C","content":"goes"},{"key":"D","content":"gone"}]','C',1,'She (third person) + verb-s: "goes".',-0.5,1,NULL),
('40000000-0000-0000-0000-000000000212','30000000-0000-0000-0000-000000000022','Bahasa Inggris','The opposite of "happy" is ...','[{"key":"A","content":"glad"},{"key":"B","content":"cheerful"},{"key":"C","content":"joyful"},{"key":"D","content":"sad"}]','D',1,'"Sad" is the opposite of "happy".',-0.2,1,NULL),
('40000000-0000-0000-0000-000000000213','30000000-0000-0000-0000-000000000022','Bahasa Inggris','In "The students are studying", the subject is ...','[{"key":"A","content":"The students"},{"key":"B","content":"are"},{"key":"C","content":"studying"},{"key":"D","content":"The"}]','A',1,'The subject is "The students".',0.1,1,NULL),
('40000000-0000-0000-0000-000000000214','30000000-0000-0000-0000-000000000022','Bahasa Inggris','Tomorrow they ___ visit the museum.','[{"key":"A","content":"visit"},{"key":"B","content":"will go"},{"key":"C","content":"visited"},{"key":"D","content":"visits"}]','B',1,'Future tense dengan "will": "will go".',0.3,1,NULL),
('40000000-0000-0000-0000-000000000221','30000000-0000-0000-0000-000000000023','IPA','Satuan gaya dalam SI adalah ...','[{"key":"A","content":"Joule"},{"key":"B","content":"Watt"},{"key":"C","content":"Pascal"},{"key":"D","content":"Newton"}]','D',1,'Gaya diukur dengan satuan Newton.',-0.4,1,NULL),
('40000000-0000-0000-0000-000000000222','30000000-0000-0000-0000-000000000023','IPA','Fotosintesis menghasilkan ...','[{"key":"A","content":"Oksigen"},{"key":"B","content":"Karbon dioksida"},{"key":"C","content":"Nitrogen"},{"key":"D","content":"Hidrogen"}]','A',1,'Tumbuhan melepaskan oksigen saat fotosintesis.',0,1,NULL),
('40000000-0000-0000-0000-000000000223','30000000-0000-0000-0000-000000000023','IPA','Organel sel tempat pembentukan energi adalah ...','[{"key":"A","content":"Nukleus"},{"key":"B","content":"Ribosom"},{"key":"C","content":"Mitokondria"},{"key":"D","content":"Membran sel"}]','C',1,'Mitokondria menghasilkan energi (ATP).',0.2,1,NULL),
('40000000-0000-0000-0000-000000000224','30000000-0000-0000-0000-000000000023','IPA','Contoh benda konduktor panas adalah ...','[{"key":"A","content":"Kayu"},{"key":"B","content":"Besi"},{"key":"C","content":"Plastik"},{"key":"D","content":"Styrofoam"}]','B',1,'Besi menghantarkan panas dengan baik (konduktor).',0.4,1,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name,
content_text=EXCLUDED.content_text, options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer,
score_weight=EXCLUDED.score_weight, explanation_text=EXCLUDED.explanation_text,
difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

-- ---------- KEPEMILIKAN PAKET ----------
DELETE FROM user_packages WHERE user_id IN (
'10000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103')
AND package_id IN ('20000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000003','20000000-0000-0000-0000-000000000004');

-- Tryout Eceran (paket 1) dimiliki semua siswa agar hasil ranking bermakna.
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000001',NOW()+INTERVAL '7 days','active' FROM users
WHERE id IN ('10000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103');

-- Bank Soal Guru (paket 2) dimiliki pembeli berbayar; Tryout Gratis (paket 3) diambil Raka.
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000002',NOW()+INTERVAL '14 days','active' FROM users
WHERE id IN ('10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103');
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000003',NOW()+INTERVAL '7 days','active' FROM users
WHERE id = '10000000-0000-0000-0000-000000000101';

-- Tryout CBT Bu Rina (paket 4) adalah gratis; pendaftar demo dapat langsung mengerjakannya.
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000004',NOW()+INTERVAL '14 days','active' FROM users
WHERE id IN ('10000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103');

-- ---------- HASIL UJIAN (RANKING GLOBAL), 3 BUAH ----------
INSERT INTO user_exams (id, user_id, exam_id, status, started_at, finished_at, total_score) VALUES
('50000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000101','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '6 days 31 minutes',NOW()-INTERVAL '6 days',82.50),
('50000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000102','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '5 days 29 minutes',NOW()-INTERVAL '5 days',76.00),
('50000000-0000-0000-0000-000000000103','10000000-0000-0000-0000-000000000103','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '4 days 28 minutes',NOW()-INTERVAL '4 days',92.50)
ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status, started_at=EXCLUDED.started_at,
finished_at=EXCLUDED.finished_at, total_score=EXCLUDED.total_score;

-- ---------- TRANSAKSI, 3 BUAH (2 paket berbayar terbeli) ----------
INSERT INTO transactions (id, user_id, package_id, invoice_number, amount, payment_status, payment_method, paid_at)
VALUES
('60000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000103','20000000-0000-0000-0000-000000000001','SEED-2026-0201',25000,'paid','virtual_account',NOW()-INTERVAL '3 days'),
('60000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000102','20000000-0000-0000-0000-000000000002','SEED-2026-0202',15000,'paid','virtual_account',NOW()-INTERVAL '2 days'),
('60000000-0000-0000-0000-000000000003','10000000-0000-0000-0000-000000000101','20000000-0000-0000-0000-000000000002','SEED-2026-0203',15000,'paid','virtual_account',NOW()-INTERVAL '1 day')
ON CONFLICT (id) DO UPDATE SET user_id=EXCLUDED.user_id, package_id=EXCLUDED.package_id,
invoice_number=EXCLUDED.invoice_number, amount=EXCLUDED.amount, payment_status=EXCLUDED.payment_status,
payment_method=EXCLUDED.payment_method, paid_at=EXCLUDED.paid_at;

-- ---------- KOMISI GURU, 3 BUAH (biaya unggah + 2 bonus penjualan) ----------
INSERT INTO teacher_commissions (id, teacher_id, package_id, transaction_id, kind, base_amount, rate_percent, amount, status, available_at) VALUES
('80000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000201','20000000-0000-0000-0000-000000000002',NULL,'upload_fee',30000,100,30000,'available',NOW()-INTERVAL '5 days'),
('80000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000201','20000000-0000-0000-0000-000000000002','60000000-0000-0000-0000-000000000002','sales_bonus',15000,10,1500,'available',NOW()-INTERVAL '2 days'),
('80000000-0000-0000-0000-000000000003','10000000-0000-0000-0000-000000000201','20000000-0000-0000-0000-000000000002','60000000-0000-0000-0000-000000000003','sales_bonus',15000,10,1500,'available',NOW()-INTERVAL '1 day')
ON CONFLICT (id) DO UPDATE SET teacher_id=EXCLUDED.teacher_id, package_id=EXCLUDED.package_id,
transaction_id=EXCLUDED.transaction_id, kind=EXCLUDED.kind, base_amount=EXCLUDED.base_amount,
rate_percent=EXCLUDED.rate_percent, amount=EXCLUDED.amount, status=EXCLUDED.status,
available_at=EXCLUDED.available_at;

-- Rekening payout guru agar menu Pencairan bisa dicoba.
INSERT INTO teacher_payout_accounts (teacher_id, method, provider, account_number, account_holder_name, phone) VALUES
('10000000-0000-0000-0000-000000000201','bank_transfer','BCA','1234567890','Rina Kusuma','081234567890')
ON CONFLICT (teacher_id) DO UPDATE SET method=EXCLUDED.method, provider=EXCLUDED.provider,
account_number=EXCLUDED.account_number, account_holder_name=EXCLUDED.account_holder_name, phone=EXCLUDED.phone;

-- Satu pengajuan pencairan menunggu tinjauan finance.
INSERT INTO teacher_payout_requests (id, teacher_id, amount, status, payout_method, provider, account_number, account_holder_name, submitted_at) VALUES
('90000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000201',30000,'submitted','bank_transfer','BCA','1234567890','Rina Kusuma',NOW()-INTERVAL '6 hours')
ON CONFLICT (id) DO UPDATE SET teacher_id=EXCLUDED.teacher_id, amount=EXCLUDED.amount, status=EXCLUDED.status,
payout_method=EXCLUDED.payout_method, provider=EXCLUDED.provider, account_number=EXCLUDED.account_number,
account_holder_name=EXCLUDED.account_holder_name, submitted_at=EXCLUDED.submitted_at;

-- Profil finance guru agar panel Kelola User finance terisi.
INSERT INTO user_finance_profiles (user_id, account_status, notes) VALUES
('10000000-0000-0000-0000-000000000201','active','Guru aktif menerima komisi penjualan.')
ON CONFLICT (user_id) DO UPDATE SET account_status=EXCLUDED.account_status, notes=EXCLUDED.notes, updated_at=NOW();

-- ---------- TESTIMONI, 3 BUAH (2 disetujui + 1 menunggu) ----------
INSERT INTO testimonials (id, user_id, quote, status) VALUES
('70000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000101','Latihan soalnya mirip banget sama ujian aslinya, jadi lebih pede pas hari-H.','approved'),
('70000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000102','Rankingnya bikin aku semangat belajar lagi buat naik peringkat setiap minggu.','approved'),
('70000000-0000-0000-0000-000000000003','10000000-0000-0000-0000-000000000103','Analisis hasil per mata pelajaran sangat membantu memahami kelemahan saya.','pending')
ON CONFLICT (id) DO NOTHING;

-- ---------- AUDIT LOG DEMO ----------
INSERT INTO audit_logs (actor_id, actor_email, action, entity_type, entity_id, detail) VALUES
(NULL,'system','seed','site_settings','', '{"note":"Dataset demo diinisialisasi ulang"}');

COMMIT;