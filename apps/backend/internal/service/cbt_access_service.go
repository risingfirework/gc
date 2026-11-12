package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tka/apps/backend/internal/domain"
)

const cbtTrialToken = "trial"

type CBTAccessService struct {
	repository domain.CBTAccessRepository
}

func NewCBTAccessService(repository domain.CBTAccessRepository) *CBTAccessService {
	return &CBTAccessService{repository: repository}
}

func (s *CBTAccessService) audit(ctx context.Context, actorID, actorEmail, action, entityType, entityID string, detail any) {
	raw, err := json.Marshal(detail)
	if err != nil {
		return
	}
	_ = s.repository.InsertAuditLog(ctx, domain.AuditLogEntry{
		ActorID: actorID, ActorEmail: actorEmail, Action: action, EntityType: entityType, EntityID: entityID, Detail: raw,
	})
}

func (s *CBTAccessService) GetSettings(ctx context.Context) (*domain.CBTAccessSettings, error) {
	return s.repository.GetSettings(ctx)
}

func (s *CBTAccessService) UpdateSettings(ctx context.Context, actorID, actorEmail string, input domain.CBTAccessSettings) (*domain.CBTAccessSettings, error) {
	input.AccessMode = strings.TrimSpace(input.AccessMode)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.FreeQuotaReset = strings.TrimSpace(input.FreeQuotaReset)
	input.PaymentMode = strings.TrimSpace(input.PaymentMode)
	input.LockScope = strings.TrimSpace(input.LockScope)
	input.PaymentInstructions = strings.TrimSpace(input.PaymentInstructions)
	input.ContactInfo = strings.TrimSpace(input.ContactInfo)
	if input.Currency == "" {
		input.Currency = "IDR"
	}
	if input.AccessMode != domain.CBTAccessModeFree && input.AccessMode != domain.CBTAccessModeTrialThenPaid && input.AccessMode != domain.CBTAccessModePaid {
		return nil, fmt.Errorf("%w: mode akses tidak dikenal", domain.ErrCBTAccessSettings)
	}
	if input.PriceAmount < 0 || input.ValidityDays < 0 || input.ValidityDays > 3650 ||
		input.FreeQuota < 0 || input.FreeQuota > 100 || input.GraceDays < 0 || input.GraceDays > 90 ||
		len(input.Currency) != 3 {
		return nil, fmt.Errorf("%w: nominal, masa berlaku, atau jatah tidak valid", domain.ErrCBTAccessSettings)
	}
	if input.FreeQuotaReset != domain.CBTAccessQuotaNever && input.FreeQuotaReset != domain.CBTAccessQuotaYearly {
		return nil, fmt.Errorf("%w: reset jatah tidak dikenal", domain.ErrCBTAccessSettings)
	}
	if input.PaymentMode != domain.CBTAccessPaymentManual && input.PaymentMode != domain.CBTAccessPaymentGateway {
		return nil, fmt.Errorf("%w: metode pembayaran tidak dikenal", domain.ErrCBTAccessSettings)
	}
	if input.LockScope != domain.CBTAccessLockCreateNew && input.LockScope != domain.CBTAccessLockAll {
		return nil, fmt.Errorf("%w: cakupan kunci tidak dikenal", domain.ErrCBTAccessSettings)
	}
	if len(input.PaymentInstructions) > 4000 || len(input.ContactInfo) > 500 {
		return nil, fmt.Errorf("%w: teks instruksi terlalu panjang", domain.ErrCBTAccessSettings)
	}
	item, err := s.repository.UpdateSettings(ctx, input, actorID)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_access_settings_updated", "cbt_access_settings", "1", map[string]any{
		"access_mode": item.AccessMode, "price_amount": item.PriceAmount, "validity_days": item.ValidityDays,
		"free_quota": item.FreeQuota, "lock_scope": item.LockScope, "grace_days": item.GraceDays,
	})
	return item, nil
}

func (s *CBTAccessService) GetTeacherStatus(ctx context.Context, userID string) (*domain.CBTAccessStatus, error) {
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	verified, err := s.repository.GetTeacherVerification(ctx, userID)
	if err != nil {
		return nil, err
	}
	entitlement, err := s.repository.EnsureEntitlement(ctx, userID)
	if err != nil {
		return nil, err
	}
	pending, err := s.repository.GetPendingPayment(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildStatus(settings, entitlement, verified, pending), nil
}

func (s *CBTAccessService) buildStatus(settings *domain.CBTAccessSettings, entitlement *domain.TeacherCBTEntitlement, verified string, pending *domain.TeacherCBTPayment) *domain.CBTAccessStatus {
	now := time.Now().UTC()
	current := *entitlement
	if current.Status == domain.CBTAccessEntitlementActive && current.ActiveUntil != nil && !now.Before(*current.ActiveUntil) {
		current.Status = domain.CBTAccessEntitlementExpired
	}
	canCreate, reason, message := resolveCBTAccess(settings, &current, verified, pending, now)
	return &domain.CBTAccessStatus{
		Settings:       *settings,
		Entitlement:    current,
		CanCreateCBT:   canCreate,
		Reason:         reason,
		Message:        message,
		PendingPayment: pending,
	}
}

func resolveCBTAccess(settings *domain.CBTAccessSettings, entitlement *domain.TeacherCBTEntitlement, verified string, pending *domain.TeacherCBTPayment, now time.Time) (bool, string, string) {
	if settings.RequireVerifiedTeacher && verified != domain.TeacherVerificationApproved {
		return false, domain.CBTAccessReasonVerifiedRequired, "Akun guru belum terverifikasi. Selesaikan verifikasi sebelum membuat paket CBT."
	}
	if settings.AccessMode == domain.CBTAccessModeFree {
		return true, domain.CBTAccessReasonFreeMode, "Akses CBT sedang gratis untuk seluruh guru."
	}
	if entitlement.Status == domain.CBTAccessEntitlementBlocked {
		return false, domain.CBTAccessReasonAccessDenied, "Akses CBT Anda sedang dinonaktifkan admin. Hubungi admin untuk informasi lebih lanjut."
	}
	if entitlement.Status == domain.CBTAccessEntitlementActive && (entitlement.ActiveUntil == nil || now.Before(*entitlement.ActiveUntil)) {
		if entitlement.ActiveUntil == nil {
			return true, domain.CBTAccessReasonActive, "Akses CBT aktif tanpa batas waktu."
		}
		return true, domain.CBTAccessReasonActive, "Akses CBT aktif sampai " + entitlement.ActiveUntil.Format("02 Jan 2006") + "."
	}
	if settings.AccessMode == domain.CBTAccessModeTrialThenPaid && entitlement.FreeQuotaUsed < settings.FreeQuota {
		remaining := settings.FreeQuota - entitlement.FreeQuotaUsed
		return true, domain.CBTAccessReasonTrialAvailable, fmt.Sprintf("Anda memiliki %d jatah paket CBT gratis.", remaining)
	}
	if settings.GraceDays > 0 && entitlement.ActiveUntil != nil && now.Before(entitlement.ActiveUntil.AddDate(0, 0, settings.GraceDays)) {
		graceEnd := entitlement.ActiveUntil.AddDate(0, 0, settings.GraceDays)
		return true, domain.CBTAccessReasonGracePeriod, "Masa berlaku akses CBT sudah berakhir, tetapi Anda masih dalam masa tenggang sampai " + graceEnd.Format("02 Jan 2006") + "."
	}
	if pending != nil {
		return false, domain.CBTAccessReasonPendingReview, "Pembayaran Anda sedang menunggu verifikasi admin."
	}
	if entitlement.Status == domain.CBTAccessEntitlementExpired || (entitlement.ActiveUntil != nil && !now.Before(*entitlement.ActiveUntil)) {
		return false, domain.CBTAccessReasonExpired, "Masa berlaku akses CBT sudah berakhir. Perpanjang akses untuk membuat paket baru."
	}
	return false, domain.CBTAccessReasonPaymentRequired, "Jatah gratis sudah digunakan. Aktifkan akses CBT untuk membuat paket baru."
}

// BeginCBTCreation memeriksa hak akses guru sebelum membuat paket CBT baru.
// Bila guru memakai jatah gratis, kuota dikonsumsi secara transaksional dan
// token "trial" dikembalikan agar bisa di-refund bila pembuatan paket gagal.
func (s *CBTAccessService) BeginCBTCreation(ctx context.Context, userID string) (string, error) {
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return "", err
	}
	verified, err := s.repository.GetTeacherVerification(ctx, userID)
	if err != nil {
		return "", err
	}
	entitlement, err := s.repository.EnsureEntitlement(ctx, userID)
	if err != nil {
		return "", err
	}
	pending, err := s.repository.GetPendingPayment(ctx, userID)
	if err != nil {
		return "", err
	}
	status := s.buildStatus(settings, entitlement, verified, pending)
	if !status.CanCreateCBT {
		return "", &domain.CBTAccessDeniedError{Reason: status.Reason, Message: status.Message, Access: status}
	}
	if status.Reason != domain.CBTAccessReasonTrialAvailable {
		return "", nil
	}
	ok, err := s.repository.ConsumeTrial(ctx, userID, settings.FreeQuota, time.Now().UTC())
	if err != nil {
		return "", err
	}
	if !ok {
		fresh, statusErr := s.GetTeacherStatus(ctx, userID)
		if statusErr != nil {
			return "", statusErr
		}
		return "", &domain.CBTAccessDeniedError{
			Reason:  domain.CBTAccessReasonTrialUsed,
			Message: "Jatah gratis CBT sudah habis. Aktifkan akses CBT untuk membuat paket baru.",
			Access:  fresh,
		}
	}
	s.audit(ctx, userID, "", "cbt_trial_consumed", "teacher_cbt_entitlement", userID, map[string]any{"free_quota": settings.FreeQuota})
	return cbtTrialToken, nil
}

func (s *CBTAccessService) RefundCBTCreation(ctx context.Context, userID, token string) error {
	if token != cbtTrialToken {
		return nil
	}
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return err
	}
	return s.repository.RefundTrial(ctx, userID, settings.FreeQuota)
}

func (s *CBTAccessService) SubmitPayment(ctx context.Context, userID string, input domain.CBTPaymentRequest) (*domain.TeacherCBTPayment, error) {
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings.AccessMode == domain.CBTAccessModeFree || settings.PriceAmount <= 0 {
		return nil, domain.ErrCBTPaymentNotRequired
	}
	verified, err := s.repository.GetTeacherVerification(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings.RequireVerifiedTeacher && verified != domain.TeacherVerificationApproved {
		return nil, &domain.CBTAccessDeniedError{Reason: domain.CBTAccessReasonVerifiedRequired, Message: "Akun guru belum terverifikasi."}
	}
	proof := strings.TrimSpace(input.ProofImage)
	if !strings.HasPrefix(proof, "data:image/") || len(proof) > 6<<20 {
		return nil, domain.ErrInvalidInput
	}
	note := strings.TrimSpace(input.Note)
	if len(note) > 500 {
		return nil, domain.ErrInvalidInput
	}
	pending, err := s.repository.GetPendingPayment(ctx, userID)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, domain.ErrCBTPaymentPending
	}
	item, err := s.repository.CreatePayment(ctx, domain.TeacherCBTPayment{
		UserID: userID, Amount: settings.PriceAmount, Currency: settings.Currency,
		ProofImage: proof, Note: note, Status: domain.CBTPaymentPending,
	})
	if err != nil {
		return nil, err
	}
	s.audit(ctx, userID, item.UserEmail, "cbt_payment_submitted", "teacher_cbt_payment", item.ID, map[string]any{"amount": item.Amount, "note": note})
	return item, nil
}

func (s *CBTAccessService) ListMyPayments(ctx context.Context, userID string) ([]domain.TeacherCBTPayment, error) {
	return s.repository.ListUserPayments(ctx, userID, 50)
}

func (s *CBTAccessService) ListPayments(ctx context.Context, status string, page, perPage int) (domain.Page[domain.TeacherCBTPayment], error) {
	return s.repository.ListPayments(ctx, status, page, perPage)
}

func (s *CBTAccessService) ApprovePayment(ctx context.Context, actorID, actorEmail, paymentID string, input domain.CBTPaymentReviewRequest) (*domain.TeacherCBTPayment, error) {
	payment, err := s.repository.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment.Status == domain.CBTPaymentApproved {
		return payment, nil
	}
	if payment.Status != domain.CBTPaymentPending {
		return nil, domain.ErrCBTPaymentTransition
	}
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	start := now
	if input.PeriodStart != nil {
		start = input.PeriodStart.UTC()
	}
	var end *time.Time
	if input.PeriodEnd != nil {
		value := input.PeriodEnd.UTC()
		end = &value
	} else if settings.ValidityDays > 0 {
		value := start.AddDate(0, 0, settings.ValidityDays)
		end = &value
	}
	if end != nil && !end.After(start) {
		return nil, domain.ErrInvalidInput
	}
	item, err := s.repository.ApprovePayment(ctx, paymentID, actorID, now, &start, end)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_payment_approved", "teacher_cbt_payment", paymentID, map[string]any{
		"user_id": payment.UserID, "amount": payment.Amount, "period_start": start, "period_end": end, "note": strings.TrimSpace(input.Note),
	})
	return item, nil
}

func (s *CBTAccessService) RejectPayment(ctx context.Context, actorID, actorEmail, paymentID string, input domain.CBTPaymentReviewRequest) (*domain.TeacherCBTPayment, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, domain.ErrInvalidInput
	}
	payment, err := s.repository.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment.Status == domain.CBTPaymentRejected {
		return payment, nil
	}
	if payment.Status != domain.CBTPaymentPending {
		return nil, domain.ErrCBTPaymentTransition
	}
	item, err := s.repository.RejectPayment(ctx, paymentID, actorID, reason, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_payment_rejected", "teacher_cbt_payment", paymentID, map[string]any{"user_id": payment.UserID, "reason": reason})
	return item, nil
}

func (s *CBTAccessService) ListTeacherAccess(ctx context.Context, status, q string, page, perPage int) (domain.Page[domain.TeacherCBTAccessRow], error) {
	settings, err := s.repository.GetSettings(ctx)
	if err != nil {
		return domain.Page[domain.TeacherCBTAccessRow]{}, err
	}
	result, err := s.repository.ListTeacherAccess(ctx, status, strings.TrimSpace(q), page, perPage)
	if err != nil {
		return result, err
	}
	now := time.Now().UTC()
	for index := range result.Items {
		row := &result.Items[index]
		var pending *domain.TeacherCBTPayment
		if row.HasPendingPayment {
			pending = &domain.TeacherCBTPayment{Status: domain.CBTPaymentPending}
		}
		current := row.Entitlement
		if current.Status == domain.CBTAccessEntitlementActive && current.ActiveUntil != nil && !now.Before(*current.ActiveUntil) {
			current.Status = domain.CBTAccessEntitlementExpired
		}
		canCreate, reason, message := resolveCBTAccess(settings, &current, row.VerificationStatus, pending, now)
		row.Entitlement = current
		row.CanCreateCBT = canCreate
		row.Reason = reason
		row.Message = message
	}
	return result, nil
}

func (s *CBTAccessService) GrantAccess(ctx context.Context, actorID, actorEmail, userID string, input domain.CBTAccessGrantRequest) error {
	if input.ValidityDays < 0 || input.ValidityDays > 3650 {
		return domain.ErrInvalidInput
	}
	if _, err := s.repository.GetTeacherVerification(ctx, userID); err != nil {
		return err
	}
	if _, err := s.repository.EnsureEntitlement(ctx, userID); err != nil {
		return err
	}
	now := time.Now().UTC()
	var until *time.Time
	if input.ValidityDays > 0 {
		value := now.AddDate(0, 0, input.ValidityDays)
		until = &value
	}
	if err := s.repository.ActivateEntitlement(ctx, userID, domain.CBTAccessSourceAdminGrant, now, until); err != nil {
		return err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_access_granted", "teacher_cbt_entitlement", userID, map[string]any{
		"validity_days": input.ValidityDays, "reason": strings.TrimSpace(input.Reason),
	})
	return nil
}

func (s *CBTAccessService) RevokeAccess(ctx context.Context, actorID, actorEmail, userID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return domain.ErrInvalidInput
	}
	if _, err := s.repository.GetTeacherVerification(ctx, userID); err != nil {
		return err
	}
	if _, err := s.repository.EnsureEntitlement(ctx, userID); err != nil {
		return err
	}
	if err := s.repository.RevokeEntitlement(ctx, userID, time.Now().UTC()); err != nil {
		return err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_access_revoked", "teacher_cbt_entitlement", userID, map[string]any{"reason": reason})
	return nil
}

var _ domain.CBTAccessService = (*CBTAccessService)(nil)
