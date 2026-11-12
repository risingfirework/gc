package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type CBTAccessRepository struct{ db *pgxpool.Pool }

func NewCBTAccessRepository(db *pgxpool.Pool) *CBTAccessRepository {
	return &CBTAccessRepository{db: db}
}

const cbtAccessSettingsColumns = `access_mode,price_amount,currency,validity_days,free_quota,free_quota_reset,payment_mode,payment_instructions,contact_info,lock_scope,allow_manage_existing,grace_days,require_verified_teacher,updated_at`

const cbtEntitlementColumns = `user_id::text,status,free_quota_used,active_from,active_until,COALESCE(source,''),created_at,updated_at`

func scanCBTAccessSettings(row pgx.Row) (*domain.CBTAccessSettings, error) {
	var item domain.CBTAccessSettings
	if err := row.Scan(&item.AccessMode, &item.PriceAmount, &item.Currency, &item.ValidityDays, &item.FreeQuota, &item.FreeQuotaReset, &item.PaymentMode, &item.PaymentInstructions, &item.ContactInfo, &item.LockScope, &item.AllowManageExisting, &item.GraceDays, &item.RequireVerifiedTeacher, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CBTAccessRepository) GetSettings(ctx context.Context) (*domain.CBTAccessSettings, error) {
	item, err := scanCBTAccessSettings(r.db.QueryRow(ctx, `SELECT `+cbtAccessSettingsColumns+` FROM cbt_access_settings WHERE id=1`))
	if err != nil {
		return nil, fmt.Errorf("cbt access settings: %w", err)
	}
	return item, nil
}

func (r *CBTAccessRepository) UpdateSettings(ctx context.Context, input domain.CBTAccessSettings, updatedBy string) (*domain.CBTAccessSettings, error) {
	var actor any
	if updatedBy != "" {
		actor = updatedBy
	}
	item, err := scanCBTAccessSettings(r.db.QueryRow(ctx, `UPDATE cbt_access_settings SET
		access_mode=$1,price_amount=$2,currency=$3,validity_days=$4,free_quota=$5,free_quota_reset=$6,
		payment_mode=$7,payment_instructions=$8,contact_info=$9,lock_scope=$10,allow_manage_existing=$11,
		grace_days=$12,require_verified_teacher=$13,updated_by=$14,updated_at=now()
		WHERE id=1
		RETURNING `+cbtAccessSettingsColumns,
		input.AccessMode, input.PriceAmount, input.Currency, input.ValidityDays, input.FreeQuota, input.FreeQuotaReset,
		input.PaymentMode, input.PaymentInstructions, input.ContactInfo, input.LockScope, input.AllowManageExisting,
		input.GraceDays, input.RequireVerifiedTeacher, actor))
	if err != nil {
		return nil, fmt.Errorf("update cbt access settings: %w", err)
	}
	return item, nil
}

func (r *CBTAccessRepository) GetEntitlement(ctx context.Context, userID string) (*domain.TeacherCBTEntitlement, error) {
	var item domain.TeacherCBTEntitlement
	err := r.db.QueryRow(ctx, `SELECT `+cbtEntitlementColumns+` FROM teacher_cbt_entitlements WHERE user_id=$1`, userID).
		Scan(&item.UserID, &item.Status, &item.FreeQuotaUsed, &item.ActiveFrom, &item.ActiveUntil, &item.Source, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cbt entitlement: %w", err)
	}
	return &item, nil
}

func (r *CBTAccessRepository) EnsureEntitlement(ctx context.Context, userID string) (*domain.TeacherCBTEntitlement, error) {
	if _, err := r.db.Exec(ctx, `INSERT INTO teacher_cbt_entitlements (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return nil, fmt.Errorf("ensure cbt entitlement: %w", err)
	}
	return r.GetEntitlement(ctx, userID)
}

func (r *CBTAccessRepository) GetTeacherVerification(ctx context.Context, userID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT COALESCE(teacher_verification_status,'') FROM users WHERE id=$1 AND role='teacher'`, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrCBTTeacherNotFound
	}
	if err != nil {
		return "", fmt.Errorf("teacher verification: %w", err)
	}
	return status, nil
}

func (r *CBTAccessRepository) ConsumeTrial(ctx context.Context, userID string, freeQuota int, now time.Time) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("consume cbt trial begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO teacher_cbt_entitlements (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return false, fmt.Errorf("consume cbt trial ensure: %w", err)
	}
	var used int
	var status string
	if err := tx.QueryRow(ctx, `SELECT free_quota_used,status FROM teacher_cbt_entitlements WHERE user_id=$1 FOR UPDATE`, userID).Scan(&used, &status); err != nil {
		return false, fmt.Errorf("consume cbt trial lock: %w", err)
	}
	if used >= freeQuota {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_cbt_entitlements
		SET free_quota_used=free_quota_used+1,status='trial_used',source='trial',updated_at=$2
		WHERE user_id=$1`, userID, now); err != nil {
		return false, fmt.Errorf("consume cbt trial update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("consume cbt trial commit: %w", err)
	}
	return true, nil
}

func (r *CBTAccessRepository) RefundTrial(ctx context.Context, userID string, freeQuota int) error {
	if _, err := r.db.Exec(ctx, `UPDATE teacher_cbt_entitlements
		SET free_quota_used=GREATEST(free_quota_used-1,0),
		    status=CASE WHEN GREATEST(free_quota_used-1,0) < $2 THEN 'trial_available' ELSE status END,
		    updated_at=now()
		WHERE user_id=$1 AND source='trial' AND status='trial_used'`, userID, freeQuota); err != nil {
		return fmt.Errorf("refund cbt trial: %w", err)
	}
	return nil
}

func (r *CBTAccessRepository) ActivateEntitlement(ctx context.Context, userID, source string, from time.Time, until *time.Time) error {
	if _, err := r.db.Exec(ctx, `INSERT INTO teacher_cbt_entitlements (user_id,status,active_from,active_until,source,updated_at)
		VALUES ($1,'active',$2,$3,$4,now())
		ON CONFLICT (user_id) DO UPDATE SET
			status='active',active_from=EXCLUDED.active_from,active_until=EXCLUDED.active_until,
			source=EXCLUDED.source,updated_at=now()`, userID, from, until, source); err != nil {
		return fmt.Errorf("activate cbt entitlement: %w", err)
	}
	return nil
}

func (r *CBTAccessRepository) RevokeEntitlement(ctx context.Context, userID string, now time.Time) error {
	if _, err := r.db.Exec(ctx, `UPDATE teacher_cbt_entitlements
		SET status='blocked',active_until=$2,updated_at=$2
		WHERE user_id=$1`, userID, now); err != nil {
		return fmt.Errorf("revoke cbt entitlement: %w", err)
	}
	return nil
}

const cbtPaymentColumns = `p.id::text,p.user_id::text,COALESCE(u.email,''),COALESCE(u.name,''),p.amount,p.currency,COALESCE(p.proof_image,''),p.note,p.status,p.period_start,p.period_end,COALESCE(p.reviewed_by::text,''),p.reviewed_at,p.rejection_reason,p.created_at,p.updated_at`

func scanCBTPayment(row pgx.Row) (*domain.TeacherCBTPayment, error) {
	var item domain.TeacherCBTPayment
	if err := row.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.UserName, &item.Amount, &item.Currency, &item.ProofImage, &item.Note, &item.Status, &item.PeriodStart, &item.PeriodEnd, &item.ReviewedBy, &item.ReviewedAt, &item.RejectionReason, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CBTAccessRepository) GetPayment(ctx context.Context, id string) (*domain.TeacherCBTPayment, error) {
	item, err := scanCBTPayment(r.db.QueryRow(ctx, `SELECT `+cbtPaymentColumns+` FROM teacher_cbt_payments p JOIN users u ON u.id=p.user_id WHERE p.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCBTPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cbt payment: %w", err)
	}
	return item, nil
}

func (r *CBTAccessRepository) ListPayments(ctx context.Context, status string, page, perPage int) (domain.Page[domain.TeacherCBTPayment], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	where := ""
	args := []any{}
	if status != "" && status != "all" {
		where = " WHERE p.status=$1"
		args = append(args, status)
	}
	var total int
	countQuery := `SELECT COUNT(*) FROM teacher_cbt_payments p` + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return domain.Page[domain.TeacherCBTPayment]{}, fmt.Errorf("count cbt payments: %w", err)
	}
	query := `SELECT ` + cbtPaymentColumns + ` FROM teacher_cbt_payments p JOIN users u ON u.id=p.user_id` + where +
		fmt.Sprintf(` ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return domain.Page[domain.TeacherCBTPayment]{}, fmt.Errorf("list cbt payments: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherCBTPayment{}
	for rows.Next() {
		var item domain.TeacherCBTPayment
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.UserName, &item.Amount, &item.Currency, &item.ProofImage, &item.Note, &item.Status, &item.PeriodStart, &item.PeriodEnd, &item.ReviewedBy, &item.ReviewedAt, &item.RejectionReason, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return domain.Page[domain.TeacherCBTPayment]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.TeacherCBTPayment]{}, err
	}
	return domain.NewPage(items, page, perPage, total), nil
}

func (r *CBTAccessRepository) ListUserPayments(ctx context.Context, userID string, limit int) ([]domain.TeacherCBTPayment, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	rows, err := r.db.Query(ctx, `SELECT `+cbtPaymentColumns+` FROM teacher_cbt_payments p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 ORDER BY p.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user cbt payments: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherCBTPayment{}
	for rows.Next() {
		var item domain.TeacherCBTPayment
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.UserName, &item.Amount, &item.Currency, &item.ProofImage, &item.Note, &item.Status, &item.PeriodStart, &item.PeriodEnd, &item.ReviewedBy, &item.ReviewedAt, &item.RejectionReason, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CBTAccessRepository) GetPendingPayment(ctx context.Context, userID string) (*domain.TeacherCBTPayment, error) {
	item, err := scanCBTPayment(r.db.QueryRow(ctx, `SELECT `+cbtPaymentColumns+` FROM teacher_cbt_payments p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND p.status='pending' ORDER BY p.created_at DESC LIMIT 1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pending cbt payment: %w", err)
	}
	return item, nil
}

func (r *CBTAccessRepository) CreatePayment(ctx context.Context, payment domain.TeacherCBTPayment) (*domain.TeacherCBTPayment, error) {
	var id string
	var createdAt, updatedAt time.Time
	if err := r.db.QueryRow(ctx, `INSERT INTO teacher_cbt_payments (user_id,amount,currency,proof_image,note,status)
		VALUES ($1,$2,$3,$4,$5,'pending') RETURNING id::text,created_at,updated_at`,
		payment.UserID, payment.Amount, payment.Currency, payment.ProofImage, payment.Note).Scan(&id, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("create cbt payment: %w", err)
	}
	return r.GetPayment(ctx, id)
}

func (r *CBTAccessRepository) ApprovePayment(ctx context.Context, id, reviewerID string, now time.Time, periodStart, periodEnd *time.Time) (*domain.TeacherCBTPayment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("approve cbt payment begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID, status string
	if err := tx.QueryRow(ctx, `SELECT user_id::text,status FROM teacher_cbt_payments WHERE id=$1 FOR UPDATE`, id).Scan(&userID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCBTPaymentNotFound
		}
		return nil, fmt.Errorf("approve cbt payment lock: %w", err)
	}
	if status != domain.CBTPaymentPending {
		return nil, domain.ErrCBTPaymentTransition
	}
	var reviewer any
	if reviewerID != "" {
		reviewer = reviewerID
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_cbt_payments
		SET status='approved',reviewed_by=$2,reviewed_at=$3,period_start=$4,period_end=$5,updated_at=$3
		WHERE id=$1`, id, reviewer, now, periodStart, periodEnd); err != nil {
		return nil, fmt.Errorf("approve cbt payment update: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO teacher_cbt_entitlements (user_id,status,active_from,active_until,source,updated_at)
		VALUES ($1,'active',$2,$3,'payment',now())
		ON CONFLICT (user_id) DO UPDATE SET
			status='active',active_from=EXCLUDED.active_from,active_until=EXCLUDED.active_until,
			source='payment',updated_at=now()`, userID, now, periodEnd); err != nil {
		return nil, fmt.Errorf("approve cbt payment entitlement: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("approve cbt payment commit: %w", err)
	}
	return r.GetPayment(ctx, id)
}

func (r *CBTAccessRepository) RejectPayment(ctx context.Context, id, reviewerID, reason string, now time.Time) (*domain.TeacherCBTPayment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("reject cbt payment begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM teacher_cbt_payments WHERE id=$1 FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCBTPaymentNotFound
		}
		return nil, fmt.Errorf("reject cbt payment lock: %w", err)
	}
	if status != domain.CBTPaymentPending {
		return nil, domain.ErrCBTPaymentTransition
	}
	var reviewer any
	if reviewerID != "" {
		reviewer = reviewerID
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_cbt_payments
		SET status='rejected',reviewed_by=$2,reviewed_at=$3,rejection_reason=$4,updated_at=$3
		WHERE id=$1`, id, reviewer, now, reason); err != nil {
		return nil, fmt.Errorf("reject cbt payment update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("reject cbt payment commit: %w", err)
	}
	return r.GetPayment(ctx, id)
}

func (r *CBTAccessRepository) ListTeacherAccess(ctx context.Context, status, q string, page, perPage int) (domain.Page[domain.TeacherCBTAccessRow], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	where := " WHERE u.role='teacher'"
	args := []any{}
	if status != "" && status != "all" {
		args = append(args, status)
		where += fmt.Sprintf(" AND COALESCE(e.status,'trial_available')=$%d", len(args))
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		where += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.name ILIKE $%d)", len(args), len(args))
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users u LEFT JOIN teacher_cbt_entitlements e ON e.user_id=u.id`+where, args...).Scan(&total); err != nil {
		return domain.Page[domain.TeacherCBTAccessRow]{}, fmt.Errorf("count teacher cbt access: %w", err)
	}
	query := `SELECT u.id::text,COALESCE(u.email,''),COALESCE(u.name,''),COALESCE(u.teacher_verification_status,''),
		COALESCE(e.status,'trial_available'),COALESCE(e.free_quota_used,0),e.active_from,e.active_until,COALESCE(e.source,''),
		COALESCE(e.created_at,u.created_at),COALESCE(e.updated_at,u.updated_at),
		EXISTS(SELECT 1 FROM teacher_cbt_payments p WHERE p.user_id=u.id AND p.status='pending')
		FROM users u LEFT JOIN teacher_cbt_entitlements e ON e.user_id=u.id` + where +
		fmt.Sprintf(` ORDER BY u.name,u.email LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return domain.Page[domain.TeacherCBTAccessRow]{}, fmt.Errorf("list teacher cbt access: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherCBTAccessRow{}
	for rows.Next() {
		var item domain.TeacherCBTAccessRow
		if err := rows.Scan(&item.UserID, &item.Email, &item.Name, &item.VerificationStatus, &item.Entitlement.Status, &item.Entitlement.FreeQuotaUsed, &item.Entitlement.ActiveFrom, &item.Entitlement.ActiveUntil, &item.Entitlement.Source, &item.Entitlement.CreatedAt, &item.Entitlement.UpdatedAt, &item.HasPendingPayment); err != nil {
			return domain.Page[domain.TeacherCBTAccessRow]{}, err
		}
		item.Entitlement.UserID = item.UserID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.TeacherCBTAccessRow]{}, err
	}
	return domain.NewPage(items, page, perPage, total), nil
}

func (r *CBTAccessRepository) InsertAuditLog(ctx context.Context, entry domain.AuditLogEntry) error {
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
