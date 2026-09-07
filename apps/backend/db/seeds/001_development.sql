BEGIN;

-- Semua akun demo lokal menggunakan password: 12345678
WITH password AS MATERIALIZED (SELECT crypt('12345678', gen_salt('bf', 10)) AS hash)
INSERT INTO users (id, email, password_hash, role, school_level, name)
SELECT data.id::uuid, data.email, password.hash, data.role, data.school_level, data.name
FROM password CROSS JOIN (VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin@tka.local', 'admin', 'SMA', 'Admin Platform'),
    ('10000000-0000-0000-0000-000000000101', 'siswa.sd1@tka.local', 'student', 'SD', 'Raka Pratama'),
    ('10000000-0000-0000-0000-000000000102', 'siswa.sd2@tka.local', 'student', 'SD', 'Nadia Safitri'),
    ('10000000-0000-0000-0000-000000000103', 'siswa.smp1@tka.local', 'student', 'SMP', 'Bimo Ardiansyah'),
    ('10000000-0000-0000-0000-000000000104', 'siswa.smp2@tka.local', 'student', 'SMP', 'Citra Dewi'),
    ('10000000-0000-0000-0000-000000000105', 'siswa.sma1@tka.local', 'student', 'SMA', 'Dewa Mahendra'),
    ('10000000-0000-0000-0000-000000000201', 'guru.sma1@tka.local', 'teacher', 'SMA', 'Rina Kusuma')
) AS data(id, email, role, school_level, name)
ON CONFLICT (id) DO UPDATE SET email=EXCLUDED.email, password_hash=EXCLUDED.password_hash,
role=EXCLUDED.role, school_level=EXCLUDED.school_level, name=EXCLUDED.name, updated_at=NOW();

INSERT INTO packages (id, title, description, price, validity_days, kode, jenjang) VALUES
('20000000-0000-0000-0000-000000000001','Tryout TKA Eceran','Satu simulasi lengkap dengan pembahasan.',25000,7,'TKA-0001','SMA'),
('20000000-0000-0000-0000-000000000002','Bundling TKA Intensif','Paket latihan intensif, analitik, dan video pembahasan.',99000,30,'TKA-0002','SMA')
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price, validity_days=EXCLUDED.validity_days, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang;

INSERT INTO packages (id, title, description, price, validity_days, publisher_id, kode, jenjang) VALUES
('20000000-0000-0000-0000-000000000011','Bank Soal Matematika Bu Rina','Kumpulan soal latihan Matematika karya guru bersertifikat.',15000,14,'10000000-0000-0000-0000-000000000201','TKA-0003','SMA')
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price, validity_days=EXCLUDED.validity_days, publisher_id=EXCLUDED.publisher_id, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang;

INSERT INTO packages (id, title, description, price, validity_days, kode, jenjang) VALUES
('20000000-0000-0000-0000-000000000021','Tryout Gratis TKA','Simulasi tryout gratis untuk mencoba pengalaman ujian penuh TKA Juara.',0,7,'TKA-0008','SMA')
ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, price=EXCLUDED.price, validity_days=EXCLUDED.validity_days, kode=EXCLUDED.kode, jenjang=EXCLUDED.jenjang;

INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method)
VALUES ('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000002','Simulasi TKA Matematika dan Literasi',30,20,70,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title, duration_minutes=EXCLUDED.duration_minutes,
total_questions=EXCLUDED.total_questions, passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
('40000000-0000-0000-0000-000000000001','30000000-0000-0000-0000-000000000001','Matematika','Jika $2x + 6 = 16$, nilai $x$ adalah ...','[{"key":"A","content":"3"},{"key":"B","content":"4"},{"key":"C","content":"5"},{"key":"D","content":"6"}]','C',1,'Kurangi kedua ruas dengan 6, sehingga $2x=10$ dan $x=5$.',-1.2,1.0,NULL),
('40000000-0000-0000-0000-000000000002','30000000-0000-0000-0000-000000000001','Matematika','Nilai dari $3^2 + 4^2$ adalah ...','[{"key":"A","content":"7"},{"key":"B","content":"12"},{"key":"C","content":"25"},{"key":"D","content":"49"}]','C',1,'$3^2+4^2=9+16=25$.',-1.0,1.1,NULL),
('40000000-0000-0000-0000-000000000003','30000000-0000-0000-0000-000000000001','Matematika','Luas persegi dengan sisi 8 cm adalah ...','[{"key":"A","content":"16 cm²"},{"key":"B","content":"32 cm²"},{"key":"C","content":"64 cm²"},{"key":"D","content":"80 cm²"}]','C',1,'Luas persegi $L=s^2=8^2=64$ cm².',-0.9,1.0,NULL),
('40000000-0000-0000-0000-000000000004','30000000-0000-0000-0000-000000000001','Matematika','Perhatikan diagram segitiga: https://placehold.co/640x320/png?text=Segitiga+Siku-siku . Jika alas 6 cm dan tinggi 4 cm, luasnya ...','[{"key":"A","content":"10 cm²"},{"key":"B","content":"12 cm²"},{"key":"C","content":"20 cm²"},{"key":"D","content":"24 cm²"}]','B',1,'Gunakan $\frac{1}{2}at=\frac{1}{2}(6)(4)=12$ cm².',-0.7,1.2,NULL),
('40000000-0000-0000-0000-000000000005','30000000-0000-0000-0000-000000000001','Matematika','Hasil $\frac{3}{4}+\frac{1}{8}$ adalah ...','[{"key":"A","content":"$\\frac{1}{2}$"},{"key":"B","content":"$\\frac{5}{8}$"},{"key":"C","content":"$\\frac{7}{8}$"},{"key":"D","content":"1"}]','C',1,'Samakan penyebut: $\frac{6}{8}+\frac{1}{8}=\frac{7}{8}$.',-0.4,1.2,NULL),
('40000000-0000-0000-0000-000000000006','30000000-0000-0000-0000-000000000001','Matematika','Rata-rata dari 6, 8, 10, dan 12 adalah ...','[{"key":"A","content":"8"},{"key":"B","content":"9"},{"key":"C","content":"10"},{"key":"D","content":"11"}]','B',1,'Jumlah data 36 dibagi 4 menghasilkan 9.',-0.2,1.0,NULL),
('40000000-0000-0000-0000-000000000007','30000000-0000-0000-0000-000000000001','Matematika','Jika $f(x)=2x^2-3$, maka $f(2)$ adalah ...','[{"key":"A","content":"1"},{"key":"B","content":"5"},{"key":"C","content":"8"},{"key":"D","content":"13"}]','B',1,'Substitusi $x=2$: $2(2^2)-3=5$.',0.0,1.1,NULL),
('40000000-0000-0000-0000-000000000008','30000000-0000-0000-0000-000000000001','Matematika','Akar positif dari $x^2-49=0$ adalah ...','[{"key":"A","content":"5"},{"key":"B","content":"6"},{"key":"C","content":"7"},{"key":"D","content":"49"}]','C',1,'$x^2=49$, sehingga akar positifnya $x=7$.',0.1,1.3,NULL),
('40000000-0000-0000-0000-000000000009','30000000-0000-0000-0000-000000000001','Matematika','Barang Rp80.000 mendapat diskon 25%. Harga akhirnya ...','[{"key":"A","content":"Rp20.000"},{"key":"B","content":"Rp55.000"},{"key":"C","content":"Rp60.000"},{"key":"D","content":"Rp75.000"}]','C',1,'Diskon Rp20.000, jadi harga akhir Rp60.000.',0.2,1.0,NULL),
('40000000-0000-0000-0000-000000000010','30000000-0000-0000-0000-000000000001','Matematika','Peluang muncul angka genap pada satu lemparan dadu adalah ...','[{"key":"A","content":"$\\frac{1}{6}$"},{"key":"B","content":"$\\frac{1}{3}$"},{"key":"C","content":"$\\frac{1}{2}$"},{"key":"D","content":"$\\frac{2}{3}$"}]','C',1,'Ada 3 hasil genap dari 6, jadi $\frac{3}{6}=\frac{1}{2}$.',0.3,1.1,NULL),
('40000000-0000-0000-0000-000000000011','30000000-0000-0000-0000-000000000001','Literasi','Kalimat utama dalam paragraf umumnya memuat ...','[{"key":"A","content":"Ide pokok"},{"key":"B","content":"Contoh tambahan"},{"key":"C","content":"Nama penulis"},{"key":"D","content":"Daftar pustaka"}]','A',1,'Kalimat utama menyatakan ide pokok.',-0.8,0.9,NULL),
('40000000-0000-0000-0000-000000000012','30000000-0000-0000-0000-000000000001','Literasi','Sinonim kata “cermat” adalah ...','[{"key":"A","content":"Ceroboh"},{"key":"B","content":"Teliti"},{"key":"C","content":"Lambat"},{"key":"D","content":"Keras"}]','B',1,'Cermat dan teliti bermakna saksama.',-0.6,1.0,NULL),
('40000000-0000-0000-0000-000000000013','30000000-0000-0000-0000-000000000001','Literasi','Penulisan kata baku yang tepat adalah ...','[{"key":"A","content":"Aktifitas"},{"key":"B","content":"Resiko"},{"key":"C","content":"Analisis"},{"key":"D","content":"Ijin"}]','C',1,'Bentuk bakunya: aktivitas, risiko, analisis, dan izin.',-0.3,1.1,NULL),
('40000000-0000-0000-0000-000000000014','30000000-0000-0000-0000-000000000001','Literasi','Teks yang menjelaskan proses terjadinya fenomena disebut teks ...','[{"key":"A","content":"Narasi"},{"key":"B","content":"Eksplanasi"},{"key":"C","content":"Negosiasi"},{"key":"D","content":"Anekdot"}]','B',1,'Teks eksplanasi menerangkan sebab-akibat fenomena.',-0.1,1.0,NULL),
('40000000-0000-0000-0000-000000000015','30000000-0000-0000-0000-000000000001','Literasi','Kata hubung yang menyatakan pertentangan adalah ...','[{"key":"A","content":"Karena"},{"key":"B","content":"Sehingga"},{"key":"C","content":"Tetapi"},{"key":"D","content":"Kemudian"}]','C',1,'“Tetapi” menghubungkan gagasan yang bertentangan.',0.0,0.9,NULL),
('40000000-0000-0000-0000-000000000016','30000000-0000-0000-0000-000000000001','Literasi','Simpulan yang baik harus ...','[{"key":"A","content":"Memuat informasi baru"},{"key":"B","content":"Sesuai isi bacaan"},{"key":"C","content":"Selalu panjang"},{"key":"D","content":"Berupa pertanyaan"}]','B',1,'Simpulan merangkum bacaan tanpa informasi baru.',0.2,1.1,NULL),
('40000000-0000-0000-0000-000000000017','30000000-0000-0000-0000-000000000001','Literasi','Makna imbuhan “pe-” pada kata “pelari” adalah ...','[{"key":"A","content":"Alat"},{"key":"B","content":"Tempat"},{"key":"C","content":"Pelaku"},{"key":"D","content":"Proses"}]','C',1,'“Pelari” berarti pelaku yang berlari.',0.3,1.0,NULL),
('40000000-0000-0000-0000-000000000018','30000000-0000-0000-0000-000000000001','Literasi','Tujuan utama membaca kritis adalah ...','[{"key":"A","content":"Menghafal semua kata"},{"key":"B","content":"Menilai informasi dan argumen"},{"key":"C","content":"Membaca secepat mungkin"},{"key":"D","content":"Menyalin paragraf"}]','B',1,'Membaca kritis mengevaluasi informasi, bukti, dan argumen.',0.5,1.2,NULL),
('40000000-0000-0000-0000-000000000019','30000000-0000-0000-0000-000000000001','Literasi','Kalimat efektif adalah kalimat yang ...','[{"key":"A","content":"Bertele-tele"},{"key":"B","content":"Memiliki banyak istilah asing"},{"key":"C","content":"Jelas, hemat, dan logis"},{"key":"D","content":"Selalu dua paragraf"}]','C',1,'Kalimat efektif menyampaikan gagasan secara jelas, hemat, dan logis.',0.6,1.2,NULL),
('40000000-0000-0000-0000-000000000020','30000000-0000-0000-0000-000000000001','Literasi','Dalam kalimat “Data itu sangat akurat”, kata “akurat” berarti ...','[{"key":"A","content":"Tepat"},{"key":"B","content":"Samar"},{"key":"C","content":"Berlebihan"},{"key":"D","content":"Sementara"}]','A',1,'Akurat berarti tepat atau teliti.',0.8,1.3,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name, content_text=EXCLUDED.content_text,
options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer, score_weight=EXCLUDED.score_weight,
explanation_text=EXCLUDED.explanation_text, difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

DELETE FROM user_packages WHERE user_id IN (
'10000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103',
'10000000-0000-0000-0000-000000000104','10000000-0000-0000-0000-000000000105')
AND package_id='20000000-0000-0000-0000-000000000002';
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000002',NOW()+INTERVAL '30 days','active' FROM users
WHERE id IN ('10000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000103','10000000-0000-0000-0000-000000000104','10000000-0000-0000-0000-000000000105');

-- Hasil demo agar menu Ranking Global langsung memiliki data saat pertama dibuka.
INSERT INTO user_exams (id, user_id, exam_id, status, started_at, finished_at, total_score) VALUES
('50000000-0000-0000-0000-000000000101','10000000-0000-0000-0000-000000000101','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '6 days 31 minutes',NOW()-INTERVAL '6 days',82.50),
('50000000-0000-0000-0000-000000000102','10000000-0000-0000-0000-000000000102','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '5 days 29 minutes',NOW()-INTERVAL '5 days',76.00),
('50000000-0000-0000-0000-000000000103','10000000-0000-0000-0000-000000000103','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '4 days 28 minutes',NOW()-INTERVAL '4 days',92.50),
('50000000-0000-0000-0000-000000000104','10000000-0000-0000-0000-000000000104','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '3 days 30 minutes',NOW()-INTERVAL '3 days',88.00),
('50000000-0000-0000-0000-000000000105','10000000-0000-0000-0000-000000000105','30000000-0000-0000-0000-000000000001','submitted',NOW()-INTERVAL '2 days 27 minutes',NOW()-INTERVAL '2 days',85.00)
ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status, started_at=EXCLUDED.started_at,
finished_at=EXCLUDED.finished_at, total_score=EXCLUDED.total_score;

-- Bank soal milik guru demo (akun guru.sma1@tka.local / 12345678).
INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method)
VALUES ('30000000-0000-0000-0000-000000000011','20000000-0000-0000-0000-000000000011','Latihan Matematika Bab Aljabar',20,5,60,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title, duration_minutes=EXCLUDED.duration_minutes,
total_questions=EXCLUDED.total_questions, passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
('40000000-0000-0000-0000-000000000021','30000000-0000-0000-0000-000000000011','Matematika','Jika $a=3$ dan $b=4$, nilai $2a+b$ adalah ...','[{"key":"A","content":"9"},{"key":"B","content":"10"},{"key":"C","content":"11"},{"key":"D","content":"14"}]','B',1,'$2(3)+4=6+4=10$.',-0.5,1.0,NULL),
('40000000-0000-0000-0000-000000000022','30000000-0000-0000-0000-000000000011','Matematika','Faktor dari $x^2-9$ adalah ...','[{"key":"A","content":"$(x-3)^2$"},{"key":"B","content":"$(x-3)(x+3)$"},{"key":"C","content":"$(x-9)(x+1)$"},{"key":"D","content":"$(x+9)(x-1)$"}]','B',1,'Selisih kuadrat: $x^2-9=(x-3)(x+3)$.',0.1,1.1,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name, content_text=EXCLUDED.content_text,
options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer, score_weight=EXCLUDED.score_weight,
explanation_text=EXCLUDED.explanation_text, difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

-- Penjualan demo untuk paket guru.
INSERT INTO transactions (id, user_id, package_id, invoice_number, amount, payment_status, payment_method, paid_at)
VALUES
('60000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000104','20000000-0000-0000-0000-000000000011','SEED-2026-0141',15000,'paid','virtual_account',NOW()-INTERVAL '3 days'),
('60000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000105','20000000-0000-0000-0000-000000000011','SEED-2026-0142',15000,'paid','virtual_account',NOW()-INTERVAL '1 day')
ON CONFLICT (id) DO UPDATE SET user_id=EXCLUDED.user_id, package_id=EXCLUDED.package_id, invoice_number=EXCLUDED.invoice_number,
amount=EXCLUDED.amount, payment_status=EXCLUDED.payment_status, payment_method=EXCLUDED.payment_method, paid_at=EXCLUDED.paid_at;

DELETE FROM user_packages WHERE user_id IN ('10000000-0000-0000-0000-000000000104','10000000-0000-0000-0000-000000000105')
AND package_id='20000000-0000-0000-0000-000000000011';
INSERT INTO user_packages (user_id, package_id, expired_at, status)
SELECT id,'20000000-0000-0000-0000-000000000011',NOW()+INTERVAL '14 days','active' FROM users
WHERE id IN ('10000000-0000-0000-0000-000000000104','10000000-0000-0000-0000-000000000105');

-- Testimoni demo: dua disetujui (tampil di landing), satu menunggu persetujuan admin.
INSERT INTO testimonials (id, user_id, quote, status)
VALUES
('70000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000101','Latihan soalnya mirip banget sama ujian aslinya, jadi lebih pede pas hari-H.','approved'),
('70000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000103','Rankingnya bikin aku semangat belajar lagi buat naik peringkat setiap minggu.','approved'),
('70000000-0000-0000-0000-000000000003','10000000-0000-0000-0000-000000000105','Analisis hasil per mata pelajaran sangat membantu memahami kelemahan saya.','pending')
ON CONFLICT (id) DO UPDATE SET user_id=EXCLUDED.user_id, quote=EXCLUDED.quote, status=EXCLUDED.status;

-- Ujian demo untuk paket gratis (TKA-0008 Tryout Gratis TKA).
INSERT INTO exams (id, package_id, title, duration_minutes, total_questions, passing_score, scoring_method)
VALUES ('30000000-0000-0000-0000-000000000021','20000000-0000-0000-0000-000000000021','Tryout Percobaan Gratis',10,2,50,'standard')
ON CONFLICT (id) DO UPDATE SET package_id=EXCLUDED.package_id, title=EXCLUDED.title, duration_minutes=EXCLUDED.duration_minutes,
total_questions=EXCLUDED.total_questions, passing_score=EXCLUDED.passing_score, scoring_method=EXCLUDED.scoring_method;

INSERT INTO questions (id, exam_id, subject_name, content_text, options_json, correct_answer, score_weight, explanation_text, difficulty, discrimination, explanation_video_url) VALUES
('40000000-0000-0000-0000-000000000031','30000000-0000-0000-0000-000000000021','Matematika','Hasil dari $25 \\times 4$ adalah ...','[{"key":"A","content":"80"},{"key":"B","content":"90"},{"key":"C","content":"100"},{"key":"D","content":"120"}]','C',1,'$25 \\times 4 = 100$.',-0.8,1.0,NULL),
('40000000-0000-0000-0000-000000000032','30000000-0000-0000-0000-000000000021','Literasi','Sinonim dari kata "cerdas" adalah ...','[{"key":"A","content":"bodoh"},{"key":"B","content":"pintar"},{"key":"C","content":"malas"},{"key":"D","content":"lemah"}]','B',1,'Pintar memiliki makna yang sama dengan cerdas.',0.0,1.0,NULL)
ON CONFLICT (id) DO UPDATE SET exam_id=EXCLUDED.exam_id, subject_name=EXCLUDED.subject_name, content_text=EXCLUDED.content_text,
options_json=EXCLUDED.options_json, correct_answer=EXCLUDED.correct_answer, score_weight=EXCLUDED.score_weight,
explanation_text=EXCLUDED.explanation_text, difficulty=EXCLUDED.difficulty, discrimination=EXCLUDED.discrimination,
explanation_video_url=EXCLUDED.explanation_video_url;

COMMIT;
