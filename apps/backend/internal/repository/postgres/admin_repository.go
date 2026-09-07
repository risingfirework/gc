package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	result := &domain.AdminDashboard{Levels: []string{}, Jenjangs: []domain.MasterItem{}, Mapels: []domain.MasterItem{}, AcademicYears: []domain.MasterItem{}, Users: []domain.UserResponse{}, Packages: []domain.AdminPackage{}, Exams: []domain.AdminExam{}, Transactions: []domain.AdminTransaction{}, Questions: []domain.AdminQuestion{}}
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
		if err := packages.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.SalesCount, &item.ViewCount); err != nil {
			packages.Close()
			return nil, err
		}
		result.Packages = append(result.Packages, item)
	}
	if err := packages.Err(); err != nil {
		packages.Close()
		return nil, err
	}
	packages.Close()
	exams, err := r.db.Query(ctx, `SELECT e.id,e.package_id,e.title,p.title,COALESCE(e.mapel_id::text,''),COALESCE(e.tahun_ajaran_id::text,''),e.duration_minutes,e.total_questions,e.passing_score,e.status,e.created_at FROM exams e JOIN packages p ON p.id=e.package_id ORDER BY e.created_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("admin exams: %w", err)
	}
	for exams.Next() {
		var item domain.AdminExam
		if err := exams.Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.CreatedAt); err != nil {
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
	transactions, err := r.db.Query(ctx, `SELECT t.id,t.invoice_number,u.email,p.title,t.amount,t.payment_status,t.created_at FROM transactions t JOIN users u ON u.id=t.user_id JOIN packages p ON p.id=t.package_id ORDER BY t.created_at DESC LIMIT 200`)
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

func (r *AdminRepository) CreatePackage(ctx context.Context, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	const query = `INSERT INTO packages(title,description,price,validity_days,status,kode,jenjang) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING ` + adminPackageColumns
	return r.adminPackageRow(ctx, query, input.Title, input.Description, input.Price, input.ValidityDays, input.Status, input.Kode, input.Jenjang)
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
	const query = `UPDATE packages SET title=$2,description=$3,price=$4,validity_days=$5,status=$6,kode=$7,jenjang=$8 WHERE id=$1 RETURNING ` + adminPackageColumns
	var item domain.AdminPackage
	err = tx.QueryRow(ctx, query, id, input.Title, input.Description, input.Price, input.ValidityDays, input.Status, input.Kode, input.Jenjang).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	}
	if err != nil {
		return nil, adminMutationError(err)
	}
	if previousStatus != domain.StatusActive && item.Status == domain.StatusActive && item.PublisherID != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO teacher_commissions(teacher_id,package_id,kind,base_amount,amount,status,available_at)
			SELECT $1,$2,'upload_fee',0,fs.teacher_upload_fee,'available',NOW() FROM finance_settings fs WHERE fs.singleton=TRUE
			ON CONFLICT (package_id,kind) WHERE kind='upload_fee' DO NOTHING`, item.PublisherID, item.ID); err != nil {
			return nil, fmt.Errorf("record upload honorarium: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}
func (r *AdminRepository) adminPackageRow(ctx context.Context, query string, args ...any) (*domain.AdminPackage, error) {
	var item domain.AdminPackage
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *AdminRepository) DeletePackage(ctx context.Context, id string) error {
	return r.deleteByID(ctx, `DELETE FROM packages WHERE id=$1`, id, domain.ErrPackageNotFound)
}

func (r *AdminRepository) CreateExam(ctx context.Context, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (INSERT INTO exams(package_id,title,mapel_id,tahun_ajaran_id,duration_minutes,total_questions,passing_score,status) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8) RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	return r.examRow(ctx, query, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status)
}
func (r *AdminRepository) UpdateExam(ctx context.Context, id string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (UPDATE exams SET package_id=$2,title=$3,mapel_id=NULLIF($4,'')::uuid,tahun_ajaran_id=NULLIF($5,'')::uuid,duration_minutes=$6,total_questions=$7,passing_score=$8,status=$9 WHERE id=$1 RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	item, err := r.examRow(ctx, query, id, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExamNotFound
	}
	return item, err
}
func (r *AdminRepository) examRow(ctx context.Context, query string, args ...any) (*domain.AdminExam, error) {
	var item domain.AdminExam
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *AdminRepository) DeleteExam(ctx context.Context, id string) error {
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
