package domain

import (
	"context"
	"time"
)

// AffiliateDashboard berisi ringkasan kegiatan affiliate: kode rujukan,
// daftar siswa yang dirujuk, komisi referral bonus, dan pencairan.
type AffiliateDashboard struct {
	ReferralCode      string                     `json:"referral_code"`
	ReferralLink      string                     `json:"referral_link"`
	Profile           TeacherPayoutAccount       `json:"payout_account"`
	Referrals         []AffiliateReferral        `json:"referrals"`
	Commissions       []TeacherCommission        `json:"commissions"`
	CommissionSummary TeacherCommissionSummary   `json:"commission_summary"`
	Payouts           []TeacherPayout            `json:"payouts"`
	PayoutRequests    []TeacherWithdrawalRequest `json:"payout_requests"`
	FinanceSettings   FinanceSettings            `json:"finance_settings"`
}

type AffiliateReferral struct {
	UserID           string     `json:"user_id"`
	Email            string     `json:"email"`
	Name             string     `json:"name"`
	ReferredAt       time.Time  `json:"referred_at"`
	FirstPurchaseAt  *time.Time `json:"first_purchase_at"`
	CommissionID     string     `json:"commission_id,omitempty"`
	CommissionAmount float64    `json:"commission_amount,omitempty"`
}

// AffiliateRepository membaca data affiliate dari database.
type AffiliateRepository interface {
	GetDashboard(ctx context.Context, affiliateID string) (*AffiliateDashboard, error)
}

// AffiliateService adalah layanan mitra rujukan. Komisi dicairkan lewat
// mekanisme payout guru (teacher_payout_accounts / teacher_payout_requests).
type AffiliateService interface {
	GetDashboard(ctx context.Context, affiliateID string) (*AffiliateDashboard, error)
	UpdatePayoutAccount(ctx context.Context, affiliateID string, input TeacherPayoutAccount) (*TeacherPayoutAccount, error)
	CreatePayoutRequest(ctx context.Context, affiliateID string, amount float64) (*TeacherWithdrawalRequest, error)
	CancelPayoutRequest(ctx context.Context, affiliateID, requestID string) (*TeacherWithdrawalRequest, error)
}
