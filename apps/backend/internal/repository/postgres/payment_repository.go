package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type PaymentRepository struct{ db *pgxpool.Pool }

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository { return &PaymentRepository{db: db} }

func (r *PaymentRepository) ListPackages(ctx context.Context, limit, offset int) ([]domain.Package, error) {
	const query = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, p.kode, p.jenjang, COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at,
		(SELECT COUNT(*) FROM exams e WHERE e.package_id = p.id AND e.status = 'active'),
		(SELECT COUNT(*) FROM questions q JOIN exams e ON e.id = q.exam_id WHERE e.package_id = p.id AND e.status = 'active'),
		(SELECT COUNT(*) FROM transactions t WHERE t.package_id=p.id AND t.payment_status='paid'),
		(SELECT COUNT(*) FROM package_views pv WHERE pv.package_id=p.id)
		FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id WHERE p.status = 'active' ORDER BY p.created_at DESC, p.id LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Package, 0, limit)
	for rows.Next() {
		var item domain.Package
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.ExamCount, &item.QuestionCount, &item.SalesCount, &item.ViewCount); err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packages: %w", err)
	}
	return items, nil
}

func (r *PaymentRepository) TrackPackageView(ctx context.Context, packageID, visitorKey string) (int64, error) {
	const upsert = `INSERT INTO package_views(package_id,visitor_key)
		SELECT id,$2 FROM packages WHERE id=$1 AND status='active'
		ON CONFLICT(package_id,visitor_key) DO UPDATE SET last_viewed_at=NOW()`
	result, err := r.db.Exec(ctx, upsert, packageID, visitorKey)
	if err != nil {
		return 0, fmt.Errorf("track package view: %w", err)
	}
	if result.RowsAffected() == 0 {
		return 0, domain.ErrPackageNotFound
	}
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM package_views WHERE package_id=$1`, packageID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count package views: %w", err)
	}
	return count, nil
}

func (r *PaymentRepository) GetPackage(ctx context.Context, packageID string) (*domain.Package, error) {
	const query = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, p.kode, p.jenjang, COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id WHERE p.id = $1`
	var item domain.Package
	err := r.db.QueryRow(ctx, query, packageID).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPackageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get package: %w", err)
	}
	return &item, nil
}

func (r *PaymentRepository) ListMyPackages(ctx context.Context, userID string) ([]domain.OwnedPackage, error) {
	const query = `
		SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, p.kode, p.jenjang,
		       COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at,
		       up.expired_at, COALESCE(t.paid_at, up.expired_at),
		       (SELECT COUNT(*) FROM exams e WHERE e.package_id = p.id AND e.status = 'active'),
		       (SELECT COUNT(*) FROM questions q JOIN exams e ON e.id = q.exam_id WHERE e.package_id = p.id AND e.status = 'active')
		FROM user_packages up
		JOIN packages p ON p.id = up.package_id
		LEFT JOIN users pu ON pu.id = p.publisher_id
		LEFT JOIN LATERAL (
			SELECT tr.paid_at FROM transactions tr
			WHERE tr.user_id = up.user_id AND tr.package_id = up.package_id AND tr.payment_status = 'paid'
			ORDER BY tr.paid_at DESC LIMIT 1
		) t ON true
		WHERE up.user_id = $1 AND up.status = 'active' AND up.expired_at > NOW()
		ORDER BY up.expired_at ASC, p.created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list my packages: %w", err)
	}
	defer rows.Close()
	items := make([]domain.OwnedPackage, 0, 8)
	for rows.Next() {
		var item domain.OwnedPackage
		var paidAt *time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays, &item.Status,
			&item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt, &item.ExpiredAt, &paidAt, &item.ExamCount, &item.QuestionCount); err != nil {
			return nil, fmt.Errorf("scan my package: %w", err)
		}
		if paidAt != nil {
			item.PaidAt = *paidAt
		} else {
			item.PaidAt = item.CreatedAt
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate my packages: %w", err)
	}
	return items, nil
}

func (r *PaymentRepository) ClaimFreePackage(ctx context.Context, userID, packageID string) (*domain.OwnedPackage, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const getQuery = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, p.kode, p.jenjang,
		COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at
		FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id WHERE p.id = $1`
	var item domain.Package
	if err := tx.QueryRow(ctx, getQuery, packageID).Scan(&item.ID, &item.Title, &item.Description, &item.Price, &item.ValidityDays,
		&item.Status, &item.Kode, &item.Jenjang, &item.PublisherID, &item.PublisherEmail, &item.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPackageNotFound
		}
		return nil, fmt.Errorf("claim get package: %w", err)
	}

	now := time.Now().UTC()
	const licenseQuery = `
		WITH current_license AS (
			SELECT id FROM user_packages WHERE user_id = $1 AND package_id = $2 AND status = 'active'
			ORDER BY expired_at DESC LIMIT 1 FOR UPDATE
		), extended AS (
			UPDATE user_packages up SET expired_at = GREATEST(up.expired_at, $3) + make_interval(days => p.validity_days)
			FROM packages p, current_license cl WHERE up.id = cl.id AND p.id = $2 RETURNING up.expired_at AS expired_at
		), inserted AS (
			INSERT INTO user_packages (user_id, package_id, expired_at, status)
			SELECT $1, p.id, $3 + make_interval(days => p.validity_days), 'active'
			FROM packages p
			WHERE p.id = $2 AND NOT EXISTS (SELECT 1 FROM extended)
			RETURNING expired_at
		)
		SELECT expired_at FROM extended UNION ALL SELECT expired_at FROM inserted`
	var expiredAt time.Time
	if err := tx.QueryRow(ctx, licenseQuery, userID, packageID, now).Scan(&expiredAt); err != nil {
		return nil, fmt.Errorf("claim activate license: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}
	return &domain.OwnedPackage{Package: item, ExpiredAt: expiredAt, PaidAt: now}, nil
}

func (r *PaymentRepository) CreateOrGetTransaction(ctx context.Context, item domain.Transaction, idempotencyKey string) (*domain.Transaction, error) {
	const query = `
		INSERT INTO transactions (id, user_id, package_id, invoice_number, amount, payment_status, payment_method, payment_url, expires_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8, $9)
		ON CONFLICT (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL
		DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING id, user_id, package_id, invoice_number, amount, payment_status, payment_method, payment_url, expires_at, paid_at, created_at`
	var result domain.Transaction
	err := r.db.QueryRow(ctx, query, item.ID, item.UserID, item.PackageID, item.InvoiceNumber, item.Amount, item.PaymentMethod, item.PaymentURL, item.ExpiresAt, idempotencyKey).Scan(
		&result.ID, &result.UserID, &result.PackageID, &result.InvoiceNumber, &result.Amount, &result.PaymentStatus,
		&result.PaymentMethod, &result.PaymentURL, &result.ExpiresAt, &result.PaidAt, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create checkout transaction: %w", err)
	}
	return &result, nil
}

func (r *PaymentRepository) GetUserFinancePolicy(ctx context.Context, userID string) (float64, bool, error) {
	const query = `SELECT COALESCE(uf.discount_percent,fs.default_discount_percent),COALESCE(uf.account_status,'active')='active' FROM finance_settings fs LEFT JOIN user_finance_profiles uf ON uf.user_id=$1 WHERE fs.singleton=TRUE`
	var discount float64
	var active bool
	if err := r.db.QueryRow(ctx, query, userID).Scan(&discount, &active); err != nil {
		return 0, false, fmt.Errorf("get user finance policy: %w", err)
	}
	return discount, active, nil
}

func (r *PaymentRepository) ProcessWebhook(ctx context.Context, event domain.PaymentWebhookRequest, payloadSHA256 string, now time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return false, fmt.Errorf("begin webhook transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockQuery = `SELECT id, user_id, package_id, amount, payment_status FROM transactions WHERE invoice_number = $1 FOR UPDATE`
	var transactionID, userID, packageID, currentStatus string
	var amount float64
	if err := tx.QueryRow(ctx, lockQuery, event.InvoiceNumber).Scan(&transactionID, &userID, &packageID, &amount, &currentStatus); errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrTransactionNotFound
	} else if err != nil {
		return false, fmt.Errorf("lock webhook transaction: %w", err)
	}
	if math.Abs(amount-event.Amount) > 0.005 {
		return false, domain.ErrInvalidPayment
	}
	if !validPaymentTransition(currentStatus, event.PaymentStatus) {
		return false, domain.ErrInvalidPaymentTransition
	}

	const eventQuery = `INSERT INTO payment_webhook_events (event_id, transaction_id, payload_sha256) VALUES ($1, $2, $3) ON CONFLICT (event_id) DO NOTHING`
	tag, err := tx.Exec(ctx, eventQuery, event.EventID, transactionID, payloadSHA256)
	if err != nil {
		return false, fmt.Errorf("record webhook event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var recordedHash string
		if err := tx.QueryRow(ctx, `SELECT payload_sha256 FROM payment_webhook_events WHERE event_id = $1`, event.EventID).Scan(&recordedHash); err != nil {
			return false, fmt.Errorf("read duplicate webhook: %w", err)
		}
		if recordedHash != payloadSHA256 {
			return false, domain.ErrInvalidPayment
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit duplicate webhook: %w", err)
		}
		return false, nil
	}

	paidAt := event.PaidAt
	if event.PaymentStatus == "paid" && paidAt == nil {
		paidAt = &now
	}
	const updateQuery = `UPDATE transactions SET payment_status = $2, payment_method = COALESCE(NULLIF($3, ''), payment_method), paid_at = COALESCE($4, paid_at) WHERE id = $1`
	if _, err := tx.Exec(ctx, updateQuery, transactionID, event.PaymentStatus, event.PaymentMethod, paidAt); err != nil {
		return false, fmt.Errorf("update transaction: %w", err)
	}

	if event.PaymentStatus == "paid" && currentStatus != "paid" {
		const licenseQuery = `
			WITH current_license AS (
				SELECT id FROM user_packages WHERE user_id = $1 AND package_id = $2 AND status = 'active' ORDER BY expired_at DESC LIMIT 1 FOR UPDATE
			), extended AS (
				UPDATE user_packages up SET expired_at = GREATEST(up.expired_at, $3) + make_interval(days => p.validity_days)
				FROM packages p, current_license cl WHERE up.id = cl.id AND p.id = $2 RETURNING up.id
			)
			INSERT INTO user_packages (user_id, package_id, expired_at, status)
			SELECT $1, p.id, $3 + make_interval(days => p.validity_days), 'active' FROM packages p
			WHERE p.id = $2 AND NOT EXISTS (SELECT 1 FROM extended)`
		if _, err := tx.Exec(ctx, licenseQuery, userID, packageID, now); err != nil {
			return false, fmt.Errorf("activate package license: %w", err)
		}
		const bonusQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
			SELECT p.publisher_id,p.id,$1,'sales_bonus',$2,fs.teacher_sales_bonus_percent,ROUND(($2*fs.teacher_sales_bonus_percent/100)::numeric,2),'pending',$3+make_interval(days=>fs.commission_hold_days)
			FROM packages p CROSS JOIN finance_settings fs WHERE p.id=$4 AND p.publisher_id IS NOT NULL AND fs.singleton=TRUE
			ON CONFLICT (transaction_id,kind) WHERE kind IN ('sales_bonus','refund_reversal') DO NOTHING`
		if _, err := tx.Exec(ctx, bonusQuery, transactionID, amount, now, packageID); err != nil {
			return false, fmt.Errorf("record teacher sales bonus: %w", err)
		}
	}
	if event.PaymentStatus == "refunded" && currentStatus == "paid" {
		if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='cancelled' WHERE transaction_id=$1 AND kind='sales_bonus' AND status<>'paid'`, transactionID); err != nil {
			return false, fmt.Errorf("cancel refunded teacher bonus: %w", err)
		}
		const reversalQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
			SELECT teacher_id,package_id,transaction_id,'refund_reversal',base_amount,rate_percent,-amount,'available',$2
			FROM teacher_commissions WHERE transaction_id=$1 AND kind='sales_bonus' AND status='paid'
			ON CONFLICT (transaction_id,kind) WHERE kind IN ('sales_bonus','refund_reversal') DO NOTHING`
		if _, err := tx.Exec(ctx, reversalQuery, transactionID, now); err != nil {
			return false, fmt.Errorf("record teacher bonus reversal: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit webhook transaction: %w", err)
	}
	return true, nil
}

func validPaymentTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case "pending":
		return to == "paid" || to == "failed" || to == "expired"
	case "paid":
		return to == "refunded"
	default:
		return false
	}
}

var _ domain.PaymentRepository = (*PaymentRepository)(nil)
