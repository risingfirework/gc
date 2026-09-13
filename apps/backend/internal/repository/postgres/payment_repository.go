package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type PaymentRepository struct{ db *pgxpool.Pool }

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository { return &PaymentRepository{db: db} }

// FindReferralAffiliate mengembalikan ID user affiliate pemilik kode rujukan.
func (r *PaymentRepository) FindReferralAffiliate(ctx context.Context, code string) (string, error) {
	var affiliateID string
	err := r.db.QueryRow(ctx, `SELECT id FROM affiliates WHERE referral_code = UPPER($1)`, strings.TrimSpace(code)).Scan(&affiliateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrReferralNotFound
	}
	if err != nil {
		return "", err
	}
	return affiliateID, nil
}

// InsertReferral mencatat atribusi first-touch; pengguna yang sama hanya
// dapat dirujuk sekali (unique constraint referrals_referred_user_uq).
func (r *PaymentRepository) InsertReferral(ctx context.Context, affiliateID, referredUserID string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO referrals(affiliate_id, referred_user_id) VALUES($1,$2)
		ON CONFLICT (referred_user_id) DO NOTHING`, affiliateID, referredUserID)
	return err
}

func (r *PaymentRepository) ListPackages(ctx context.Context, limit, offset int) ([]domain.Package, error) {
	const query = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, COALESCE(p.kode,''), p.jenjang, COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at,
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
	const query = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, COALESCE(p.kode,''), p.jenjang, COALESCE(p.publisher_id::text,''), COALESCE(pu.email,''), p.created_at FROM packages p LEFT JOIN users pu ON pu.id=p.publisher_id WHERE p.id = $1`
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
SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, COALESCE(p.kode,''), p.jenjang,
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
		  AND (
		      p.exam_type <> 'cbt'
		      OR EXISTS (
		          SELECT 1 FROM exams e2
		          WHERE e2.package_id = p.id AND e2.status = 'active'
		            AND (
		                e2.publish_pembahasan
		                OR NOT EXISTS (
		                    SELECT 1 FROM user_exams ue2
		                    WHERE ue2.exam_id = e2.id AND ue2.user_id = $1 AND ue2.status = 'submitted'
		                )
		            )
		      )
		  )
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

	const getQuery = `SELECT p.id, p.title, p.description, p.price, p.validity_days, p.status, COALESCE(p.kode,''), p.jenjang,
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
		INSERT INTO user_packages (user_id, package_id, expired_at, status)
		SELECT $1, p.id, $3::timestamptz + make_interval(days => p.validity_days), 'active'
		FROM packages p WHERE p.id = $2
		ON CONFLICT (user_id, package_id) DO UPDATE SET
			status = 'active',
			expired_at = GREATEST(user_packages.expired_at, EXCLUDED.expired_at)
				+ make_interval(days => (SELECT validity_days FROM packages WHERE id = EXCLUDED.package_id))
		RETURNING expired_at`
	var expiredAt time.Time
	if err := tx.QueryRow(ctx, licenseQuery, userID, packageID, now).Scan(&expiredAt); err != nil {
		return nil, fmt.Errorf("claim activate license: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}
	return &domain.OwnedPackage{Package: item, ExpiredAt: expiredAt, PaidAt: now}, nil
}

func (r *PaymentRepository) HasEverOwnedPackage(ctx context.Context, userID, packageID string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM user_packages WHERE user_id=$1 AND package_id=$2)`
	var owned bool
	if err := r.db.QueryRow(ctx, query, userID, packageID).Scan(&owned); err != nil {
		return false, fmt.Errorf("check historical ownership: %w", err)
	}
	return owned, nil
}

func (r *PaymentRepository) CreateOrGetTransaction(ctx context.Context, item domain.Transaction, idempotencyKey string) (*domain.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin checkout transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertQuery = `
		INSERT INTO transactions (id, user_id, package_id, invoice_number, amount, payment_status, payment_method, payment_url, expires_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8, $9)
		ON CONFLICT (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL
		DO NOTHING
		RETURNING id, user_id, package_id, invoice_number, amount, platform_commission, payment_status, payment_method, payment_url, expires_at, paid_at, created_at`
	var created domain.Transaction
	err = tx.QueryRow(ctx, insertQuery, item.ID, item.UserID, item.PackageID, item.InvoiceNumber, item.Amount, item.PaymentMethod, item.PaymentURL, item.ExpiresAt, idempotencyKey).Scan(
		&created.ID, &created.UserID, &created.PackageID, &created.InvoiceNumber, &created.Amount, &created.PlatformCommission, &created.PaymentStatus,
		&created.PaymentMethod, &created.PaymentURL, &created.ExpiresAt, &created.PaidAt, &created.CreatedAt,
	)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit checkout transaction: %w", err)
		}
		return &created, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("create checkout transaction: %w", err)
	}

	// Idempotensi: kunci yang sama mengembalikan transaksi yang sama.
	const existingQuery = `
		SELECT id, user_id, package_id, invoice_number, amount, platform_commission, payment_status, payment_method, payment_url, expires_at, paid_at, created_at
		FROM transactions WHERE user_id = $1 AND idempotency_key = $2 FOR UPDATE`
	var existing domain.Transaction
	if err := tx.QueryRow(ctx, existingQuery, item.UserID, idempotencyKey).Scan(
		&existing.ID, &existing.UserID, &existing.PackageID, &existing.InvoiceNumber, &existing.Amount, &existing.PlatformCommission, &existing.PaymentStatus,
		&existing.PaymentMethod, &existing.PaymentURL, &existing.ExpiresAt, &existing.PaidAt, &existing.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("read existing checkout: %w", err)
	}
	if existing.PackageID != item.PackageID {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit existing checkout: %w", err)
		}
		return &existing, nil
	}
	now := time.Now().UTC()
	stillOpen := existing.PaymentStatus == "pending" && (existing.ExpiresAt == nil || now.Before(*existing.ExpiresAt))
	if existing.PaymentStatus == "paid" || existing.PaymentStatus == "refunded" || stillOpen {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit existing checkout: %w", err)
		}
		return &existing, nil
	}

	// Invoice pending yang sudah kedaluwarsa (atau gagal) disegarkan ulang
	// agar bisa dibayar kembali tanpa harus menunggu job kedaluwarsa.
	const renewQuery = `
		UPDATE transactions
		SET payment_status = 'pending', amount = $3, payment_method = $4, payment_url = $5, expires_at = $6, paid_at = NULL, invoice_number = $7
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, package_id, invoice_number, amount, platform_commission, payment_status, payment_method, payment_url, expires_at, paid_at, created_at`
	if err := tx.QueryRow(ctx, renewQuery, existing.ID, existing.UserID, item.Amount, item.PaymentMethod, item.PaymentURL, item.ExpiresAt, item.InvoiceNumber).Scan(
		&created.ID, &created.UserID, &created.PackageID, &created.InvoiceNumber, &created.Amount, &created.PlatformCommission, &created.PaymentStatus,
		&created.PaymentMethod, &created.PaymentURL, &created.ExpiresAt, &created.PaidAt, &created.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("renew checkout transaction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit renewed checkout: %w", err)
	}
	return &created, nil
}

// HasPendingCheckout menyatakan apakah user masih memiliki invoice pending
// yang belum kedaluwarsa untuk paket yang sama. Kunci idempotensi saat ini
// dikecualikan agar percobaan ulang checkout yang sama tidak diblokir.
func (r *PaymentRepository) HasPendingCheckout(ctx context.Context, userID, packageID, idempotencyKey string) (bool, error) {
	const query = `SELECT EXISTS (
		SELECT 1 FROM transactions
		WHERE user_id = $1 AND package_id = $2 AND payment_status = 'pending'
		  AND idempotency_key IS DISTINCT FROM $3
		  AND (expires_at IS NULL OR expires_at > NOW())
	)`
	var exists bool
	if err := r.db.QueryRow(ctx, query, userID, packageID, idempotencyKey).Scan(&exists); err != nil {
		return false, fmt.Errorf("check pending checkout: %w", err)
	}
	return exists, nil
}

// ListPendingTransactions mengembalikan invoice pending yang masih bisa dibayar
// agar siswa dapat melanjutkan pembayaran yang sempat tertunda.
func (r *PaymentRepository) ListPendingTransactions(ctx context.Context, userID string) ([]domain.PendingTransaction, error) {
	const query = `SELECT t.id, t.invoice_number, COALESCE(p.title,''), t.amount, COALESCE(t.payment_method,''), COALESCE(t.payment_url,''), t.expires_at
		FROM transactions t LEFT JOIN packages p ON p.id=t.package_id
		WHERE t.user_id=$1 AND t.payment_status='pending' AND (t.expires_at IS NULL OR t.expires_at > NOW())
		ORDER BY t.expires_at ASC NULLS LAST LIMIT 10`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list pending transactions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.PendingTransaction, 0, 10)
	for rows.Next() {
		var item domain.PendingTransaction
		var expiresAt *time.Time
		if err := rows.Scan(&item.ID, &item.InvoiceNumber, &item.PackageTitle, &item.Amount, &item.PaymentMethod, &item.PaymentURL, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan pending transaction: %w", err)
		}
		if expiresAt != nil {
			item.ExpiresAt = *expiresAt
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending transactions: %w", err)
	}
	return items, nil
}

// ExpirePendingTransactions menandai invoice pending yang melewati batas waktu.
func (r *PaymentRepository) ExpirePendingTransactions(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `UPDATE transactions SET payment_status='expired'
		WHERE payment_status='pending' AND expires_at IS NOT NULL AND expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("expire pending transactions: %w", err)
	}
	return tag.RowsAffected(), nil
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

	const lockQuery = `SELECT id, user_id, COALESCE(package_id::text, ''), amount, payment_status, expires_at FROM transactions WHERE invoice_number = $1 FOR UPDATE`
	var transactionID, userID, packageID, currentStatus string
	var amount float64
	var expiresAt *time.Time
	if err := tx.QueryRow(ctx, lockQuery, event.InvoiceNumber).Scan(&transactionID, &userID, &packageID, &amount, &currentStatus, &expiresAt); errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrTransactionNotFound
	} else if err != nil {
		return false, fmt.Errorf("lock webhook transaction: %w", err)
	}
	if math.Abs(amount-event.Amount) > 0.005 {
		return false, domain.ErrInvalidPayment
	}
	// Pembayaran yang dilaporkan lunas setelah batas waktu ditolak;
	// invoice hanya boleh dibayar selama jendela pembayaran masih terbuka.
	if event.PaymentStatus == "paid" && expiresAt != nil && now.After(*expiresAt) {
		return false, domain.ErrPaymentExpired
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
		// Satu lisensi per (user, paket): pembelian baru memperpanjang lisensi
		// yang masih aktif, atau menghidupkan kembali lisensi yang kedaluwarsa.
		const licenseQuery = `
			INSERT INTO user_packages (user_id, package_id, expired_at, status)
			SELECT $1, p.id, $3::timestamptz + make_interval(days => p.validity_days), 'active'
			FROM packages p WHERE p.id = $2
			ON CONFLICT (user_id, package_id) DO UPDATE SET
				status = 'active',
				expired_at = GREATEST(user_packages.expired_at, EXCLUDED.expired_at)
					+ make_interval(days => (SELECT validity_days FROM packages WHERE id = EXCLUDED.package_id))
			RETURNING user_packages.id`
		if _, err := tx.Exec(ctx, licenseQuery, userID, packageID, now); err != nil {
			return false, fmt.Errorf("activate package license: %w", err)
		}
		const platQuery = `UPDATE transactions SET platform_commission=ROUND(($1*fs.platform_commission_percent/100)::numeric,2) FROM finance_settings fs WHERE id=$2 AND fs.singleton=TRUE`
		if _, err := tx.Exec(ctx, platQuery, amount, transactionID); err != nil {
			return false, fmt.Errorf("record platform commission: %w", err)
		}
		const bonusQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
			SELECT p.publisher_id,p.id,$1,'sales_bonus',$2,fs.teacher_sales_bonus_percent,ROUND(($2*(100-fs.platform_commission_percent)/100*fs.teacher_sales_bonus_percent/100)::numeric,2),'pending',$3::timestamptz+(fs.commission_hold_days*interval '1 day')
			FROM packages p CROSS JOIN finance_settings fs WHERE p.id=$4 AND p.publisher_id IS NOT NULL AND fs.singleton=TRUE
			ON CONFLICT (transaction_id,kind) WHERE kind IN ('sales_bonus','refund_reversal') AND split_group IS NULL DO NOTHING`
		bonusTag, err := tx.Exec(ctx, bonusQuery, transactionID, amount, now, packageID)
		if err != nil {
			return false, fmt.Errorf("record teacher sales bonus: %w", err)
		}
		if bonusTag.RowsAffected() > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,title,body,link)
				SELECT p.publisher_id,'Komisi penjualan','Bonus penjualan tercatat dan menunggu masa hold.','/dashboard/teacher'
				FROM packages p WHERE p.id=$1`, packageID); err != nil {
				return false, fmt.Errorf("notify sales bonus: %w", err)
			}
		}
		const referralQuery = `
				WITH claimed AS (
					UPDATE referrals r SET first_purchase_at=$3::timestamptz
					FROM users u, affiliates a
					WHERE r.referred_user_id=u.id AND u.id=$5 AND r.first_purchase_at IS NULL AND a.id=r.affiliate_id AND a.id<>u.id
					RETURNING r.affiliate_id
				)
				INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
				SELECT c.affiliate_id,$4,$1,'referral_bonus',$2,fs.affiliate_rate_percent,
					ROUND(($2*fs.platform_commission_percent/100*fs.affiliate_rate_percent/100)::numeric,2),'pending',
					$3::timestamptz+(fs.commission_hold_days*interval '1 day')
				FROM claimed c CROSS JOIN finance_settings fs WHERE fs.singleton=TRUE
				ON CONFLICT (transaction_id,kind) WHERE kind IN ('referral_bonus','referral_reversal') AND split_group IS NULL DO NOTHING`
		referralTag, err := tx.Exec(ctx, referralQuery, transactionID, amount, now, packageID, userID)
		if err != nil {
			return false, fmt.Errorf("record affiliate referral bonus: %w", err)
		}
		if referralTag.RowsAffected() > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,title,body,link)
					SELECT r.affiliate_id,'Komisi rujukan','Ada pembelian baru dari kode rujukanmu. Bonus menunggu masa hold.','/dashboard/affiliate'
					FROM referrals r WHERE r.referred_user_id=$1`, userID); err != nil {
				return false, fmt.Errorf("notify referral bonus: %w", err)
			}
		}
	}
	if event.PaymentStatus == "refunded" && currentStatus == "paid" {
		// Refund mencabut lisensi pengguna atas paket tersebut.
		if _, err := tx.Exec(ctx, `UPDATE user_packages SET status='expired' WHERE user_id=$1 AND package_id=$2 AND status='active'`, userID, packageID); err != nil {
			return false, fmt.Errorf("expire refunded package license: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='cancelled' WHERE transaction_id=$1 AND kind='sales_bonus' AND status<>'paid'`, transactionID); err != nil {
			return false, fmt.Errorf("cancel refunded teacher bonus: %w", err)
		}
		const reversalQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
			SELECT teacher_id,package_id,transaction_id,'refund_reversal',base_amount,rate_percent,-amount,'available',$2
			FROM teacher_commissions WHERE transaction_id=$1 AND kind='sales_bonus' AND status='paid'
			ON CONFLICT (transaction_id,kind) WHERE kind IN ('sales_bonus','refund_reversal') AND split_group IS NULL DO NOTHING`
		if _, err := tx.Exec(ctx, reversalQuery, transactionID, now); err != nil {
			return false, fmt.Errorf("record teacher bonus reversal: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='cancelled' WHERE transaction_id=$1 AND kind='referral_bonus' AND status<>'paid'`, transactionID); err != nil {
			return false, fmt.Errorf("cancel refunded referral bonus: %w", err)
		}
		const referralReversalQuery = `INSERT INTO teacher_commissions(teacher_id,package_id,transaction_id,kind,base_amount,rate_percent,amount,status,available_at)
			SELECT teacher_id,package_id,transaction_id,'referral_reversal',base_amount,rate_percent,-amount,'available',$2
			FROM teacher_commissions WHERE transaction_id=$1 AND kind='referral_bonus' AND status='paid'
			ON CONFLICT (transaction_id,kind) WHERE kind IN ('referral_bonus','referral_reversal') AND split_group IS NULL DO NOTHING`
		if _, err := tx.Exec(ctx, referralReversalQuery, transactionID, now); err != nil {
			return false, fmt.Errorf("record referral bonus reversal: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit webhook transaction: %w", err)
	}
	return true, nil
}

func (r *PaymentRepository) ListMyTransactions(ctx context.Context, userID string) ([]domain.AdminTransaction, error) {
	const query = `SELECT t.id, t.invoice_number, COALESCE(p.title,''), t.amount, t.payment_status, t.created_at
		FROM transactions t LEFT JOIN packages p ON p.id=t.package_id
		WHERE t.user_id=$1 AND t.payment_status IN ('paid','refunded')
		ORDER BY t.created_at DESC LIMIT 50`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list my transactions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AdminTransaction, 0, 16)
	for rows.Next() {
		var item domain.AdminTransaction
		if err := rows.Scan(&item.ID, &item.InvoiceNumber, &item.PackageTitle, &item.Amount, &item.Status, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan my transaction: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate my transactions: %w", err)
	}
	return items, nil
}

func (r *PaymentRepository) GetInvoice(ctx context.Context, transactionID, requesterID string, isAdmin bool) (*domain.Invoice, error) {
	const query = `SELECT t.id, t.invoice_number, t.amount, t.payment_status, COALESCE(t.payment_method,''), t.paid_at, t.created_at,
		COALESCE(p.title,''), COALESCE(p.kode,''), COALESCE(p.jenjang,''), COALESCE(p.validity_days,0),
		COALESCE(u.name,''), u.email, COALESCE(u.school_level,''),
		COALESCE(s.platform_name,''), COALESCE(s.platform_tagline,'')
		FROM transactions t
		LEFT JOIN packages p ON p.id=t.package_id
		JOIN users u ON u.id=t.user_id
		CROSS JOIN site_settings s
		WHERE t.id=$1 AND (t.user_id=$2 OR $3)
		AND s.singleton=TRUE`
	var item domain.Invoice
	err := r.db.QueryRow(ctx, query, transactionID, requesterID, isAdmin).Scan(
		&item.TransactionID, &item.InvoiceNumber, &item.Price, &item.PaymentStatus,
		&item.PaymentMethod, &item.PaidAt, &item.CreatedAt,
		&item.PackageTitle, &item.PackageKode, &item.PackageJenjang, &item.PackageValidity,
		&item.BuyerName, &item.BuyerEmail, &item.BuyerSchool,
		&item.PlatformName, &item.PlatformTagline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return &item, nil
}

func validPaymentTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case "pending":
		return to == "paid" || to == "failed" || to == "expired"
	case "failed":
		// Retur pembayaran boleh berhasil setelah diproses ulang gateway.
		return to == "paid" || to == "expired"
	case "expired":
		// Koreksi gateway yang tetap memproses pembayaran sebelum batas waktu.
		return to == "paid"
	case "paid":
		return to == "refunded"
	default:
		return false
	}
}

var _ domain.PaymentRepository = (*PaymentRepository)(nil)
