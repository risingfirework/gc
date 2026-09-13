package service

import (
	"context"
	"net/url"
	"strings"

	"tka/apps/backend/internal/domain"
)

type AffiliateService struct {
	repository   domain.AffiliateRepository
	teacherRepo  domain.TeacherRepository
	publicWebURL string
}

func NewAffiliateService(repository domain.AffiliateRepository, teacherRepo domain.TeacherRepository, publicWebURL string) *AffiliateService {
	return &AffiliateService{repository: repository, teacherRepo: teacherRepo, publicWebURL: strings.TrimRight(publicWebURL, "/")}
}

func (s *AffiliateService) GetDashboard(ctx context.Context, affiliateID string) (*domain.AffiliateDashboard, error) {
	result, err := s.repository.GetDashboard(ctx, affiliateID)
	if err != nil {
		return nil, err
	}
	if s.publicWebURL != "" && result.ReferralCode != "" {
		result.ReferralLink = s.publicWebURL + "/register?ref=" + url.QueryEscape(result.ReferralCode)
	}
	return result, nil
}

func (s *AffiliateService) UpdatePayoutAccount(ctx context.Context, affiliateID string, input domain.TeacherPayoutAccount) (*domain.TeacherPayoutAccount, error) {
	input.Method = strings.ToLower(strings.TrimSpace(input.Method))
	input.Provider = strings.TrimSpace(input.Provider)
	input.AccountNumber = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(input.AccountNumber), " ", ""), "-", "")
	input.AccountHolderName = strings.Join(strings.Fields(input.AccountHolderName), " ")
	input.Phone = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(input.Phone), " ", ""), "-", "")
	if (input.Method != "bank_transfer" && input.Method != "e_wallet") || input.Provider == "" || len(input.Provider) > 80 || !payoutNumberPattern.MatchString(input.AccountNumber) || input.AccountHolderName == "" || len(input.AccountHolderName) > 120 || (input.Phone != "" && !payoutPhonePattern.MatchString(input.Phone)) {
		return nil, domain.ErrInvalidInput
	}
	return s.teacherRepo.UpdatePayoutAccount(ctx, affiliateID, input)
}

func (s *AffiliateService) CreatePayoutRequest(ctx context.Context, affiliateID string, amount float64) (*domain.TeacherWithdrawalRequest, error) {
	if !validUUID(affiliateID) || amount <= 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.teacherRepo.CreatePayoutRequest(ctx, affiliateID, amount)
}

func (s *AffiliateService) CancelPayoutRequest(ctx context.Context, affiliateID, requestID string) (*domain.TeacherWithdrawalRequest, error) {
	if !validUUID(affiliateID) || !validUUID(requestID) {
		return nil, domain.ErrInvalidInput
	}
	return s.teacherRepo.CancelPayoutRequest(ctx, affiliateID, requestID)
}

var _ domain.AffiliateService = (*AffiliateService)(nil)
