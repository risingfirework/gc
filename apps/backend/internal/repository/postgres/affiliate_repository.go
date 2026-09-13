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

type AffiliateRepository struct{ db *pgxpool.Pool }

func NewAffiliateRepository(db *pgxpool.Pool) *AffiliateRepository {
	return &AffiliateRepository{db: db}
}

func (r *AffiliateRepository) GetDashboard(ctx context.Context, affiliateID string) (*domain.AffiliateDashboard, error) {
	result := &domain.AffiliateDashboard{
		Referrals:      []domain.AffiliateReferral{},
		Commissions:    []domain.TeacherCommission{},
		Payouts:        []domain.TeacherPayout{},
		PayoutRequests: []domain.TeacherWithdrawalRequest{},
	}
	if err := r.db.QueryRow(ctx, `SELECT referral_code FROM affiliates WHERE id=$1`, affiliateID).Scan(&result.ReferralCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("affiliate profile: %w", err)
	}
	if err := r.db.QueryRow(ctx, `SELECT `+financeSettingsColumns+` FROM finance_settings WHERE singleton=TRUE`).Scan(&result.FinanceSettings.PlatformCommissionPercent, &result.FinanceSettings.DefaultDiscountPercent, &result.FinanceSettings.TaxPercent, &result.FinanceSettings.MinimumPayout, &result.FinanceSettings.PayoutCycle, &result.FinanceSettings.AutoPayout, &result.FinanceSettings.TeacherUploadFee, &result.FinanceSettings.TeacherSalesBonusPercent, &result.FinanceSettings.AffiliateRatePercent, &result.FinanceSettings.CommissionHoldDays, &result.FinanceSettings.UpdatedAt); err != nil {
		return nil, fmt.Errorf("affiliate finance settings: %w", err)
	}
	var payoutUpdatedAt *time.Time
	err := r.db.QueryRow(ctx, `SELECT method,provider,account_number,account_holder_name,phone,updated_at FROM teacher_payout_accounts WHERE teacher_id=$1`, affiliateID).Scan(&result.Profile.Method, &result.Profile.Provider, &result.Profile.AccountNumber, &result.Profile.AccountHolderName, &result.Profile.Phone, &payoutUpdatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("affiliate payout account: %w", err)
	}
	result.Profile.TeacherID = affiliateID
	result.Profile.UpdatedAt = payoutUpdatedAt

	const referralQuery = `
		SELECT u.id,u.email,COALESCE(u.name,''),r.referred_at,r.first_purchase_at,
		       COALESCE(c.id::text,''),COALESCE(c.amount,0)
		FROM referrals r
		JOIN users u ON u.id=r.referred_user_id
		LEFT JOIN LATERAL (
			SELECT c.id,c.amount FROM teacher_commissions c
			WHERE c.teacher_id=r.affiliate_id AND c.transaction_id IS NOT NULL
			AND c.kind='referral_bonus' AND c.status<>'cancelled'
			ORDER BY c.created_at DESC LIMIT 1
		) c ON TRUE
		WHERE r.affiliate_id=$1
		ORDER BY r.referred_at DESC LIMIT 200`
	referralRows, err := r.db.Query(ctx, referralQuery, affiliateID)
	if err != nil {
		return nil, fmt.Errorf("affiliate referrals: %w", err)
	}
	defer referralRows.Close()
	for referralRows.Next() {
		var item domain.AffiliateReferral
		if err := referralRows.Scan(&item.UserID, &item.Email, &item.Name, &item.ReferredAt, &item.FirstPurchaseAt, &item.CommissionID, &item.CommissionAmount); err != nil {
			return nil, fmt.Errorf("scan affiliate referral: %w", err)
		}
		result.Referrals = append(result.Referrals, item)
	}
	if err := referralRows.Err(); err != nil {
		return nil, err
	}

	const commissionQuery = `SELECT c.id,c.teacher_id,u.email,c.package_id,p.title,COALESCE(t.invoice_number,''),c.kind,c.base_amount,c.rate_percent,c.amount,
		CASE WHEN c.status='pending' AND c.available_at<=NOW() THEN 'available' ELSE c.status END,c.available_at,c.paid_at,c.payout_reference,c.created_at
		FROM teacher_commissions c JOIN users u ON u.id=c.teacher_id JOIN packages p ON p.id=c.package_id LEFT JOIN transactions t ON t.id=c.transaction_id
		WHERE c.teacher_id=$1 AND NOT (c.split_group IS NOT NULL AND c.amount=0) ORDER BY c.created_at DESC LIMIT 200`
	commissionRows, err := r.db.Query(ctx, commissionQuery, affiliateID)
	if err != nil {
		return nil, fmt.Errorf("affiliate commissions: %w", err)
	}
	defer commissionRows.Close()
	for commissionRows.Next() {
		var item domain.TeacherCommission
		if err := commissionRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.PackageID, &item.PackageTitle, &item.InvoiceNumber, &item.Kind, &item.BaseAmount, &item.RatePercent, &item.Amount, &item.Status, &item.AvailableAt, &item.PaidAt, &item.PayoutReference, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan affiliate commission: %w", err)
		}
		result.Commissions = append(result.Commissions, item)
	}
	if err := commissionRows.Err(); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount) FILTER (WHERE kind='upload_fee' AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE kind IN ('referral_bonus','referral_reversal') AND status<>'cancelled'),0),
		COALESCE(SUM(amount) FILTER (WHERE status='pending' AND available_at>NOW()),0),
		COALESCE(SUM(amount) FILTER (WHERE (status='available' OR (status='pending' AND available_at<=NOW())) AND payout_request_id IS NULL),0),
		COALESCE(SUM(amount) FILTER (WHERE status='paid'),0) FROM teacher_commissions WHERE teacher_id=$1`, affiliateID).Scan(&result.CommissionSummary.UploadFees, &result.CommissionSummary.SaleBonus, &result.CommissionSummary.Held, &result.CommissionSummary.Available, &result.CommissionSummary.Paid); err != nil {
		return nil, fmt.Errorf("affiliate commission summary: %w", err)
	}

	payoutRows, err := r.db.Query(ctx, `SELECT tp.id,tp.teacher_id,u.email,tp.amount,tp.reference,tp.paid_at FROM teacher_payouts tp JOIN users u ON u.id=tp.teacher_id WHERE tp.teacher_id=$1 ORDER BY tp.paid_at DESC LIMIT 100`, affiliateID)
	if err != nil {
		return nil, fmt.Errorf("affiliate payouts: %w", err)
	}
	defer payoutRows.Close()
	for payoutRows.Next() {
		var item domain.TeacherPayout
		if err := payoutRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Reference, &item.PaidAt); err != nil {
			return nil, err
		}
		result.Payouts = append(result.Payouts, item)
	}
	if err := payoutRows.Err(); err != nil {
		return nil, err
	}

	requestRows, err := r.db.Query(ctx, `SELECT pr.id,pr.teacher_id,u.email,pr.amount,pr.status,pr.payout_method,pr.provider,pr.account_number,pr.account_holder_name,pr.phone,pr.admin_note,pr.transfer_reference,pr.proof_url,pr.submitted_at,pr.reviewed_at,pr.paid_at,pr.updated_at FROM teacher_payout_requests pr JOIN users u ON u.id=pr.teacher_id WHERE pr.teacher_id=$1 ORDER BY pr.submitted_at DESC LIMIT 50`, affiliateID)
	if err != nil {
		return nil, fmt.Errorf("affiliate payout requests: %w", err)
	}
	defer requestRows.Close()
	for requestRows.Next() {
		var item domain.TeacherWithdrawalRequest
		if err := requestRows.Scan(&item.ID, &item.TeacherID, &item.TeacherEmail, &item.Amount, &item.Status, &item.PayoutMethod, &item.Provider, &item.AccountNumber, &item.AccountHolderName, &item.Phone, &item.AdminNote, &item.TransferReference, &item.ProofURL, &item.SubmittedAt, &item.ReviewedAt, &item.PaidAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan affiliate payout request: %w", err)
		}
		result.PayoutRequests = append(result.PayoutRequests, item)
	}
	if err := requestRows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var _ domain.AffiliateRepository = (*AffiliateRepository)(nil)
