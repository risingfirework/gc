package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tka/apps/backend/internal/domain"
)

const financeSettingsColumns = `platform_commission_percent,default_discount_percent,tax_percent,minimum_payout,payout_cycle,auto_payout,teacher_upload_fee,teacher_sales_bonus_percent,affiliate_rate_percent,commission_hold_days,updated_at`

func (r *AdminRepository) GetFinanceDashboard(ctx context.Context) (*domain.FinanceDashboard, error) {
	result := &domain.FinanceDashboard{Users: []domain.UserFinanceProfile{}, Commissions: []domain.TeacherCommission{}, Payouts: []domain.TeacherPayout{}, PayoutAccounts: []domain.TeacherPayoutAccount{}, PayoutRequests: []domain.TeacherWithdrawalRequest{}}
	if err := r.db.QueryRow(ctx, `SELECT `+financeSettingsColumns+` FROM finance_settings WHERE singleton=TRUE`).Scan(
		&result.Settings.PlatformCommissionPercent, &result.Settings.DefaultDiscountPercent, &result.Settings.TaxPercent,
		&result.Settings.MinimumPayout, &result.Settings.PayoutCycle, &result.Settings.AutoPayout, &result.Settings.TeacherUploadFee,
		&result.Settings.TeacherSalesBonusPercent, &result.Settings.AffiliateRatePercent, &result.Settings.CommissionHoldDays, &result.Settings.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("finance settings: %w", err)
	}
	const overviewQuery = `
		SELECT COALESCE((SELECT SUM(amount) FROM transactions WHERE payment_status='paid'),0),
		       COALESCE((SELECT SUM(amount) FROM transactions WHERE payment_status='pending'),0),
		       GREATEST(0,COALESCE((SELECT SUM(amount) FROM transactions WHERE payment_status='paid'),0)-COALESCE((SELECT SUM(amount) FROM teacher_commissions WHERE status<>'cancelled'),0)),
		       COALESCE((SELECT SUM(amount) FROM teacher_commissions WHERE status<>'cancelled' AND status<>'paid'),0),
		       (SELECT COUNT(*) FROM transactions WHERE payment_status='paid')`
	if err := r.db.QueryRow(ctx, overviewQuery).Scan(&result.Overview.GrossRevenue, &result.Overview.PendingRevenue, &result.Overview.PlatformCommission, &result.Overview.EstimatedPayouts, &result.Overview.PaidTransactions); err != nil {
		return nil, fmt.Errorf("finance overview: %w", err)
	}
	const usersQuery = `
		SELECT u.id,u.email,COALESCE(u.name,''),u.role,uf.commission_percent,uf.discount_percent,
		       COALESCE(uf.commission_percent,fs.platform_commission_percent),COALESCE(uf.discount_percent,fs.default_discount_percent),
		       COALESCE(uf.account_status,'active'),COALESCE(uf.notes,''),
		       COALESCE((SELECT SUM(t.amount) FROM transactions t WHERE t.user_id=u.id AND t.payment_status='paid'),0),
		       COALESCE((SELECT SUM(t.amount) FROM transactions t JOIN packages p ON p.id=t.package_id WHERE p.publisher_id=u.id AND t.payment_status='paid'),0),
		       (SELECT COUNT(*) FROM transactions t WHERE t.user_id=u.id),COALESCE(uf.updated_at,u.updated_at)
		FROM users u CROSS JOIN finance_settings fs LEFT JOIN user_finance_profiles uf ON uf.user_id=u.id
		WHERE fs.singleton=TRUE ORDER BY u.role,u.email LIMIT 500`
	rows, err := r.db.Query(ctx, usersQuery)
	if err != nil {
		return nil, fmt.Errorf("finance users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.UserFinanceProfile
		if err := rows.Scan(&item.UserID, &item.Email, &item.Name, &item.Role, &item.CommissionPercent, &item.DiscountPercent, &item.EffectiveCommissionPercent, &item.EffectiveDiscountPercent, &item.AccountStatus, &item.Notes, &item.TotalSpend, &item.GrossRevenue, &item.TransactionCount, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan finance user: %w", err)
		}
		result.Users = append(result.Users, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	accountRows, err := r.db.Query(ctx, `SELECT teacher_id,method,provider,account_number,account_holder_name,phone,updated_at FROM teacher_payout_accounts ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("finance payout accounts: %w", err)
	}
	defer accountRows.Close()
	for accountRows.Next() {
		var item domain.TeacherPayoutAccount
		var updatedAt time.Time
		if err := accountRows.Scan(&item.TeacherID, &item.Method, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan payout account: %w", err)
		}
		item.UpdatedAt = &updatedAt
		result.PayoutAccounts = append(result.PayoutAccounts, item)
	}
	if err := accountRows.Err(); err != nil {
		return nil, err
	}
	requestRows, err := r.db.Query(ctx, `SELECT pr.id,pr.teacher_id,u.email,pr.amount,pr.status,pr.payout_method,pr.provider,pr.account_number,pr.account_holder_name,pr.phone,pr.admin_note,pr.transfer_reference,pr.proof_url,pr.submitted_at,pr.reviewed_at,pr.paid_at,pr.updated_at FROM teacher_payout_requests pr JOIN users u ON u.id=pr.teacher_id ORDER BY CASE pr.status WHEN 'submitted' THEN 0 WHEN 'approved' THEN 1 ELSE 2 END,pr.submitted_at DESC LIMIT 300`)
	if err != nil {
		return nil, fmt.Errorf("finance payout requests: %w", err)
	}
	defer requestRows.Close()
	for requestRows.Next() {
		var item domain.TeacherWithdrawalRequest
		if err := requestRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.ProofURL, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan payout request: %w", err)
		}
		result.PayoutRequests = append(result.PayoutRequests, item)
	}
	if err := requestRows.Err(); err != nil {
		return nil, err
	}
	const commissionQuery = `SELECT c.id,c.teacher_id,u.email,c.package_id,p.title,COALESCE(t.invoice_number,''),c.kind,c.base_amount,c.rate_percent,c.amount,
		CASE WHEN c.status='pending' AND c.available_at<=NOW() THEN 'available' ELSE c.status END,c.available_at,c.paid_at,c.payout_reference,c.created_at
		FROM teacher_commissions c JOIN users u ON u.id=c.teacher_id JOIN packages p ON p.id=c.package_id LEFT JOIN transactions t ON t.id=c.transaction_id
		WHERE NOT (c.split_group IS NOT NULL AND c.amount=0) ORDER BY c.created_at DESC LIMIT 500`
	commissionRows, err := r.db.Query(ctx, commissionQuery)
	if err != nil {
		return nil, fmt.Errorf("finance commissions: %w", err)
	}
	defer commissionRows.Close()
	for commissionRows.Next() {
		var item domain.TeacherCommission
		if err := commissionRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.PackageID, &item.PackageTitle, &item.InvoiceNumber, &item.Kind, &item.BaseAmount, &item.RatePercent, &item.Amount, &item.Status, &item.AvailableAt, &item.PaidAt, &item.PayoutReference, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan finance commission: %w", err)
		}
		result.Commissions = append(result.Commissions, item)
	}
	if err := commissionRows.Err(); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount) FILTER (WHERE kind='upload_fee' AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE kind IN ('sales_bonus','referral_bonus','refund_reversal','referral_reversal') AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE status='pending' AND available_at>NOW()),0),
		COALESCE(SUM(amount) FILTER (WHERE (status='available' OR (status='pending' AND available_at<=NOW())) AND payout_request_id IS NULL),0),
		COALESCE(SUM(amount) FILTER (WHERE status='paid'),0) FROM teacher_commissions`).Scan(&result.Summary.UploadFees, &result.Summary.SaleBonus, &result.Summary.Held, &result.Summary.Available, &result.Summary.Paid); err != nil {
		return nil, fmt.Errorf("finance commission summary: %w", err)
	}
	payoutRows, err := r.db.Query(ctx, `SELECT tp.id,tp.teacher_id,u.email,tp.amount,tp.reference,tp.paid_at FROM teacher_payouts tp JOIN users u ON u.id=tp.teacher_id ORDER BY tp.paid_at DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("finance payouts: %w", err)
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

// ExportFinanceLedger mengembalikan seluruh baris ledger komisi (tanpa batas
// 500 milik dashboard) untuk keperluan laporan/export CSV.
func (r *AdminRepository) ExportFinanceLedger(ctx context.Context) ([]domain.TeacherCommission, error) {
	const query = `SELECT c.id,c.teacher_id,u.email,c.package_id,p.title,COALESCE(t.invoice_number,''),c.kind,c.base_amount,c.rate_percent,c.amount,
		CASE WHEN c.status='pending' AND c.available_at<=NOW() THEN 'available' ELSE c.status END,c.available_at,c.paid_at,c.payout_reference,c.created_at
		FROM teacher_commissions c JOIN users u ON u.id=c.teacher_id JOIN packages p ON p.id=c.package_id LEFT JOIN transactions t ON t.id=c.transaction_id
		WHERE NOT (c.split_group IS NOT NULL AND c.amount=0) ORDER BY c.created_at DESC, c.id`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("export finance ledger: %w", err)
	}
	defer rows.Close()
	items := []domain.TeacherCommission{}
	for rows.Next() {
		var item domain.TeacherCommission
		if err := rows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.PackageID, &item.PackageTitle, &item.InvoiceNumber, &item.Kind, &item.BaseAmount, &item.RatePercent, &item.Amount, &item.Status, &item.AvailableAt, &item.PaidAt, &item.PayoutReference, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan export ledger: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AdminRepository) UpdateFinanceSettings(ctx context.Context, input domain.FinanceSettings) (*domain.FinanceSettings, error) {
	var item domain.FinanceSettings
	err := r.db.QueryRow(ctx, `UPDATE finance_settings SET platform_commission_percent=$1,default_discount_percent=$2,tax_percent=$3,minimum_payout=$4,payout_cycle=$5,auto_payout=$6,teacher_upload_fee=$7,teacher_sales_bonus_percent=$8,affiliate_rate_percent=$9,commission_hold_days=$10,updated_at=NOW() WHERE singleton=TRUE RETURNING `+financeSettingsColumns, input.PlatformCommissionPercent, input.DefaultDiscountPercent, input.TaxPercent, input.MinimumPayout, input.PayoutCycle, input.AutoPayout, input.TeacherUploadFee, input.TeacherSalesBonusPercent, input.AffiliateRatePercent, input.CommissionHoldDays).Scan(
		&item.PlatformCommissionPercent, &item.DefaultDiscountPercent, &item.TaxPercent, &item.MinimumPayout, &item.PayoutCycle, &item.AutoPayout, &item.TeacherUploadFee, &item.TeacherSalesBonusPercent, &item.AffiliateRatePercent, &item.CommissionHoldDays, &item.UpdatedAt,
	)
	if err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}

func (r *AdminRepository) PayTeacherCommissions(ctx context.Context, teacherID, reference string) (*domain.TeacherPayout, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin teacher payout: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var minimum, amount float64
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, teacherID); err != nil {
		return nil, fmt.Errorf("lock teacher payout: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT minimum_payout FROM finance_settings WHERE singleton=TRUE`).Scan(&minimum); err != nil {
		return nil, err
	}
	var payoutAccountReady bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teacher_payout_accounts WHERE teacher_id=$1)`, teacherID).Scan(&payoutAccountReady); err != nil {
		return nil, fmt.Errorf("check teacher payout account: %w", err)
	}
	if !payoutAccountReady {
		return nil, domain.ErrInvalidInput
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM teacher_commissions WHERE teacher_id=$1 AND payout_request_id IS NULL AND (status='available' OR (status='pending' AND available_at<=NOW()))`, teacherID).Scan(&amount); err != nil {
		return nil, fmt.Errorf("calculate teacher payout: %w", err)
	}
	if amount <= 0 || amount < minimum {
		return nil, domain.ErrInvalidInput
	}
	var item domain.TeacherPayout
	if err := tx.QueryRow(ctx, `INSERT INTO teacher_payouts(teacher_id,amount,reference) VALUES($1,$2,$3) RETURNING id,teacher_id,amount,reference,paid_at`, teacherID, amount, reference).Scan(&item.ID, &item.TeacherID, &item.Amount, &item.Reference, &item.PaidAt); err != nil {
		return nil, adminMutationError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='paid',paid_at=$2,payout_reference=$3 WHERE teacher_id=$1 AND payout_request_id IS NULL AND (status='available' OR (status='pending' AND available_at<=NOW()))`, teacherID, item.PaidAt, reference); err != nil {
		return nil, fmt.Errorf("mark commissions paid: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, teacherID).Scan(&item.TeacherEmail); err != nil {
		return nil, domain.ErrUserNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) ReviewPayoutRequest(ctx context.Context, requestID string, input domain.PayoutRequestReview) (*domain.TeacherWithdrawalRequest, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin payout review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item domain.TeacherWithdrawalRequest
	const locked = `SELECT pr.id,pr.teacher_id,u.email,pr.amount,pr.status,pr.payout_method,pr.provider,pr.account_number,pr.account_holder_name,pr.phone,pr.admin_note,pr.transfer_reference,pr.proof_url,pr.submitted_at,pr.reviewed_at,pr.paid_at,pr.updated_at FROM teacher_payout_requests pr JOIN users u ON u.id=pr.teacher_id WHERE pr.id=$1 FOR UPDATE OF pr`
	if err := tx.QueryRow(ctx, locked, requestID).Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.ProofURL, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPayoutRequestNotFound
	} else if err != nil {
		return nil, err
	}
	allowed := (input.Status == "approved" && item.Status == "submitted") || (input.Status == "rejected" && (item.Status == "submitted" || item.Status == "approved")) || (input.Status == "paid" && item.Status == "approved")
	if !allowed {
		return nil, domain.ErrPayoutTransition
	}
	if input.Status == "approved" || input.Status == "paid" {
		var reservedAmount float64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM teacher_commissions WHERE payout_request_id=$1 AND status<>'cancelled'`, requestID).Scan(&reservedAmount); err != nil {
			return nil, err
		}
		if reservedAmount != item.Amount {
			return nil, fmt.Errorf("%w: nominal komisi berubah; tolak pengajuan agar guru dapat mengajukan ulang", domain.ErrPayoutTransition)
		}
	}
	if input.Status == "rejected" {
		if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET payout_request_id=NULL WHERE payout_request_id=$1 AND status<>'paid'`, requestID); err != nil {
			return nil, err
		}
	}
	if input.Status == "paid" {
		if input.Reference == "" {
			return nil, domain.ErrInvalidInput
		}
		if input.ProofURL == "" {
			return nil, domain.ErrPayoutProofRequired
		}
		var payout domain.TeacherPayout
		if err := tx.QueryRow(ctx, `INSERT INTO teacher_payouts(teacher_id,amount,reference,payout_request_id) VALUES($1,$2,$3,$4) RETURNING id,teacher_id,amount,reference,paid_at`, item.TeacherID, item.Amount, input.Reference, item.ID).Scan(&payout.ID, &payout.TeacherID, &payout.Amount, &payout.Reference, &payout.PaidAt); err != nil {
			return nil, adminMutationError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE teacher_commissions SET status='paid',paid_at=$2,payout_reference=$3 WHERE payout_request_id=$1 AND status<>'cancelled'`, requestID, payout.PaidAt, input.Reference); err != nil {
			return nil, err
		}
	}
	update := `UPDATE teacher_payout_requests SET status=$2::varchar,admin_note=$3::varchar,transfer_reference=CASE WHEN $4::varchar='' THEN transfer_reference ELSE $4::varchar END,proof_url=CASE WHEN $5::varchar='' THEN proof_url ELSE $5::varchar END,reviewed_at=CASE WHEN $2::varchar IN ('approved','rejected') THEN NOW() ELSE reviewed_at END,paid_at=CASE WHEN $2::varchar='paid' THEN NOW() ELSE paid_at END,updated_at=NOW() WHERE id=$1::uuid RETURNING status,admin_note,transfer_reference,proof_url,reviewed_at,paid_at,updated_at`
	if err := tx.QueryRow(ctx, update, requestID, input.Status, input.Note, input.Reference, input.ProofURL).Scan(&item.Status, &item.AdminNote, &item.TransferReference, &item.ProofURL, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	link := "/dashboard/teacher"
	switch item.Status {
	case "approved", "paid":
		title, body := "Pencairan disetujui", "Pengajuan pencairan Anda telah disetujui admin."
		if item.Status == "paid" {
			title, body = "Dana cair", "Dana pencairan Anda telah ditransfer."
		}
		if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,title,body,link) VALUES($1,$2,$3,$4)`, item.TeacherID, title, body, link); err != nil {
			return nil, fmt.Errorf("notify payout %s: %w", item.Status, err)
		}
	case "rejected":
		note := item.AdminNote
		if note == "" {
			note = "Anda dapat mengajukan ulang setelah meneliti kendalanya."
		}
		if _, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,title,body,link) VALUES($1,'Pencairan ditolak',$2,$3)`, item.TeacherID, note, link); err != nil {
			return nil, fmt.Errorf("notify payout rejection: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AdminRepository) ListPayoutEligibleTeachers(ctx context.Context) ([]string, error) {
	const query = `
		SELECT tc.teacher_id
		FROM teacher_commissions tc
		JOIN teacher_payout_accounts tpa ON tpa.teacher_id = tc.teacher_id
		WHERE tc.payout_request_id IS NULL AND (tc.status='available' OR (tc.status='pending' AND tc.available_at<=NOW()))
		  AND NOT EXISTS (SELECT 1 FROM teacher_payout_requests pr WHERE pr.teacher_id=tc.teacher_id AND pr.status IN ('submitted','approved'))
		GROUP BY tc.teacher_id
		HAVING SUM(tc.amount) >= (SELECT minimum_payout FROM finance_settings WHERE singleton=TRUE)`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list payout eligible teachers: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan eligible teacher: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eligible teachers: %w", err)
	}
	return ids, nil
}

func (r *AdminRepository) UpdateUserFinance(ctx context.Context, userID string, input domain.UserFinanceUpdateRequest) (*domain.UserFinanceProfile, error) {
	const upsert = `INSERT INTO user_finance_profiles(user_id,commission_percent,discount_percent,account_status,notes) SELECT id,$2,$3,$4,$5 FROM users WHERE id=$1 ON CONFLICT(user_id) DO UPDATE SET commission_percent=EXCLUDED.commission_percent,discount_percent=EXCLUDED.discount_percent,account_status=EXCLUDED.account_status,notes=EXCLUDED.notes,updated_at=NOW() RETURNING user_id`
	var changedID string
	if err := r.db.QueryRow(ctx, upsert, userID, input.CommissionPercent, input.DiscountPercent, input.AccountStatus, input.Notes).Scan(&changedID); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	} else if err != nil {
		return nil, adminMutationError(err)
	}
	const query = `SELECT u.id,u.email,COALESCE(u.name,''),u.role,uf.commission_percent,uf.discount_percent,COALESCE(uf.commission_percent,fs.platform_commission_percent),COALESCE(uf.discount_percent,fs.default_discount_percent),uf.account_status,uf.notes,COALESCE((SELECT SUM(t.amount) FROM transactions t WHERE t.user_id=u.id AND t.payment_status='paid'),0),COALESCE((SELECT SUM(t.amount) FROM transactions t JOIN packages p ON p.id=t.package_id WHERE p.publisher_id=u.id AND t.payment_status='paid'),0),(SELECT COUNT(*) FROM transactions t WHERE t.user_id=u.id),uf.updated_at FROM users u JOIN user_finance_profiles uf ON uf.user_id=u.id CROSS JOIN finance_settings fs WHERE u.id=$1 AND fs.singleton=TRUE`
	var item domain.UserFinanceProfile
	if err := r.db.QueryRow(ctx, query, changedID).Scan(&item.UserID, &item.Email, &item.Name, &item.Role, &item.CommissionPercent, &item.DiscountPercent, &item.EffectiveCommissionPercent, &item.EffectiveDiscountPercent, &item.AccountStatus, &item.Notes, &item.TotalSpend, &item.GrossRevenue, &item.TransactionCount, &item.UpdatedAt); err != nil {
		return nil, adminMutationError(err)
	}
	return &item, nil
}
