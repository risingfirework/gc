package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

const adminPackageColumns = `id,title,description,price,validity_days,status,kode,jenjang,COALESCE(publisher_id::text,''),created_at`

const adminPackageSelectColumns = `p.id,p.title,p.description,p.price,p.validity_days,p.status,p.kode,p.jenjang,COALESCE(p.publisher_id::text,''),COALESCE(pu.email,''),p.created_at,
	(SELECT COUNT(*) FROM transactions t WHERE t.package_id=p.id AND t.payment_status='paid'),
	(SELECT COUNT(*) FROM package_views pv WHERE pv.package_id=p.id)`

type TeacherRepository struct{ db *pgxpool.Pool }

func NewTeacherRepository(db *pgxpool.Pool) *TeacherRepository { return &TeacherRepository{db: db} }

func (r *TeacherRepository) GetDashboard(ctx context.Context, publisherID string) (*domain.TeacherDashboard, error) {
	result := &domain.TeacherDashboard{Levels: []string{}, Mapels: []domain.MasterItem{}, AcademicYears: []domain.MasterItem{}, Packages: []domain.AdminPackage{}, Exams: []domain.AdminExam{}, Questions: []domain.AdminQuestion{}, Transactions: []domain.AdminTransaction{}, Commissions: []domain.TeacherCommission{}, Payouts: []domain.TeacherPayout{}, PayoutRequests: []domain.TeacherWithdrawalRequest{}}
	if err := r.db.QueryRow(ctx, `SELECT `+financeSettingsColumns+` FROM finance_settings WHERE singleton=TRUE`).Scan(&result.FinanceSettings.PlatformCommissionPercent, &result.FinanceSettings.DefaultDiscountPercent, &result.FinanceSettings.TaxPercent, &result.FinanceSettings.MinimumPayout, &result.FinanceSettings.PayoutCycle, &result.FinanceSettings.AutoPayout, &result.FinanceSettings.TeacherUploadFee, &result.FinanceSettings.TeacherSalesBonusPercent, &result.FinanceSettings.CommissionHoldDays, &result.FinanceSettings.UpdatedAt); err != nil {
		return nil, fmt.Errorf("teacher finance settings: %w", err)
	}
	const counts = `SELECT
		(SELECT COUNT(*) FROM packages WHERE publisher_id=$1),
		(SELECT COUNT(*) FROM exams e JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$1),
		(SELECT COUNT(*) FROM questions q JOIN exams e ON e.id=q.exam_id JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$1),
		(SELECT COUNT(*) FROM transactions t JOIN packages p ON p.id=t.package_id WHERE p.publisher_id=$1 AND t.payment_status='paid'),
		(SELECT COALESCE(SUM(c.amount),0) FROM teacher_commissions c WHERE c.teacher_id=$1 AND c.status<>'cancelled')`
	if err := r.db.QueryRow(ctx, counts, publisherID).Scan(&result.Overview.TotalPackages, &result.Overview.TotalExams, &result.Overview.TotalQuestions, &result.Overview.TotalSales, &result.Overview.TotalRevenue); err != nil {
		return nil, fmt.Errorf("teacher overview: %w", err)
	}
	result.PayoutAccount.TeacherID = publisherID
	var payoutUpdatedAt *time.Time
	err := r.db.QueryRow(ctx, `SELECT method,provider,account_number,account_holder_name,phone,updated_at FROM teacher_payout_accounts WHERE teacher_id=$1`, publisherID).Scan(&result.PayoutAccount.Method, &result.PayoutAccount.Provider, &result.PayoutAccount.AccountNumber, &result.PayoutAccount.AccountHolderName, &result.PayoutAccount.Phone, &payoutUpdatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("teacher payout account: %w", err)
	}
	result.PayoutAccount.UpdatedAt = payoutUpdatedAt
	if err := scanLevels(ctx, r.db, &result.Levels); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterMapel, &result.Mapels); err != nil {
		return nil, err
	}
	if err := scanMasterList(ctx, r.db, domain.MasterTahunAjaran, &result.AcademicYears); err != nil {
		return nil, err
	}
	packages, err := r.db.Query(ctx, `SELECT `+adminPackageSelectColumns+` FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id WHERE p.publisher_id=$1 ORDER BY p.created_at DESC LIMIT 200`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher packages: %w", err)
	}
	defer packages.Close()
	for packages.Next() {
		var item domain.AdminPackage
		if err := packages.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.SalesCount, &item.ViewCount); err != nil {
			return nil, err
		}
		result.Packages = append(result.Packages, item)
	}
	if err := packages.Err(); err != nil {
		return nil, err
	}
	exams, err := r.db.Query(ctx, `SELECT e.id,e.package_id,e.title,p.title,COALESCE(e.mapel_id::text,''),COALESCE(e.tahun_ajaran_id::text,''),e.duration_minutes,e.total_questions,e.passing_score,e.status,e.created_at FROM exams e JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$1 ORDER BY e.created_at DESC LIMIT 100`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher exams: %w", err)
	}
	defer exams.Close()
	for exams.Next() {
		var item domain.AdminExam
		if err := exams.Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		result.Exams = append(result.Exams, item)
	}
	if err := exams.Err(); err != nil {
		return nil, err
	}
	transactions, err := r.db.Query(ctx, `SELECT t.id,t.invoice_number,u.email,p.title,t.amount,t.payment_status,t.created_at FROM transactions t JOIN users u ON u.id=t.user_id JOIN packages p ON p.id=t.package_id WHERE p.publisher_id=$1 ORDER BY t.created_at DESC LIMIT 200`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher transactions: %w", err)
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
	questions, err := r.db.Query(ctx, `SELECT q.id,q.exam_id,e.title,q.subject_name,q.content_text,q.question_type,q.presentation_type,COALESCE(q.group_code,''),q.stimulus_text,q.question_image_url,q.stimulus_image_url,q.category_labels_json,q.options_json,q.correct_answer,q.score_weight,q.explanation_text,q.status FROM questions q JOIN exams e ON e.id=q.exam_id JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$1 ORDER BY q.id DESC LIMIT 200`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher questions: %w", err)
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
	if err := questions.Err(); err != nil {
		return nil, err
	}
	const commissionsQuery = `SELECT c.id,c.teacher_id,u.email,c.package_id,p.title,COALESCE(t.invoice_number,''),c.kind,c.base_amount,c.rate_percent,c.amount,
		CASE WHEN c.status='pending' AND c.available_at<=NOW() THEN 'available' ELSE c.status END,c.available_at,c.paid_at,c.payout_reference,c.created_at
		FROM teacher_commissions c JOIN users u ON u.id=c.teacher_id JOIN packages p ON p.id=c.package_id LEFT JOIN transactions t ON t.id=c.transaction_id
		WHERE c.teacher_id=$1 ORDER BY c.created_at DESC LIMIT 300`
	commissionRows, err := r.db.Query(ctx, commissionsQuery, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher commissions: %w", err)
	}
	defer commissionRows.Close()
	for commissionRows.Next() {
		var item domain.TeacherCommission
		if err := commissionRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.PackageID, &item.PackageTitle, &item.InvoiceNumber, &item.Kind, &item.BaseAmount, &item.RatePercent, &item.Amount, &item.Status, &item.AvailableAt, &item.PaidAt, &item.PayoutReference, &item.CreatedAt); err != nil {
			return nil, err
		}
		result.Commissions = append(result.Commissions, item)
	}
	if err := commissionRows.Err(); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount) FILTER (WHERE kind='upload_fee' AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE kind IN ('sales_bonus','refund_reversal') AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE status='pending' AND available_at>NOW()),0),
		COALESCE(SUM(amount) FILTER (WHERE (status='available' OR (status='pending' AND available_at<=NOW())) AND payout_request_id IS NULL),0),
		COALESCE(SUM(amount) FILTER (WHERE status='paid'),0) FROM teacher_commissions WHERE teacher_id=$1`, publisherID).Scan(&result.CommissionSummary.UploadFees, &result.CommissionSummary.SaleBonus, &result.CommissionSummary.Held, &result.CommissionSummary.Available, &result.CommissionSummary.Paid); err != nil {
		return nil, fmt.Errorf("teacher commission summary: %w", err)
	}
	requestRows, err := r.db.Query(ctx, `SELECT pr.id,pr.teacher_id,u.email,pr.amount,pr.status,pr.payout_method,pr.provider,pr.account_number,pr.account_holder_name,pr.phone,pr.admin_note,pr.transfer_reference,pr.submitted_at,pr.reviewed_at,pr.paid_at,pr.updated_at FROM teacher_payout_requests pr JOIN users u ON u.id=pr.teacher_id WHERE pr.teacher_id=$1 ORDER BY pr.submitted_at DESC LIMIT 100`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher payout requests: %w", err)
	}
	defer requestRows.Close()
	for requestRows.Next() {
		var item domain.TeacherWithdrawalRequest
		if err := requestRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result.PayoutRequests = append(result.PayoutRequests, item)
	}
	if err := requestRows.Err(); err != nil {
		return nil, err
	}
	payoutRows, err := r.db.Query(ctx, `SELECT tp.id,tp.teacher_id,u.email,tp.amount,tp.reference,tp.paid_at FROM teacher_payouts tp JOIN users u ON u.id=tp.teacher_id WHERE tp.teacher_id=$1 ORDER BY tp.paid_at DESC LIMIT 100`, publisherID)
	if err != nil {
		return nil, fmt.Errorf("teacher payouts: %w", err)
	}
	defer payoutRows.Close()
	for payoutRows.Next() {
		var item domain.TeacherPayout
		if err := payoutRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Reference, &item.PaidAt); err != nil {
			return nil, err
		}
		result.Payouts = append(result.Payouts, item)
	}
	return result, payoutRows.Err()
}

func (r *TeacherRepository) CreatePayoutRequest(ctx context.Context, publisherID string) (*domain.TeacherWithdrawalRequest, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin payout request: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, publisherID); err != nil {
		return nil, err
	}
	var minimum, amount float64
	if err := tx.QueryRow(ctx, `SELECT minimum_payout FROM finance_settings WHERE singleton=TRUE`).Scan(&minimum); err != nil {
		return nil, err
	}
	var method, provider, number, holder, phone string
	if err := tx.QueryRow(ctx, `SELECT method,provider,account_number,account_holder_name,phone FROM teacher_payout_accounts WHERE teacher_id=$1`, publisherID).Scan(&method, &provider, &number, &holder, &phone); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPayoutAccountRequired
	} else if err != nil {
		return nil, err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teacher_payout_requests WHERE teacher_id=$1 AND status IN ('submitted','approved'))`, publisherID).Scan(&active); err != nil {
		return nil, err
	}
	if active {
		return nil, domain.ErrPayoutRequestActive
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM teacher_commissions WHERE teacher_id=$1 AND payout_request_id IS NULL AND (status='available' OR (status='pending' AND available_at<=NOW()))`, publisherID).Scan(&amount); err != nil {
		return nil, err
	}
	if amount <= 0 || amount < minimum {
		return nil, domain.ErrPayoutMinimum
	}
	var item domain.TeacherWithdrawalRequest
	if err := tx.QueryRow(ctx, `INSERT INTO teacher_payout_requests(teacher_id,amount,payout_method,provider,account_number,account_holder_name,phone) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,teacher_id,amount,status,payout_method,provider,account_number,account_holder_name,phone,admin_note,transfer_reference,submitted_at,reviewed_at,paid_at,updated_at`, publisherID, amount, method, provider, number, holder, phone).Scan(&item.ID, &item.TeacherID, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); err != nil {
		return nil, adminMutationError(err)
	}
	command, err := tx.Exec(ctx, `UPDATE teacher_commissions SET payout_request_id=$2 WHERE teacher_id=$1 AND payout_request_id IS NULL AND (status='available' OR (status='pending' AND available_at<=NOW()))`, publisherID, item.ID)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, domain.ErrPayoutMinimum
	}
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, publisherID).Scan(&item.TeacherEmail); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TeacherRepository) CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*domain.TeacherWithdrawalRequest, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item domain.TeacherWithdrawalRequest
	const query = `UPDATE teacher_payout_requests SET status='cancelled',updated_at=NOW() WHERE id=$1 AND teacher_id=$2 AND status='submitted' RETURNING id,teacher_id,amount,status,payout_method,provider,account_number,account_holder_name,phone,admin_note,transfer_reference,submitted_at,reviewed_at,paid_at,updated_at`
	if err := tx.QueryRow(ctx, query, requestID, publisherID).Scan(&item.ID, &item.TeacherID, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPayoutTransition
	} else if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET payout_request_id=NULL WHERE payout_request_id=$1 AND status<>'paid'`, requestID); err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, publisherID).Scan(&item.TeacherEmail); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TeacherRepository) UpdatePayoutAccount(ctx context.Context, publisherID string, input domain.TeacherPayoutAccount) (*domain.TeacherPayoutAccount, error) {
	const query = `INSERT INTO teacher_payout_accounts(teacher_id,method,provider,account_number,account_holder_name,phone) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(teacher_id) DO UPDATE SET method=EXCLUDED.method,provider=EXCLUDED.provider,account_number=EXCLUDED.account_number,account_holder_name=EXCLUDED.account_holder_name,phone=EXCLUDED.phone,updated_at=NOW()
		RETURNING teacher_id,method,provider,account_number,account_holder_name,phone,updated_at`
	var item domain.TeacherPayoutAccount
	var updatedAt time.Time
	if err := r.db.QueryRow(ctx, query, publisherID, input.Method, input.Provider, input.AccountNumber, input.AccountHolderName, input.Phone).Scan(&item.TeacherID, &item.Method, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &updatedAt); err != nil {
		return nil, fmt.Errorf("update teacher payout account: %w", err)
	}
	item.UpdatedAt = &updatedAt
	return &item, nil
}

func (r *TeacherRepository) CreatePackage(ctx context.Context, publisherID string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	const query = `INSERT INTO packages(title,description,price,validity_days,status,publisher_id,kode,jenjang) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING ` + adminPackageColumns
	return r.adminPackageRow(ctx, query, input.Title, input.Description, input.Price, input.ValidityDays, input.Status, publisherID, input.Kode, input.Jenjang)
}
func (r *TeacherRepository) UpdatePackage(ctx context.Context, publisherID, id string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	const query = `UPDATE packages SET title=$3,description=$4,price=$5,validity_days=$6,kode=$7,jenjang=$8 WHERE id=$1 AND publisher_id=$2 RETURNING ` + adminPackageColumns
	item, err := r.adminPackageRow(ctx, query, id, publisherID, input.Title, input.Description, input.Price, input.ValidityDays, input.Kode, input.Jenjang)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	}
	return item, err
}
func (r *TeacherRepository) adminPackageRow(ctx context.Context, query string, args ...any) (*domain.AdminPackage, error) {
	var item domain.AdminPackage
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *TeacherRepository) DeletePackage(ctx context.Context, publisherID, id string) error {
	return r.teacherDeleteByID(ctx, `DELETE FROM packages WHERE id=$1 AND publisher_id=$2`, domain.ErrPackageNotFound, id, publisherID)
}

func (r *TeacherRepository) CreateExam(ctx context.Context, publisherID string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (INSERT INTO exams(package_id,title,mapel_id,tahun_ajaran_id,duration_minutes,total_questions,passing_score,status) SELECT $1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8 FROM packages WHERE id=$1 AND publisher_id=$9 RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	item, err := r.teacherExamRow(ctx, query, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status, publisherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	}
	return item, err
}
func (r *TeacherRepository) UpdateExam(ctx context.Context, publisherID, id string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	const query = `WITH changed AS (UPDATE exams SET package_id=$3,title=$4,mapel_id=NULLIF($5,'')::uuid,tahun_ajaran_id=NULLIF($6,'')::uuid,duration_minutes=$7,total_questions=$8,passing_score=$9,status=$10 WHERE id=$1 AND package_id IN (SELECT id FROM packages WHERE publisher_id=$2) RETURNING *) SELECT c.id,c.package_id,c.title,p.title,COALESCE(c.mapel_id::text,''),COALESCE(c.tahun_ajaran_id::text,''),c.duration_minutes,c.total_questions,c.passing_score,c.status,c.created_at FROM changed c JOIN packages p ON p.id=c.package_id`
	item, err := r.teacherExamRow(ctx, query, id, publisherID, input.PackageID, input.Title, input.MapelID, input.TahunAjaranID, input.DurationMinutes, input.TotalQuestions, input.PassingScore, input.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExamNotFound
	}
	return item, err
}
func (r *TeacherRepository) teacherExamRow(ctx context.Context, query string, args ...any) (*domain.AdminExam, error) {
	var item domain.AdminExam
	err := r.db.QueryRow(ctx, query, args...).Scan(&item.ID, &item.PackageID, &item.Title, &item.PackageTitle, &item.MapelID, &item.TahunAjaranID, &item.DurationMinutes, &item.TotalQuestions, &item.PassingScore, &item.Status, &item.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
func (r *TeacherRepository) DeleteExam(ctx context.Context, publisherID, id string) error {
	return r.teacherDeleteByID(ctx, `DELETE FROM exams WHERE id=$1 AND package_id IN (SELECT id FROM packages WHERE publisher_id=$2)`, domain.ErrExamNotFound, id, publisherID)
}

func (r *TeacherRepository) CreateQuestion(ctx context.Context, publisherID string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	optionsJSON, err := json.Marshal(input.Options)
	if err != nil {
		return nil, fmt.Errorf("encode question options: %w", err)
	}
	categoryLabelsJSON, err := json.Marshal(input.CategoryLabels)
	if err != nil {
		return nil, fmt.Errorf("encode category labels: %w", err)
	}
	const query = `WITH allowed AS (SELECT e.id FROM exams e JOIN packages p ON p.id=e.package_id WHERE e.id=$16 AND p.publisher_id=$17), changed AS (INSERT INTO questions(exam_id,subject_name,content_text,question_type,presentation_type,group_code,stimulus_text,question_image_url,stimulus_image_url,category_labels_json,options_json,correct_answer,score_weight,explanation_text,status) SELECT $1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14,$15 FROM allowed RETURNING *) SELECT c.id,c.exam_id,e.title,c.subject_name,c.content_text,c.question_type,c.presentation_type,COALESCE(c.group_code,''),c.stimulus_text,c.question_image_url,c.stimulus_image_url,c.category_labels_json,c.options_json,c.correct_answer,c.score_weight,c.explanation_text,c.status FROM changed c JOIN exams e ON e.id=c.exam_id`
	item, err := r.teacherQuestionRow(ctx, query, input.ExamID, input.SubjectName, input.ContentText, input.QuestionType, input.PresentationType, input.GroupCode, input.StimulusText, input.QuestionImageURL, input.StimulusImageURL, categoryLabelsJSON, optionsJSON, input.CorrectAnswer, input.ScoreWeight, input.Explanation, input.Status, input.ExamID, publisherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExamNotFound
	}
	return item, err
}
func (r *TeacherRepository) UpdateQuestion(ctx context.Context, publisherID, id string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	optionsJSON, err := json.Marshal(input.Options)
	if err != nil {
		return nil, fmt.Errorf("encode question options: %w", err)
	}
	categoryLabelsJSON, err := json.Marshal(input.CategoryLabels)
	if err != nil {
		return nil, fmt.Errorf("encode category labels: %w", err)
	}
	const query = `WITH changed AS (UPDATE questions SET exam_id=$3,subject_name=$4,content_text=$5,question_type=$6,presentation_type=$7,group_code=NULLIF($8,''),stimulus_text=$9,question_image_url=$10,stimulus_image_url=$11,category_labels_json=$12,options_json=$13,correct_answer=$14,score_weight=$15,explanation_text=$16,status=$17 WHERE id=$1 AND exam_id IN (SELECT e.id FROM exams e JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$2) RETURNING *) SELECT c.id,c.exam_id,e.title,c.subject_name,c.content_text,c.question_type,c.presentation_type,COALESCE(c.group_code,''),c.stimulus_text,c.question_image_url,c.stimulus_image_url,c.category_labels_json,c.options_json,c.correct_answer,c.score_weight,c.explanation_text,c.status FROM changed c JOIN exams e ON e.id=c.exam_id`
	item, err := r.teacherQuestionRow(ctx, query, id, publisherID, input.ExamID, input.SubjectName, input.ContentText, input.QuestionType, input.PresentationType, input.GroupCode, input.StimulusText, input.QuestionImageURL, input.StimulusImageURL, categoryLabelsJSON, optionsJSON, input.CorrectAnswer, input.ScoreWeight, input.Explanation, input.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrQuestionNotFound
	}
	return item, err
}
func (r *TeacherRepository) teacherQuestionRow(ctx context.Context, query string, args ...any) (*domain.AdminQuestion, error) {
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
func (r *TeacherRepository) DeleteQuestion(ctx context.Context, publisherID, id string) error {
	return r.teacherDeleteByID(ctx, `DELETE FROM questions WHERE id=$1 AND exam_id IN (SELECT e.id FROM exams e JOIN packages p ON p.id=e.package_id WHERE p.publisher_id=$2)`, domain.ErrQuestionNotFound, id, publisherID)
}

func (r *TeacherRepository) teacherDeleteByID(ctx context.Context, query string, notFound error, args ...any) error {
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return adminMutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return notFound
	}
	return nil
}

var _ domain.TeacherRepository = (*TeacherRepository)(nil)
