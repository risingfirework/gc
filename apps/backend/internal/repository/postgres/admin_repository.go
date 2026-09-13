// Package postgres mengimplementasikan repository domain di atas PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type AdminRepository struct{ db *pgxpool.Pool }

func NewAdminRepository(db *pgxpool.Pool) *AdminRepository { return &AdminRepository{db: db} }

func scanLevels(ctx context.Context, pool *pgxpool.Pool, out *[]string) error {
	rows, err := pool.Query(ctx, `SELECT nama FROM jenjang ORDER BY id`)
	if err != nil {
		return fmt.Errorf("jenjang: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		*out = append(*out, name)
	}
	return rows.Err()
}

func masterTable(category domain.MasterCategory) (string, error) {
	switch category {
	case domain.MasterMapel:
		return "mapel", nil
	case domain.MasterJenjang:
		return "jenjang", nil
	case domain.MasterTahunAjaran:
		return "tahun_ajaran", nil
	case domain.MasterKategori:
		return "kategori", nil
	case domain.MasterKelas:
		return "kelas", nil
	default:
		return "", fmt.Errorf("%w: master category tidak dikenal", domain.ErrInvalidInput)
	}
}

func scanMasterList(ctx context.Context, pool *pgxpool.Pool, category domain.MasterCategory, out *[]domain.MasterItem) error {
	table, err := masterTable(category)
	if err != nil {
		return err
	}
	rows, err := pool.Query(ctx, `SELECT id::text, nama FROM `+table+` ORDER BY nama, id`)
	if err != nil {
		return fmt.Errorf("master %s: %w", category, err)
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.MasterItem
		if err := rows.Scan(&item.ID, &item.Nama); err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return rows.Err()
}

func (r *AdminRepository) ListMaster(ctx context.Context, category domain.MasterCategory) ([]domain.MasterItem, error) {
	items := []domain.MasterItem{}
	if err := scanMasterList(ctx, r.db, category, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AdminRepository) CreateMaster(ctx context.Context, category domain.MasterCategory, nama string) (*domain.MasterItem, error) {
	table, err := masterTable(category)
	if err != nil {
		return nil, err
	}
	var item domain.MasterItem
	err = r.db.QueryRow(ctx, `INSERT INTO `+table+`(nama) VALUES($1) RETURNING id::text, nama`, nama).Scan(&item.ID, &item.Nama)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicateName
		}
		return nil, adminMutationError(err)
	}
	return &item, nil
}

func (r *AdminRepository) UpdateMaster(ctx context.Context, category domain.MasterCategory, id, nama string) (*domain.MasterItem, error) {
	table, err := masterTable(category)
	if err != nil {
		return nil, err
	}
	var item domain.MasterItem
	err = r.db.QueryRow(ctx, `UPDATE `+table+` SET nama=$2 WHERE id=$1 RETURNING id::text, nama`, id, nama).Scan(&item.ID, &item.Nama)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrMasterNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicateName
		}
		return nil, adminMutationError(err)
	}
	return &item, nil
}

func (r *AdminRepository) DeleteMaster(ctx context.Context, category domain.MasterCategory, id string) error {
	table, err := masterTable(category)
	if err != nil {
		return err
	}
	return r.deleteByID(ctx, `DELETE FROM `+table+` WHERE id=$1`, id, domain.ErrMasterNotFound)
}

func (r *AdminRepository) GetDashboard(ctx context.Context) (*domain.AdminDashboard, error) {
	result := &domain.AdminDashboard{Levels: []string{}, Jenjangs: []domain.MasterItem{}, Mapels: []domain.MasterItem{}, AcademicYears: []domain.MasterItem{}, Kategoris: []domain.MasterItem{}, Kelas: []domain.MasterItem{}, Users: []domain.UserResponse{}, Packages: []domain.AdminPackage{}, Exams: []domain.AdminExam{}, Transactions: []domain.AdminTransaction{}, Questions: []domain.AdminQuestion{}}
	const counts = `SELECT COUNT(*), COUNT(*) FILTER (WHERE role='student'), (SELECT COUNT(*) FROM packages), (SELECT COUNT(*) FROM exams), (SELECT COUNT(*) FROM transactions), (SELECT COUNT(*) FROM transactions WHERE payment_status='paid') FROM users`
	if err := r.db.QueryRow(ctx, counts).Scan(&result.Overview.TotalUsers, &result.Overview.TotalStudents, &result.Overview.TotalPackages, &result.Overview.TotalExams, &result.Overview.TotalTransactions, &result.Overview.PaidTransactions); err != nil {
		return nil, fmt.Errorf("admin overview: %w", err)
	}
	if err := scanLevels(ctx, r.db, &result.Levels); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterJenjang, &result.Jenjangs); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterMapel, &result.Mapels); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterTahunAjaran, &result.AcademicYears); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterKategori, &result.Kategoris); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterKelas, &result.Kelas); err != nil {
		return nil, err
	}
	users, err := r.db.Query(ctx, `SELECT id,email,COALESCE(name,''),COALESCE(to_char(birth_date,'YYYY-MM-DD'),''),COALESCE(phone,''),role,school_level,created_at,updated_at FROM users ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("admin users: %w", err)
	}
	for users.Next() {
		var item domain.UserResponse
		if err := users.Scan(&item.ID, &item.Email, &item.Name, &item.BirthDate, &item.Phone, &item.Role, &item.SchoolLevel, &item.CreatedAt, &item.UpdatedAt); err != nil {
			users.Close()
			return nil, err
		}
		result.Users = append(result.Users, item)
	}
	if err := users.Err(); err != nil {
		users.Close()
		return nil, err
	}
	users.Close()
	packages, err := r.db.Query(ctx, `SELECT `+adminPackageSelectColumns+` FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id ORDER BY p.created_at DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("admin packages: %w", err)
	}
	for packages.Next() {
		var item domain.AdminPackage
		if err := packages.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.ExamType, &item.CBTToken, &item.StartDate, &item.EndDate, &item.KategoriID, &item.KategoriName, &item.KelasID, &item.KelasName, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.SalesCount, &item.ViewCount); err != nil {
			packages.Close()
			return nil, fmt.Errorf("scan admin package: %w", err)
		}
		result.Packages = append(result.Packages, item)
	}
	if err := packages.Err(); err != nil {
		packages.Close()
		return nil, err
	}
	packages.Close()
	exams, err := r.db.Query(ctx, `SELECT e.id,e.package_id,e.title,p.title,COALESCE(e.mapel_id::text,''),COALESCE(e.tahun_ajaran_id::text,''),e.duration_minutes,e.total_questions,e.passing_score,e.status,e.publish_pembahasan,e.created_at FROM exams e JOIN packages p ON p.id=e.package_id ORDER BY e.created_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("admin exams: %w", err)
	}
	for exams.Next() {
		var item domain.AdminExam
		if err := exams.Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.PublishPembahasan, &item.CreatedAt); err != nil {
			exams.Close()
			return nil, err
		}
		result.Exams = append(result.Exams, item)
	}
	if err := exams.Err(); err != nil {
		exams.Close()
		return nil, err
	}
	exams.Close()
	transactions, err := r.db.Query(ctx, `SELECT t.id,t.invoice_number,u.email,COALESCE(p.title,''),
		t.amount,t.payment_status,t.created_at
		FROM transactions t LEFT JOIN packages p ON p.id=t.package_id JOIN users u ON u.id=t.user_id ORDER BY t.created_at DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("admin transactions: %w", err)
	}
	defer transactions.Close()
	for transactions.Next() {
		var item domain.AdminTransaction
		if err := transactions.Scan(&item.ID, &item.InvoiceNumber, &item.UserEmail, &item.PackageTitle, &item.Amount, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		result.Transactions = append(result.Transactions, item)
	}
	if err := transactions.Err(); err != nil {
		return nil, err
	}
	questions, err := r.db.Query(ctx, `SELECT q.id,q.exam_id,e.title,q.subject_name,q.content_text,q.question_type,q.presentation_type,COALESCE(q.group_code,''),q.stimulus_text,q.question_image_url,q.stimulus_image_url,q.category_labels_json,q.options_json,q.correct_answer,q.score_weight,q.explanation_text,q.status FROM questions q JOIN exams e ON e.id=q.exam_id ORDER BY q.id DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("admin questions: %w", err)
	}
	defer questions.Close()
	for questions.Next() {
		var item domain.AdminQuestion
		var rawOptions []byte
		var rawCategoryLabels []byte
		if err := questions.Scan(&item.ID, &item.ExamID, &item.ExamTitle, &item.SubjectName, &item.ContentText, &item.QuestionType, &item.PresentationType, &item.GroupCode, &item.StimulusText, &item.QuestionImageURL, &item.StimulusImageURL, &rawCategoryLabels, &rawOptions, &item.CorrectAnswer, &item.ScoreWeight, &item.Explanation, &item.Status); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawOptions, &item.Options); err != nil {
			return nil, fmt.Errorf("decode options for question %s: %w", item.ID, err)
		}
		if err := json.Unmarshal(rawCategoryLabels, &item.CategoryLabels); err != nil {
			return nil, fmt.Errorf("decode category labels for question %s: %w", item.ID, err)
		}
		result.Questions = append(result.Questions, item)
	}
	return result, questions.Err()
}

func (r *AdminRepository) CreateUser(ctx context.Context, user domain.User) (*domain.UserResponse, error) {
	const query = `INSERT INTO users(email,password_hash,role,school_level) VALUES($1,$2,$3,$4) RETURNING id,email,COALESCE(name,''),COALESCE(to_char(birth_date,'YYYY-MM-DD'),''),COALESCE(phone,''),role,school_level,created_at,updated_at`
	var result domain.UserResponse
	err := r.db.QueryRow(ctx, query, user.Email, user.PasswordHash, user.Role, user.SchoolLevel).Scan(&result.ID, &result.Email, &result.Name, &result.BirthDate, &result.Phone, &result.Role, &result.SchoolLevel, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &result, nil
}

func (r *AdminRepository) UpdateUser(ctx context.Context, userID, role, schoolLevel string) (*domain.UserResponse, error) {
	const query = `UPDATE users SET role=$2, school_level=$3, updated_at=NOW() WHERE id=$1 RETURNING id,email,COALESCE(name,''),COALESCE(to_char(birth_date,'YYYY-MM-DD'),''),COALESCE(phone,''),role,school_level,created_at,updated_at`
	var user domain.UserResponse
	err := r.db.QueryRow(ctx, query, userID, role, schoolLevel).Scan(&user.ID, &user.Email, &user.Name, &user.BirthDate, &user.Phone, &user.Role, &user.SchoolLevel, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin update user: %w", err)
	}
	return &user, nil
}

func (r *AdminRepository) DeleteUser(ctx context.Context, userID string) error {
	return r.deleteByID(ctx, `DELETE FROM users WHERE id=$1`, userID, domain.ErrUserNotFound)
}

// EnsureAffiliate menjadikan user sebagai affiliate bila belum ada, lalu
// mengembalikan kode rujukannya (yang sudah ada bila dibuat sebelumnya).
func (r *AdminRepository) EnsureAffiliate(ctx context.Context, userID, referralCode string) (string, error) {
	const query = `INSERT INTO affiliates(id,referral_code) VALUES($1,$2)
		ON CONFLICT (id) DO NOTHING
		RETURNING referral_code`
	var code string
	err := r.db.QueryRow(ctx, query, userID, referralCode).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := r.db.QueryRow(ctx, `SELECT referral_code FROM affiliates WHERE id=$1`, userID).Scan(&code); err != nil {
			return "", fmt.Errorf("read affiliate code: %w", err)
		}
		return code, nil
	}
	if err != nil {
		return "", fmt.Errorf("ensure affiliate: %w", err)
	}
	return code, nil
}

func (r *AdminRepository) CreatePackage(ctx context.Context, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	const query = `INSERT INTO packages(title,description,price,validity_days,status,kode,jenjang,exam_type,cbt_token,start_date,end_date,kategori_id,kelas_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::timestamptz,$11::timestamptz,$12::uuid,$13::uuid) RETURNING ` + adminPackageColumns
	return r.adminPackageRow(ctx, query, input.Title, input.Description, input.Price, input.ValidityDays, input.Status, nullableKode(input.Kode), input.Jenjang, input.ExamType, input.CBTToken, nullableTimestamptz(input.StartDate), nullableTimestamptz(input.EndDate), input.KategoriID, input.KelasID)
}
func (r *AdminRepository) GetPackage(ctx context.Context, id string) (*domain.AdminPackage, error) {
	return r.adminPackageRow(ctx, `SELECT `+adminPackageColumns+` FROM packages WHERE id=$1`, id)
}
func (r *AdminRepository) UpdatePackage(ctx context.Context, id string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin package verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previousStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM packages WHERE id=$1 FOR UPDATE`, id).Scan(&previousStatus); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	} else if err != nil {
		return nil, adminMutationError(err)
	}
	const query = `UPDATE packages SET title=$2,description=$3,price=$4,validity_days=$5,status=$6,kode=$7,jenjang=$8,exam_type=$9,cbt_token=$10,start_date=$11::timestamptz,end_date=$12::timestamptz,kategori_id=$13::uuid,kelas_id=$14::uuid WHERE id=$1 RETURNING ` + adminPackageColumns
	var item domain.AdminPackage
	err = tx.QueryRow(ctx, query, id, input.Title, input.Description, input.Price, input.ValidityDays, input.Status, input.Kode, input.Jenjang, input.ExamType, input.CBTToken, nullableTimestamptz(input.StartDate), nullableTimestamptz(input.EndDate), input.KategoriID, input.KelasID).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.ExamType, &item.CBTToken, &item.StartDate, &item.EndDate, &item.KategoriID, &item.KelasID, &item.PublisherID, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	}
	if err != nil {
		return nil, adminMutationError(err)
	}
	if previousStatus != domain.StatusActive && item.Status == domain.StatusActive && item.PublisherID != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO teacher_commissions(teacher_id,package_id,kind,base_amount,amount,status,available_at)
			SELECT $1,$2,'upload_fee',0,fs.teacher_upload_fee,'available',NOW() FROM finance_settings fs WHERE fs.singleton=TRUE
			ON CONFLICT (package_id,kind) WHERE kind='upload_fee' AND split_group IS NULL DO NOTHING`, item.PublisherID, item.ID); err != nil {
			return nil, fmt.Errorf("record upload honorarium: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,title,body,link) VALUES($1,'Paket disetujui',$2,'/dashboard/teacher')`, item.PublisherID, fmt.Sprintf(`Paket "%s" sudah aktif di katalog. Honor upload tercatat di sana.`, item.Title)); err != nil {
			return nil, fmt.Errorf("notify package approval: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}
func (r *AdminRepository) adminPackageRow(ctx context.Context, query string, args ...any) (*domain.AdminPackage, error) {
	var item domain.AdminPackage
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.ExamType, &item.CBTToken, &item.StartDate, &item.EndDate, &item.KategoriID, &item.KelasID, &item.PublisherID, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *AdminRepository) DeletePackage(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete package: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var examType string
	if err := tx.QueryRow(ctx, `SELECT exam_type FROM packages WHERE id=$1`, id).Scan(&examType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPackageNotFound
		}
		return fmt.Errorf("get package exam type: %w", err)
	}
	if examType == "cbt" {
		// Paket CBT gratis dapat dihapus sampai ke seluruh datanya (riwayat
		// pengerjaan & lisensi siswa ikut terhapus via cascade).
		var hasSales bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE package_id=$1)`, id).Scan(&hasSales); err != nil {
			return fmt.Errorf("check package transactions: %w", err)
		}
		if hasSales {
			return domain.ErrPackageInUse
		}
		if _, err := tx.Exec(ctx, `DELETE FROM user_exams WHERE exam_id IN (SELECT id FROM exams WHERE package_id=$1)`, id); err != nil {
			return adminMutationError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM user_packages WHERE package_id=$1`, id); err != nil {
			return adminMutationError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM exams WHERE package_id=$1`, id); err != nil {
			return adminMutationError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM packages WHERE id=$1`, id); err != nil {
			return adminMutationError(err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit delete cbt package: %w", err)
		}
		return nil
	}
	var hasAttempts, inUse bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_exams ue JOIN exams e ON e.id=ue.exam_id WHERE e.package_id=$1)`, id).Scan(&hasAttempts); err != nil {
		return fmt.Errorf("check package attempts: %w", err)
	}
	if hasAttempts {
		return domain.ErrPackageHasAttempts
	}
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE package_id=$1) OR EXISTS(SELECT 1 FROM user_packages WHERE package_id=$1)`, id).Scan(&inUse); err != nil {
		return fmt.Errorf("check package usage: %w", err)
	}
	if inUse {
		return domain.ErrPackageInUse
	}
	// Hapus ujian beserta soal terkait (FK questions.exam_id ON DELETE CASCADE),
	// lalu paket itu sendiri dalam satu transaksi.
	if _, err := tx.Exec(ctx, `DELETE FROM exams WHERE package_id=$1`, id); err != nil {
		return adminMutationError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM packages WHERE id=$1`, id); err != nil {
		return adminMutationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete package: %w", err)
	}
	return nil
}

func (r *AdminRepository) CreateExam(ctx context.Context, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (INSERT INTO exams(package_id,title,mapel_id,tahun_ajaran_id,duration_minutes,total_questions,passing_score,status) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8) RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.publish_pembahasan,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	return r.examRow(ctx, query, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status)
}
func (r *AdminRepository) UpdateExam(ctx context.Context, id string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (UPDATE exams SET package_id=$2,title=$3,mapel_id=NULLIF($4,'')::uuid,tahun_ajaran_id=NULLIF($5,'')::uuid,duration_minutes=$6,total_questions=$7,passing_score=$8,status=$9 WHERE id=$1 RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.publish_pembahasan,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	item, err := r.examRow(ctx, query, id, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExamNotFound
	}
	return item, err
}
func (r *AdminRepository) examRow(ctx context.Context, query string, args ...any) (*domain.AdminExam, error) {
	var item domain.AdminExam
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.PublishPembahasan, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *AdminRepository) DeleteExam(ctx context.Context, id string) error {
	var hasAttempts bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_exams ue WHERE ue.exam_id=$1)`, id).Scan(&hasAttempts); err != nil {
		return fmt.Errorf("check exam attempts: %w", err)
	}
	if hasAttempts {
		return domain.ErrExamHasAttempts
	}
	return r.deleteByID(ctx, `DELETE FROM exams WHERE id=$1`, id, domain.ErrExamNotFound)
}

func (r *AdminRepository) CreateQuestion(ctx context.Context, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	optionsJSON, err := json.Marshal(input.Options)
	if err != nil {
		return nil, fmt.Errorf("encode question options: %w", err)
	}
	categoryLabelsJSON, err := json.Marshal(input.CategoryLabels)
	if err != nil {
		return nil, fmt.Errorf("encode category labels: %w", err)
	}
	const query = `WITH changed AS (INSERT INTO questions(exam_id,subject_name,content_text,question_type,presentation_type,group_code,stimulus_text,question_image_url,stimulus_image_url,category_labels_json,options_json,correct_answer,score_weight,explanation_text,status) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING *) SELECT c.id,c.exam_id,e.title,c.subject_name,c.content_text,c.question_type,c.presentation_type,COALESCE(c.group_code,''),c.stimulus_text,c.question_image_url,c.stimulus_image_url,c.category_labels_json,c.options_json,c.correct_answer,c.score_weight,c.explanation_text,c.status FROM changed c JOIN exams e ON e.id=c.exam_id`
	return r.questionRow(ctx, query, input.ExamID, input.SubjectName, input.ContentText, input.QuestionType, input.PresentationType, input.GroupCode, input.StimulusText, input.QuestionImageURL, input.StimulusImageURL, categoryLabelsJSON, optionsJSON, input.CorrectAnswer, input.ScoreWeight, input.Explanation, input.Status)
}
func (r *AdminRepository) UpdateQuestion(ctx context.Context, id string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	optionsJSON, err := json.Marshal(input.Options)
	if err != nil {
		return nil, fmt.Errorf("encode question options: %w", err)
	}
	categoryLabelsJSON, err := json.Marshal(input.CategoryLabels)
	if err != nil {
		return nil, fmt.Errorf("encode category labels: %w", err)
	}
	const query = `WITH changed AS (UPDATE questions SET exam_id=$2,subject_name=$3,content_text=$4,question_type=$5,presentation_type=$6,group_code=NULLIF($7,''),stimulus_text=$8,question_image_url=$9,stimulus_image_url=$10,category_labels_json=$11,options_json=$12,correct_answer=$13,score_weight=$14,explanation_text=$15,status=$16 WHERE id=$1 RETURNING *) SELECT c.id,c.exam_id,e.title,c.subject_name,c.content_text,c.question_type,c.presentation_type,COALESCE(c.group_code,''),c.stimulus_text,c.question_image_url,c.stimulus_image_url,c.category_labels_json,c.options_json,c.correct_answer,c.score_weight,c.explanation_text,c.status FROM changed c JOIN exams e ON e.id=c.exam_id`
	item, err := r.questionRow(ctx, query, id, input.ExamID, input.SubjectName, input.ContentText, input.QuestionType, input.PresentationType, input.GroupCode, input.StimulusText, input.QuestionImageURL, input.StimulusImageURL, categoryLabelsJSON, optionsJSON, input.CorrectAnswer, input.ScoreWeight, input.Explanation, input.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrQuestionNotFound
	}
	return item, err
}
func (r *AdminRepository) questionRow(ctx context.Context, query string, args ...any) (*domain.AdminQuestion, error) {
	var item domain.AdminQuestion
	var rawOptions []byte
	var rawCategoryLabels []byte
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.ExamID, &item.ExamTitle, &item.SubjectName, &item.ContentText, &item.QuestionType, &item.PresentationType, &item.GroupCode, &item.StimulusText, &item.QuestionImageURL, &item.StimulusImageURL, &rawCategoryLabels, &rawOptions, &item.CorrectAnswer, &item.ScoreWeight, &item.Explanation, &item.Status)
	if err != nil {
		return nil, adminMutationError(err)
	}
	if err := json.Unmarshal(rawOptions, &item.Options); err != nil {
		return nil, fmt.Errorf("decode question options: %w", err)
	}
	if err := json.Unmarshal(rawCategoryLabels, &item.CategoryLabels); err != nil {
		return nil, fmt.Errorf("decode category labels: %w", err)
	}
	return &item, nil
}
func (r *AdminRepository) DeleteQuestion(ctx context.Context, id string) error {
	return r.deleteByID(ctx, `DELETE FROM questions WHERE id=$1`, id, domain.ErrQuestionNotFound)
}
func (r *AdminRepository) BulkDeleteQuestions(ctx context.Context, ids []string, packageID string) (int, error) {
	if packageID != "" {
		tag, err := r.db.Exec(ctx, `DELETE FROM questions WHERE exam_id IN (SELECT id FROM exams WHERE package_id=$1)`, packageID)
		if err != nil {
			return 0, adminMutationError(err)
		}
		return int(tag.RowsAffected()), nil
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM questions WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return 0, adminMutationError(err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *AdminRepository) ListTransactions(ctx context.Context, page, perPage int, status string) (domain.Page[domain.AdminTransaction], error) {
	page, perPage = normalizePage(page, perPage)
	whereClause := ""
	var args []any
	if status != "" {
		whereClause = " WHERE t.payment_status=$1"
		args = append(args, status)
	}
	args = append(args, perPage, (page-1)*perPage)
	limitArg := len(args) - 1
	offsetArg := len(args)
	rows, err := r.db.Query(ctx, `SELECT t.id,t.invoice_number,u.email,COALESCE(p.title,''),
		t.amount,t.payment_status,t.created_at
		FROM transactions t LEFT JOIN packages p ON p.id=t.package_id JOIN users u ON u.id=t.user_id`+whereClause+` ORDER BY t.created_at DESC, t.id LIMIT $`+fmt.Sprint(limitArg)+` OFFSET $`+fmt.Sprint(offsetArg), args...)
	if err != nil {
		return domain.Page[domain.AdminTransaction]{}, fmt.Errorf("admin transactions: %w", err)
	}
	defer rows.Close()
	items := []domain.AdminTransaction{}
	for rows.Next() {
		var item domain.AdminTransaction
		if err := rows.Scan(&item.ID, &item.InvoiceNumber, &item.UserEmail, &item.PackageTitle, &item.Amount, &item.Status, &item.CreatedAt); err != nil {
			return domain.Page[domain.AdminTransaction]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.AdminTransaction]{}, err
	}
	var total int
	countArgs := []any{}
	if status != "" {
		countArgs = append(countArgs, status)
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions t`+whereClause, countArgs...).Scan(&total); err != nil {
		return domain.Page[domain.AdminTransaction]{}, fmt.Errorf("count admin transactions: %w", err)
	}
	return domain.NewPage(items, page, perPage, total), nil
}

func (r *AdminRepository) ListUsers(ctx context.Context, page, perPage int, role, level, q string) (*domain.AdminUsersPage, error) {
	page, perPage = normalizePage(page, perPage)
	where := []string{}
	args := []any{}
	if role != "" {
		args = append(args, role)
		where = append(where, "role=$"+fmt.Sprint(len(args)))
	}
	if level != "" {
		args = append(args, level)
		where = append(where, "school_level=$"+fmt.Sprint(len(args)))
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, "(email ILIKE $"+fmt.Sprint(len(args))+" OR COALESCE(name,'') ILIKE $"+fmt.Sprint(len(args))+")")
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}
	queryArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, `SELECT id,email,COALESCE(name,''),COALESCE(to_char(birth_date,'YYYY-MM-DD'),''),COALESCE(phone,''),role,school_level,created_at,updated_at FROM users`+whereClause+` ORDER BY created_at DESC, id LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs))+``, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("admin users list: %w", err)
	}
	defer rows.Close()
	items := []domain.UserResponse{}
	for rows.Next() {
		var item domain.UserResponse
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.BirthDate, &item.Phone, &item.Role, &item.SchoolLevel, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count admin users: %w", err)
	}
	var summary domain.UserSummary
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE role='student'), COUNT(*) FILTER (WHERE role='teacher'), COUNT(*) FILTER (WHERE role='admin') FROM users`).Scan(&summary.Total, &summary.Students, &summary.Teachers, &summary.Admins); err != nil {
		return nil, fmt.Errorf("count admin user roles: %w", err)
	}
	return &domain.AdminUsersPage{Items: items, Page: page, Count: perPage, Total: total, Summary: summary}, nil
}

func (r *AdminRepository) ListAdminPackages(ctx context.Context, page, perPage int, status, jenjang, q, examType string) (*domain.AdminPackagesPage, error) {
	page, perPage = normalizePage(page, perPage)
	where := []string{}
	args := []any{}
	if status != "" {
		args = append(args, status)
		where = append(where, "p.status=$"+fmt.Sprint(len(args)))
	}
	if jenjang != "" {
		args = append(args, jenjang)
		where = append(where, "p.jenjang=$"+fmt.Sprint(len(args)))
	}
	if examType != "" {
		args = append(args, examType)
		where = append(where, "p.exam_type=$"+fmt.Sprint(len(args)))
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, "(p.kode ILIKE $"+fmt.Sprint(len(args))+" OR p.title ILIKE $"+fmt.Sprint(len(args))+" OR COALESCE(pu.email,'') ILIKE $"+fmt.Sprint(len(args))+")")
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}
	queryArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, `SELECT `+adminPackageSelectColumns+` FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id`+whereClause+` ORDER BY p.created_at DESC, p.id LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs))+``, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("admin packages list: %w", err)
	}
	defer rows.Close()
	items := []domain.AdminPackage{}
	for rows.Next() {
		var item domain.AdminPackage
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.ExamType, &item.CBTToken, &item.StartDate, &item.EndDate, &item.KategoriID, &item.KategoriName, &item.KelasID, &item.KelasName, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.SalesCount, &item.ViewCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id`+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count admin packages: %w", err)
	}
	var counts domain.PackageStatusCounts
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE status='active'), COUNT(*) FILTER (WHERE status='inactive'), COUNT(*) FILTER (WHERE status='active' AND publisher_id IS NOT NULL), COUNT(*) FILTER (WHERE status='inactive' AND publisher_id IS NOT NULL) FROM packages`).Scan(&counts.Active, &counts.Inactive, &counts.ActiveTeacher, &counts.InactiveTeacher); err != nil {
		return nil, fmt.Errorf("count admin package statuses: %w", err)
	}
	return &domain.AdminPackagesPage{Items: items, Page: page, Count: perPage, Total: total, Counts: counts}, nil
}

func (r *AdminRepository) ListAdminExams(ctx context.Context, page, perPage int, packageID string) (domain.Page[domain.AdminExam], error) {
	page, perPage = normalizePage(page, perPage)
	whereClause := ""
	args := []any{}
	if packageID != "" {
		whereClause = " WHERE e.package_id=$1"
		args = append(args, packageID)
	}
	queryArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, `SELECT e.id,e.package_id,e.title,p.title,COALESCE(e.mapel_id::text,''),COALESCE(e.tahun_ajaran_id::text,''),e.duration_minutes,e.total_questions,e.passing_score,e.status,e.publish_pembahasan,e.created_at FROM exams e JOIN packages p ON p.id=e.package_id`+whereClause+` ORDER BY e.created_at DESC, e.id LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs))+``, queryArgs...)
	if err != nil {
		return domain.Page[domain.AdminExam]{}, fmt.Errorf("admin exams list: %w", err)
	}
	defer rows.Close()
	items := []domain.AdminExam{}
	for rows.Next() {
		var item domain.AdminExam
		if err := rows.Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.PublishPembahasan, &item.CreatedAt); err != nil {
			return domain.Page[domain.AdminExam]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.AdminExam]{}, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM exams e`+whereClause, args...).Scan(&total); err != nil {
		return domain.Page[domain.AdminExam]{}, fmt.Errorf("count admin exams: %w", err)
	}
	return domain.NewPage(items, page, perPage, total), nil
}

func (r *AdminRepository) ListCBTPublishSettings(ctx context.Context) ([]domain.CBTPublishSetting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.package_id, p.title, COALESCE(p.kode,''), p.jenjang,
		       e.title, e.duration_minutes, e.total_questions, e.passing_score,
		       e.publish_pembahasan,
		       (SELECT COUNT(*) FROM user_exams ue WHERE ue.exam_id = e.id AND ue.status = 'submitted'),
		       COALESCE(pu.email,'')
		FROM exams e
		JOIN packages p ON p.id = e.package_id
		LEFT JOIN users pu ON pu.id = p.publisher_id
		WHERE p.exam_type = 'cbt'
		ORDER BY p.created_at DESC, p.id, e.created_at DESC, e.id`)
	if err != nil {
		return nil, fmt.Errorf("list cbt publish settings: %w", err)
	}
	defer rows.Close()
	items := make([]domain.CBTPublishSetting, 0)
	for rows.Next() {
		var item domain.CBTPublishSetting
		if err := rows.Scan(&item.ExamID, &item.PackageID, &item.PackageTitle, &item.PackageKode, &item.Jenjang,
			&item.ExamTitle, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore,
			&item.PublishPembahasan, &item.Participated, &item.PublisherEmail); err != nil {
			return nil, fmt.Errorf("scan cbt publish setting: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cbt publish settings: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) SetExamPublishPembahasan(ctx context.Context, examID string, publish bool) (*domain.CBTPublishSetting, error) {
	tag, err := r.db.Exec(ctx, `UPDATE exams SET publish_pembahasan = $2 WHERE id = $1`, examID, publish)
	if err != nil {
		return nil, adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrExamNotFound
	}
	item, err := r.getCBTPublishSetting(ctx, examID)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *AdminRepository) getCBTPublishSetting(ctx context.Context, examID string) (*domain.CBTPublishSetting, error) {
	const query = `
		SELECT e.id, e.package_id, p.title, COALESCE(p.kode,''), p.jenjang,
		       e.title, e.duration_minutes, e.total_questions, e.passing_score,
		       e.publish_pembahasan,
		       (SELECT COUNT(*) FROM user_exams ue WHERE ue.exam_id = e.id AND ue.status = 'submitted'),
		       COALESCE(pu.email,'')
		FROM exams e
		JOIN packages p ON p.id = e.package_id
		LEFT JOIN users pu ON pu.id = p.publisher_id
		WHERE e.id = $1`
	var item domain.CBTPublishSetting
	err := r.db.QueryRow(ctx, query, examID).Scan(&item.ExamID, &item.PackageID, &item.PackageTitle, &item.PackageKode, &item.Jenjang,
		&item.ExamTitle, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore,
		&item.PublishPembahasan, &item.Participated, &item.PublisherEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select cbt publish setting: %w", err)
	}
	return &item, nil
}

func (r *AdminRepository) ListCBTParticipants(ctx context.Context, examID string) ([]domain.CBTParticipant, error) {
	const query = `
		SELECT ue.id, ue.user_id, COALESCE(u.name,''), u.email, COALESCE(u.school_level,''), ue.status,
		       ue.started_at, ue.finished_at, COALESCE(ue.total_score,0), e.passing_score, e.total_questions,
		       COUNT(ua.id)
		FROM user_exams ue
		JOIN users u ON u.id = ue.user_id
		JOIN exams e ON e.id = ue.exam_id
		LEFT JOIN user_answers ua ON ua.user_exam_id = ue.id
		WHERE ue.exam_id = $1
		GROUP BY ue.id, u.id, e.passing_score, e.total_questions
		ORDER BY (ue.status = 'submitted') ASC,
		         CASE WHEN ue.status = 'ongoing' THEN ue.started_at ELSE COALESCE(ue.finished_at, ue.started_at) END,
		         ue.id`
	rows, err := r.db.Query(ctx, query, examID)
	if err != nil {
		return nil, fmt.Errorf("list cbt participants: %w", err)
	}
	defer rows.Close()
	items := make([]domain.CBTParticipant, 0)
	for rows.Next() {
		var item domain.CBTParticipant
		var answered int64
		if err := rows.Scan(&item.UserExamID, &item.UserID, &item.Name, &item.Email, &item.SchoolLevel, &item.Status,
			&item.StartedAt, &item.FinishedAt, &item.TotalScore, &item.PassingScore, &item.TotalQuestions, &answered); err != nil {
			return nil, fmt.Errorf("scan cbt participant: %w", err)
		}
		if item.Status == "ongoing" {
			next := int(answered) + 1
			if next > item.TotalQuestions {
				next = item.TotalQuestions
			}
			item.CurrentQuestion = &next
		}
		item.Passed = item.Status == "submitted" && item.TotalScore >= item.PassingScore
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cbt participants: %w", err)
	}
	return items, nil
}

func (r *AdminRepository) ListAdminQuestions(ctx context.Context, page, perPage int, packageID string) (domain.Page[domain.AdminQuestion], error) {
	page, perPage = normalizePage(page, perPage)
	whereClause := ""
	args := []any{}
	if packageID != "" {
		whereClause = " WHERE e.package_id=$1"
		args = append(args, packageID)
	}
	queryArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, `SELECT q.id,q.exam_id,e.title,q.subject_name,q.content_text,q.question_type,q.presentation_type,COALESCE(q.group_code,''),q.stimulus_text,q.question_image_url,q.stimulus_image_url,q.category_labels_json,q.options_json,q.correct_answer,q.score_weight,q.explanation_text,q.status FROM questions q JOIN exams e ON e.id=q.exam_id`+whereClause+` ORDER BY q.id DESC LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs))+``, queryArgs...)
	if err != nil {
		return domain.Page[domain.AdminQuestion]{}, fmt.Errorf("admin questions list: %w", err)
	}
	defer rows.Close()
	items := []domain.AdminQuestion{}
	for rows.Next() {
		var item domain.AdminQuestion
		var rawOptions, rawCategoryLabels []byte
		if err := rows.Scan(&item.ID, &item.ExamID, &item.ExamTitle, &item.SubjectName, &item.ContentText, &item.QuestionType, &item.PresentationType, &item.GroupCode, &item.StimulusText, &item.QuestionImageURL, &item.StimulusImageURL, &rawCategoryLabels, &rawOptions, &item.CorrectAnswer, &item.ScoreWeight, &item.Explanation, &item.Status); err != nil {
			return domain.Page[domain.AdminQuestion]{}, err
		}
		if err := json.Unmarshal(rawOptions, &item.Options); err != nil {
			return domain.Page[domain.AdminQuestion]{}, fmt.Errorf("decode options for question %s: %w", item.ID, err)
		}
		if err := json.Unmarshal(rawCategoryLabels, &item.CategoryLabels); err != nil {
			return domain.Page[domain.AdminQuestion]{}, fmt.Errorf("decode category labels for question %s: %w", item.ID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.AdminQuestion]{}, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM questions q JOIN exams e ON e.id=q.exam_id`+whereClause, args...).Scan(&total); err != nil {
		return domain.Page[domain.AdminQuestion]{}, fmt.Errorf("count admin questions: %w", err)
	}
	return domain.NewPage(items, page, perPage, total), nil
}

func normalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 200 {
		perPage = 200
	}
	return page, perPage
}

// RefundTransaction membatalkan transaksi lunas: menonaktifkan lisensi paket
// dan membalik komisi guru/afiliasi (menghapus yang belum cair dan mencatat
// koreksi negatif untuk yang sudah cair) dalam satu transaksi DB.
func (r *AdminRepository) RefundTransaction(ctx context.Context, transactionID, reason string, now time.Time) (*domain.AdminTransaction, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin refund transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockQuery = `SELECT id, user_id, COALESCE(package_id::text, ''), amount, payment_status, invoice_number FROM transactions WHERE id=$1 FOR UPDATE`
	var txnID, userID, packageID, status, invoiceNumber string
	var amount float64
	if err := tx.QueryRow(ctx, lockQuery, transactionID).Scan(&txnID, &userID, &packageID, &amount, &status, &invoiceNumber); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransactionNotFound
	} else if err != nil {
		return nil, fmt.Errorf("lock refund transaction: %w", err)
	}
	if status != "paid" {
		return nil, domain.ErrTransactionNotRefundable
	}

	if _, err := tx.Exec(ctx, `UPDATE transactions SET payment_status='refunded', updated_at=NOW() WHERE id=$1`, txnID); err != nil {
		return nil, fmt.Errorf("mark transaction refunded: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE user_packages SET status='expired' WHERE user_id=$1 AND package_id=$2 AND status='active'`, userID, packageID); err != nil {
		return nil, fmt.Errorf("expire refunded license: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='cancelled' WHERE transaction_id=$1 AND kind='sales_bonus' AND status<>'paid'`, txnID); err != nil {
		return nil, fmt.Errorf("cancel refunded teacher bonus: %w", err)
	}
	const bonusReversalQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
		SELECT teacher_id,package_id,transaction_id,'refund_reversal',base_amount,rate_percent,-amount,'available',$2
		FROM teacher_commissions WHERE transaction_id=$1 AND kind='sales_bonus' AND status='paid'
		ON CONFLICT (transaction_id,kind) WHERE kind IN ('sales_bonus','refund_reversal') AND split_group IS NULL DO NOTHING`
	if _, err := tx.Exec(ctx, bonusReversalQuery, txnID, now); err != nil {
		return nil, fmt.Errorf("record teacher bonus reversal: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='cancelled' WHERE transaction_id=$1 AND kind='referral_bonus' AND status<>'paid'`, txnID); err != nil {
		return nil, fmt.Errorf("cancel refunded referral bonus: %w", err)
	}
	const referralReversalQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
		SELECT teacher_id,package_id,transaction_id,'referral_reversal',base_amount,rate_percent,-amount,'available',$2
		FROM teacher_commissions WHERE transaction_id=$1 AND kind='referral_bonus' AND status='paid'
		ON CONFLICT (transaction_id,kind) WHERE kind IN ('referral_bonus','referral_reversal') AND split_group IS NULL DO NOTHING`
	if _, err := tx.Exec(ctx, referralReversalQuery, txnID, now); err != nil {
		return nil, fmt.Errorf("record referral bonus reversal: %w", err)
	}

	userEmail, packageTitle := "", ""
	if err := tx.QueryRow(ctx, `SELECT COALESCE(u.email,''), COALESCE(p.title,'') FROM users u, packages p WHERE u.id=$1 AND p.id=$2`, userID, packageID).Scan(&userEmail, &packageTitle); err != nil {
		return nil, fmt.Errorf("read refunded transaction detail: %w", err)
	}

	item := &domain.AdminTransaction{ID: txnID, InvoiceNumber: invoiceNumber, UserEmail: userEmail, PackageTitle: packageTitle, Amount: amount, Status: "refunded", CreatedAt: now}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refund transaction: %w", err)
	}
	return item, nil
}

func (r *AdminRepository) CountOwners(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role=$1`, domain.RoleOwner).Scan(&count); err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}
	return count, nil
}

func (r *AdminRepository) GetUserRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `SELECT role FROM users WHERE id=$1`, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get user role: %w", err)
	}
	return role, nil
}

func (r *AdminRepository) InsertAuditLog(ctx context.Context, entry domain.AuditLogEntry) error {
	var actorID any
	if entry.ActorID != "" {
		actorID = entry.ActorID
	}
	_, err := r.db.Exec(ctx, `INSERT INTO audit_logs(actor_id,actor_email,action,entity_type,entity_id,detail) VALUES($1,$2,$3,$4,$5,$6)`, actorID, entry.ActorEmail, entry.Action, entry.EntityType, entry.EntityID, string(entry.Detail))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *AdminRepository) ListAuditLogs(ctx context.Context, limit int) ([]domain.AuditLogEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `SELECT id::text,COALESCE(actor_id::text,''),COALESCE(actor_email,''),action,entity_type,entity_id,detail,created_at FROM audit_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()
	items := []domain.AuditLogEntry{}
	for rows.Next() {
		var item domain.AuditLogEntry
		if err := rows.Scan(&item.ID, &item.ActorID, &item.ActorEmail, &item.Action, &item.EntityType, &item.EntityID, &item.Detail, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AdminRepository) deleteByID(ctx context.Context, query, id string, notFound error) error {
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return notFound
	}
	return nil
}

// ListTeacherVerifications menampilkan guru berstatus pending/rejected beserta
// NIK dan bukti sanggah untuk diproses operator/admin.
func (r *AdminRepository) ListTeacherVerifications(ctx context.Context, page, perPage int, status string) (domain.Page[domain.TeacherVerification], error) {
	page, perPage = normalizePage(page, perPage)
	where := `role='teacher' AND teacher_verification_status IN ('pending','rejected')`
	args := []any{}
	if status != "" {
		args = append(args, status)
		where += ` AND teacher_verification_status=$` + fmt.Sprint(len(args))
	}
	queryArgs := append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, `SELECT id,email,COALESCE(name,''),COALESCE(teacher_ktp,''),school_level,teacher_verification_status,teacher_rejection_reason,teacher_appeal_image,COALESCE(teacher_simpkb_status,'pending'),COALESCE(teacher_simpkb_image,''),created_at,updated_at
		FROM users WHERE `+where+` ORDER BY CASE teacher_verification_status WHEN 'pending' THEN 0 ELSE 1 END, created_at DESC, id LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs)), queryArgs...)
	if err != nil {
		return domain.Page[domain.TeacherVerification]{}, fmt.Errorf("teacher verification list: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherVerification{}
	for rows.Next() {
		var item domain.TeacherVerification
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.TeacherKTP, &item.SchoolLevel, &item.Status, &item.RejectionReason, &item.AppealImage, &item.SIMPKBStatus, &item.SIMPKBImage, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return domain.Page[domain.TeacherVerification]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.TeacherVerification]{}, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE `+where, args...).Scan(&total); err != nil {
		return domain.Page[domain.TeacherVerification]{}, fmt.Errorf("count teacher verification: %w", err)
	}
	return domain.NewPage(items, page, perPage, total), nil
}

// ApproveTeacher menyetujui guru sehingga bisa mengakses panel guru.
func (r *AdminRepository) ApproveTeacher(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users
		SET teacher_verification_status='approved',
		    teacher_rejection_reason='',
		    teacher_verified_at=NOW(),
		    updated_at=NOW()
		WHERE id=$1 AND role='teacher'`, userID)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// RejectTeacher menolak pendaftaran guru disertai alasan untuk ditampilkan
// pada halaman guru beserta tombol sanggah.
func (r *AdminRepository) RejectTeacher(ctx context.Context, userID, reason string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users
		SET teacher_verification_status='rejected',
		    teacher_rejection_reason=$2,
		    teacher_verified_at=NULL,
		    updated_at=NOW()
		WHERE id=$1 AND role='teacher'`, userID, reason)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// ListPendingTeacherSIMPKBChecks mengambil NIK guru yang masih menunggu dicek
// ke portal SIMPKB (verifikasi pending/ditolak & belum pernah berhasil dicek).
func (r *AdminRepository) ListPendingTeacherSIMPKBChecks(ctx context.Context, limit int) ([]domain.TeacherSIMPKBJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `SELECT id, teacher_ktp FROM users
		WHERE role='teacher' AND teacher_verification_status IN ('pending','rejected')
		  AND NULLIF(teacher_ktp,'') IS NOT NULL
		  AND (teacher_simpkb_status='pending'
		       OR (teacher_simpkb_status='checking' AND updated_at < NOW() - INTERVAL '5 minutes'))
		ORDER BY created_at ASC, id LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending simpkb checks: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherSIMPKBJob{}
	for rows.Next() {
		var item domain.TeacherSIMPKBJob
		if err := rows.Scan(&item.UserID, &item.TeacherKTP); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetTeacherKTP mengembalikan NIK guru yang masih dalam proses verifikasi.
func (r *AdminRepository) GetTeacherKTP(ctx context.Context, userID string) (string, error) {
	var ktp string
	err := r.db.QueryRow(ctx, `SELECT COALESCE(teacher_ktp,'') FROM users
		WHERE id=$1 AND role='teacher' AND teacher_verification_status IN ('pending','rejected')`, userID).Scan(&ktp)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return ktp, nil
}

// SaveTeacherSIMPKBResult menyimpan hasil dan screenshot pengecekan SIMPKB.
func (r *AdminRepository) SaveTeacherSIMPKBResult(ctx context.Context, userID, status, image string, checkedAt time.Time) error {
	tag, err := r.db.Exec(ctx, `UPDATE users
		SET teacher_simpkb_status=$2, teacher_simpkb_image=$3, teacher_simpkb_checked_at=$4, updated_at=NOW()
		WHERE id=$1 AND role='teacher'`, userID, status, image, checkedAt)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// SetTeacherSIMPKBChecking menandai pengecekan sedang berjalan (diproses async).
func (r *AdminRepository) SetTeacherSIMPKBChecking(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users
		SET teacher_simpkb_status='checking', updated_at=NOW()
		WHERE id=$1 AND role='teacher'`, userID)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func adminMutationError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if pgErr.ConstraintName == "packages_kode_idx" {
				return domain.ErrDuplicateKode
			}
			return domain.ErrEmailAlreadyExists
		case "23503":
			return domain.ErrAdminConflict
		}
	}
	return err
}

var _ domain.AdminRepository = (*AdminRepository)(nil)

// nullableTimestamptz returns nil for empty strings so the INSERT casts
// $N::timestamptz to NULL instead of failing on an empty parse.
func nullableTimestamptz(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// nullableKode maps an empty kode to NULL (CBT packages have no kode).
func nullableKode(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
