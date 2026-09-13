// Package service berisi implementasi logika bisnis aplikasi.
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"tka/apps/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	repository domain.AdminRepository
	simpkb     *SIMPKBClient
}

func NewAdminService(repository domain.AdminRepository, simpkb ...*SIMPKBClient) *AdminService {
	service := &AdminService{repository: repository}
	if len(simpkb) > 0 {
		service.simpkb = simpkb[0]
	}
	return service
}
func (s *AdminService) GetDashboard(ctx context.Context) (*domain.AdminDashboard, error) {
	return s.repository.GetDashboard(ctx)
}

func (s *AdminService) GetSiteSettings(ctx context.Context) (*domain.SiteSettings, error) {
	return s.repository.GetSiteSettings(ctx)
}

func (s *AdminService) UpdateSiteSettings(ctx context.Context, actorID, actorEmail string, input domain.SiteSettings) (*domain.SiteSettings, error) {
	input.PlatformName = strings.TrimSpace(input.PlatformName)
	input.PlatformTagline = strings.TrimSpace(input.PlatformTagline)
	input.LogoDataURL = strings.TrimSpace(input.LogoDataURL)
	input.FaviconDataURL = strings.TrimSpace(input.FaviconDataURL)
	input.SupportEmail = strings.TrimSpace(input.SupportEmail)
	input.WhatsApp = strings.TrimSpace(input.WhatsApp)
	input.InstagramURL = strings.TrimSpace(input.InstagramURL)
	input.YouTubeURL = strings.TrimSpace(input.YouTubeURL)
	validLogo := input.LogoDataURL == "" || strings.HasPrefix(input.LogoDataURL, "data:image/png;base64,") || strings.HasPrefix(input.LogoDataURL, "data:image/jpeg;base64,") || strings.HasPrefix(input.LogoDataURL, "data:image/webp;base64,")
	validFavicon := input.FaviconDataURL == "" || strings.HasPrefix(input.FaviconDataURL, "data:image/png;base64,") || strings.HasPrefix(input.FaviconDataURL, "data:image/jpeg;base64,") || strings.HasPrefix(input.FaviconDataURL, "data:image/webp;base64,") || strings.HasPrefix(input.FaviconDataURL, "data:image/svg+xml;base64,") || strings.HasPrefix(input.FaviconDataURL, "data:image/svg+xml,")
	if input.PlatformName == "" || len(input.PlatformName) > 80 || len(input.PlatformTagline) > 180 || !validLogo || len(input.LogoDataURL) > 1400000 || !validFavicon || len(input.FaviconDataURL) > 1000000 || len(input.SupportEmail) > 160 || len(input.WhatsApp) > 32 || len(input.InstagramURL) > 300 || len(input.YouTubeURL) > 300 || input.HeroIntervalMS < 2000 || input.HeroIntervalMS > 30000 || input.CatalogIntervalMS < 2000 || input.CatalogIntervalMS > 30000 || input.DefaultPackageValidityDays < 1 || input.DefaultPackageValidityDays > 3650 || input.DefaultExamDurationMinutes < 1 || input.DefaultExamDurationMinutes > 1440 || input.DefaultExamTotalQuestions < 1 || input.DefaultExamTotalQuestions > 1000 || !validPercent(input.DefaultPassingScore) || len(input.HeroSlides) < 1 || len(input.HeroSlides) > 10 {
		return nil, domain.ErrInvalidInput
	}
	for index := range input.HeroSlides {
		slide := &input.HeroSlides[index]
		slide.ID = strings.TrimSpace(slide.ID)
		slide.Title = strings.TrimSpace(slide.Title)
		validSlideImage := slide.ImageDataURL == "" || strings.HasPrefix(slide.ImageDataURL, "data:image/png;base64,") || strings.HasPrefix(slide.ImageDataURL, "data:image/jpeg;base64,") || strings.HasPrefix(slide.ImageDataURL, "data:image/webp;base64,")
		if slide.ID == "" || len(slide.ID) > 60 || !validSlideImage || len(slide.ImageDataURL) > 4200000 || len(slide.Title) > 500 {
			return nil, domain.ErrInvalidInput
		}
	}
	result, err := s.repository.UpdateSiteSettings(ctx, input)
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "update_site_settings", "site", "settings", map[string]any{"platform_name": result.PlatformName})
	}
	return result, err
}

func (s *AdminService) ListTransactions(ctx context.Context, page, perPage int, status string) (domain.Page[domain.AdminTransaction], error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "", "pending", "paid", "failed", "expired", "refunded":
	default:
		return domain.Page[domain.AdminTransaction]{}, domain.ErrInvalidInput
	}
	return s.repository.ListTransactions(ctx, page, perPage, status)
}

func (s *AdminService) ListUsers(ctx context.Context, page, perPage int, role, level, q string) (*domain.AdminUsersPage, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	level = strings.ToUpper(strings.TrimSpace(level))
	q = strings.TrimSpace(q)
	if (role != "" && !validStaffRole(role)) || (level != "" && level != "SD" && level != "SMP" && level != "SMA") || len(q) > 120 {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ListUsers(ctx, page, perPage, role, level, q)
}

func (s *AdminService) ListAdminPackages(ctx context.Context, page, perPage int, status, jenjang, q, examType string) (*domain.AdminPackagesPage, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	jenjang = strings.ToUpper(strings.TrimSpace(jenjang))
	q = strings.TrimSpace(q)
	examType = strings.ToLower(strings.TrimSpace(examType))
	if (status != "" && !validAdminStatus(status)) || (jenjang != "" && !validJenjang(jenjang)) || (examType != "" && !validExamType(examType)) || len(q) > 120 {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ListAdminPackages(ctx, page, perPage, status, jenjang, q, examType)
}

func (s *AdminService) ListAdminExams(ctx context.Context, page, perPage int, packageID string) (domain.Page[domain.AdminExam], error) {
	packageID = strings.TrimSpace(packageID)
	if packageID != "" && !validUUID(packageID) {
		return domain.Page[domain.AdminExam]{}, domain.ErrInvalidInput
	}
	return s.repository.ListAdminExams(ctx, page, perPage, packageID)
}

func (s *AdminService) ListAdminQuestions(ctx context.Context, page, perPage int, packageID string) (domain.Page[domain.AdminQuestion], error) {
	packageID = strings.TrimSpace(packageID)
	if packageID != "" && !validUUID(packageID) {
		return domain.Page[domain.AdminQuestion]{}, domain.ErrInvalidInput
	}
	return s.repository.ListAdminQuestions(ctx, page, perPage, packageID)
}

func (s *AdminService) RefundTransaction(ctx context.Context, actorID, actorEmail, transactionID string, input domain.RefundTransactionRequest) (*domain.AdminTransaction, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !validUUID(transactionID) || input.Reason == "" || len(input.Reason) > 500 {
		return nil, domain.ErrInvalidInput
	}
	result, err := s.repository.RefundTransaction(ctx, transactionID, input.Reason, time.Now())
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "refund_transaction", "transaction", transactionID, map[string]any{
			"invoice_number": result.InvoiceNumber, "amount": result.Amount, "reason": input.Reason,
		})
	}
	return result, err
}

func (s *AdminService) ListAuditLogs(ctx context.Context) ([]domain.AuditLogEntry, error) {
	return s.repository.ListAuditLogs(ctx, 100)
}

func (s *AdminService) ListTeacherVerifications(ctx context.Context, page, perPage int, status string) (domain.Page[domain.TeacherVerification], error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != domain.TeacherVerificationPending && status != domain.TeacherVerificationRejected {
		return domain.Page[domain.TeacherVerification]{}, domain.ErrInvalidInput
	}
	return s.repository.ListTeacherVerifications(ctx, page, perPage, status)
}

func (s *AdminService) ApproveTeacher(ctx context.Context, actorID, actorEmail, userID string) error {
	if !validUUID(userID) {
		return domain.ErrInvalidInput
	}
	if err := s.repository.ApproveTeacher(ctx, userID); err != nil {
		return err
	}
	s.audit(ctx, actorID, actorEmail, "approve_teacher", "user", userID, map[string]any{"status": domain.TeacherVerificationApproved})
	return nil
}

func (s *AdminService) RejectTeacher(ctx context.Context, actorID, actorEmail, userID, reason string) error {
	if !validUUID(userID) {
		return domain.ErrInvalidInput
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]byte(reason)) > 500 {
		return domain.ErrInvalidInput
	}
	if err := s.repository.RejectTeacher(ctx, userID, reason); err != nil {
		return err
	}
	s.audit(ctx, actorID, actorEmail, "reject_teacher", "user", userID, map[string]any{"status": domain.TeacherVerificationRejected, "reason": reason})
	return nil
}

// CheckTeacherSIMPKB mengecek NIK seorang guru yang masih pending ke portal
// SIMPKB atas pemicu manual dari panel admin, lalu menyimpan screenshot.
// Pengecekan dijalankan di latar belakang dan endpoint langsung merespons
// dengan status "checking" agar tidak terikat batas waktu request admin (15s).
func (s *AdminService) CheckTeacherSIMPKB(ctx context.Context, userID string) error {
	if !validUUID(userID) {
		return domain.ErrInvalidInput
	}
	if s.simpkb == nil {
		return fmt.Errorf("layanan pengecekan SIMPKB belum dikonfigurasi")
	}
	ktp, err := s.repository.GetTeacherKTP(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := normalizeTeacherKTP(ktp); err != nil {
		return s.saveSIMPKBResult(userID, domain.SIMPKBStatusError, "")
	}
	if err := s.repository.SetTeacherSIMPKBChecking(ctx, userID); err != nil {
		return err
	}
	go func(uid, nik string) {
		_ = s.runSIMPKBJob(uid, nik)
	}(userID, ktp)
	return nil
}

// SIMPKBWorkerTick memproses guru pending yang belum pernah dicek ke SIMPKB.
// Dipanggil terjadwal oleh scheduler dan sekali saat aplikasi mulai.
func (s *AdminService) SIMPKBWorkerTick(ctx context.Context) error {
	if s.simpkb == nil {
		return nil
	}
	jobs, err := s.repository.ListPendingTeacherSIMPKBChecks(ctx, 20)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err := normalizeTeacherKTP(job.TeacherKTP); err != nil {
			_ = s.saveSIMPKBResult(job.UserID, domain.SIMPKBStatusError, "")
			continue
		}
		_ = s.runSIMPKBJob(job.UserID, job.TeacherKTP)
		time.Sleep(2 * time.Second)
	}
	return nil
}

// runSIMPKBJob memanggil layanan screenshot lalu menyimpan hasil. Ditulis
// menggunakan konteks tanpa pembatalan agar hasil selalu tersimpan walau
// request admin sudah melewati batas waktunya.
func (s *AdminService) runSIMPKBJob(userID, nik string) error {
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 40*time.Second)
	defer cancel()
	result, err := s.simpkb.Check(checkCtx, nik)
	if err != nil {
		return s.saveSIMPKBResult(userID, domain.SIMPKBStatusError, "")
	}
	return s.saveSIMPKBResult(userID, normalizeSIMPKBStatus(result.Status), result.Screenshot)
}

func (s *AdminService) saveSIMPKBResult(userID, status, image string) error {
	return s.repository.SaveTeacherSIMPKBResult(context.WithoutCancel(context.Background()), userID, status, image, time.Now())
}

func normalizeSIMPKBStatus(status string) string {
	switch status {
	case domain.SIMPKBStatusFound, domain.SIMPKBStatusNotFound:
		return status
	default:
		return domain.SIMPKBStatusError
	}
}

func validStaffRole(role string) bool {
	switch role {
	case domain.RoleStudent, domain.RoleTeacher, domain.RoleAdmin, domain.RoleOwner, domain.RoleFinance, domain.RoleAffiliate:
		return true
	default:
		return false
	}
}

func (s *AdminService) audit(ctx context.Context, actorID, actorEmail, action, entityType, entityID string, detail any) {
	raw, err := json.Marshal(detail)
	if err != nil {
		return
	}
	_ = s.repository.InsertAuditLog(ctx, domain.AuditLogEntry{
		ActorID: actorID, ActorEmail: actorEmail, Action: action, EntityType: entityType, EntityID: entityID, Detail: raw,
	})
}

func validPercent(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}

// newReferralCode menghasilkan kode rujukan jinak (mis. MITRA-A1B2C3D4).
func newReferralCode() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return "MITRA-" + fmt.Sprintf("%08X", time.Now().UnixNano()%100000000)
	}
	return "MITRA-" + strings.ToUpper(hex.EncodeToString(raw))
}

// ensureAffiliate menjamin user affiliate memiliki profil afiliasi berikut kodenya.
func (s *AdminService) ensureAffiliate(ctx context.Context, userID string) (string, error) {
	return s.repository.EnsureAffiliate(ctx, userID, newReferralCode())
}

func (s *AdminService) GetFinanceDashboard(ctx context.Context) (*domain.FinanceDashboard, error) {
	return s.repository.GetFinanceDashboard(ctx)
}

var financeCSVHeader = []string{"Tanggal", "Guru", "Paket", "No. Invoice", "Jenis", "Dasar", "Persen (%)", "Nominal", "Status", "Tersedia", "Dibayar", "Referensi"}
var financeKindLabel = map[string]string{"upload_fee": "Honorarium upload", "sales_bonus": "Bonus penjualan", "refund_reversal": "Koreksi refund", "referral_bonus": "Bonus rujukan", "referral_reversal": "Koreksi rujukan"}
var financeStatusLabel = map[string]string{"pending": "Ditahan", "available": "Dapat dicairkan", "paid": "Sudah dibayar", "cancelled": "Dibatalkan"}

// ExportFinanceCSV menghasilkan laporan ledger komisi dalam format CSV
// (UTF-8 dengan BOM agar terbaca rapi di Microsoft Excel).
func (s *AdminService) ExportFinanceCSV(ctx context.Context) ([]byte, error) {
	items, err := s.repository.ExportFinanceLedger(ctx)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("\uFEFF") // UTF-8 BOM
	writer := csv.NewWriter(&buf)
	if err := writer.Write(financeCSVHeader); err != nil {
		return nil, err
	}
	dateOnly := func(value time.Time) string {
		if value.IsZero() {
			return ""
		}
		return value.Format("2006-01-02")
	}
	formatMoney := func(value float64) string {
		return strings.ReplaceAll(fmt.Sprintf("%.0f", value), "-", "")
	}
	for _, item := range items {
		status := item.Status
		if label, ok := financeStatusLabel[status]; ok {
			status = label
		}
		kind := item.Kind
		if label, ok := financeKindLabel[kind]; ok {
			kind = label
		}
		paidAt := ""
		if item.PaidAt != nil {
			paidAt = dateOnly(*item.PaidAt)
		}
		row := []string{
			dateOnly(item.CreatedAt),
			item.TeacherEmail,
			item.PackageTitle,
			item.InvoiceNumber,
			kind,
			formatMoney(item.BaseAmount),
			fmt.Sprintf("%g", item.RatePercent),
			formatMoney(item.Amount),
			status,
			dateOnly(item.AvailableAt),
			paidAt,
			item.PayoutReference,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *AdminService) UpdateFinanceSettings(ctx context.Context, actorID, actorEmail string, input domain.FinanceSettings) (*domain.FinanceSettings, error) {
	input.PayoutCycle = strings.ToLower(strings.TrimSpace(input.PayoutCycle))
	if !validPercent(input.PlatformCommissionPercent) || !validPercent(input.DefaultDiscountPercent) || !validPercent(input.TaxPercent) || !validPercent(input.TeacherSalesBonusPercent) || !validPercent(input.AffiliateRatePercent) || input.TeacherUploadFee < 0 || math.IsNaN(input.TeacherUploadFee) || math.IsInf(input.TeacherUploadFee, 0) || input.CommissionHoldDays < 0 || input.CommissionHoldDays > 90 || input.PlatformCommissionPercent+input.TaxPercent > 100 || input.MinimumPayout < 0 || math.IsNaN(input.MinimumPayout) || math.IsInf(input.MinimumPayout, 0) || (input.PayoutCycle != "weekly" && input.PayoutCycle != "monthly" && input.PayoutCycle != "manual") {
		return nil, domain.ErrInvalidInput
	}
	current, err := s.repository.GetFinanceDashboard(ctx)
	if err != nil {
		return nil, err
	}
	for _, profile := range current.Users {
		if profile.CommissionPercent != nil && *profile.CommissionPercent+input.TaxPercent > 100 {
			return nil, domain.ErrInvalidInput
		}
	}
	result, err := s.repository.UpdateFinanceSettings(ctx, input)
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "update_finance_settings", "finance_settings", "settings", map[string]any{
			"platform_commission_percent": result.PlatformCommissionPercent,
			"teacher_sales_bonus_percent": result.TeacherSalesBonusPercent,
			"affiliate_rate_percent":      result.AffiliateRatePercent,
			"tax_percent":                 result.TaxPercent,
			"minimum_payout":              result.MinimumPayout,
			"payout_cycle":                result.PayoutCycle,
			"auto_payout":                 result.AutoPayout,
		})
	}
	return result, err
}

func (s *AdminService) AutoPayTeachers(ctx context.Context) ([]domain.TeacherPayout, error) {
	ids, err := s.repository.ListPayoutEligibleTeachers(ctx)
	if err != nil {
		return nil, err
	}
	referenceBase := "AUTO"
	var out []domain.TeacherPayout
	for _, id := range ids {
		item, err := s.repository.PayTeacherCommissions(ctx, id, referenceBase+"-"+id[:8])
		if err != nil {
			continue
		}
		s.audit(ctx, "", "(sistem)", "payout_paid", "teacher_payout", item.ID, map[string]any{
			"teacher_id": id, "reference": item.Reference, "amount": item.Amount, "source": "auto",
		})
		out = append(out, *item)
	}
	return out, nil
}

func (s *AdminService) PayTeacherCommissions(ctx context.Context, actorID, actorEmail, teacherID, reference string) (*domain.TeacherPayout, error) {
	reference = strings.TrimSpace(reference)
	if !validUUID(teacherID) || reference == "" || len(reference) > 120 {
		return nil, domain.ErrInvalidInput
	}
	result, err := s.repository.PayTeacherCommissions(ctx, teacherID, reference)
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "payout_paid", "teacher_payout", result.ID, map[string]any{
			"teacher_id": teacherID, "reference": reference, "amount": result.Amount,
		})
	}
	return result, err
}

func (s *AdminService) ReviewPayoutRequest(ctx context.Context, actorID, actorEmail, requestID string, input domain.PayoutRequestReview) (*domain.TeacherWithdrawalRequest, error) {
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Note = strings.TrimSpace(input.Note)
	input.Reference = strings.TrimSpace(input.Reference)
	if !validUUID(requestID) || (input.Status != "approved" && input.Status != "rejected" && input.Status != "paid") || len(input.Note) > 500 || len(input.Reference) > 120 || (input.Status == "rejected" && input.Note == "") || (input.Status == "paid" && input.Reference == "") {
		return nil, domain.ErrInvalidInput
	}
	result, err := s.repository.ReviewPayoutRequest(ctx, requestID, input)
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "payout_review", "teacher_payout_request", result.ID, map[string]any{
			"status": input.Status, "note": input.Note, "reference": input.Reference,
		})
	}
	return result, err
}

func (s *AdminService) UpdateUserFinance(ctx context.Context, actorID, actorEmail, userID string, input domain.UserFinanceUpdateRequest) (*domain.UserFinanceProfile, error) {
	input.AccountStatus = strings.ToLower(strings.TrimSpace(input.AccountStatus))
	input.Notes = strings.TrimSpace(input.Notes)
	if !validUUID(userID) || (input.CommissionPercent != nil && !validPercent(*input.CommissionPercent)) || (input.DiscountPercent != nil && !validPercent(*input.DiscountPercent)) || (input.AccountStatus != "active" && input.AccountStatus != "hold") || len(input.Notes) > 500 {
		return nil, domain.ErrInvalidInput
	}
	if input.CommissionPercent != nil {
		current, err := s.repository.GetFinanceDashboard(ctx)
		if err != nil {
			return nil, err
		}
		if *input.CommissionPercent+current.Settings.TaxPercent > 100 {
			return nil, domain.ErrInvalidInput
		}
	}
	result, err := s.repository.UpdateUserFinance(ctx, userID, input)
	if err == nil {
		s.audit(ctx, actorID, actorEmail, "update_user_finance", "user_finance", userID, map[string]any{
			"commission_percent": input.CommissionPercent, "discount_percent": input.DiscountPercent,
			"account_status": input.AccountStatus, "notes": input.Notes,
		})
	}
	return result, err
}
func (s *AdminService) CreateUser(ctx context.Context, actorID, actorEmail string, input domain.AdminCreateUserRequest) (*domain.UserResponse, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	if err = validatePassword(input.Password); err != nil {
		return nil, err
	}
	role := strings.ToLower(strings.TrimSpace(input.Role))
	level := strings.ToUpper(strings.TrimSpace(input.SchoolLevel))
	if !validStaffRole(role) || (level != "SD" && level != "SMP" && level != "SMA") {
		return nil, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash admin-created password: %w", err)
	}
	item, err := s.repository.CreateUser(ctx, domain.User{Email: email, PasswordHash: string(hash), Role: role, SchoolLevel: level})
	if err == nil {
		if role == domain.RoleAffiliate {
			if code, codeErr := s.ensureAffiliate(ctx, item.ID); codeErr == nil {
				item.ReferralCode = code
			}
		}
		s.audit(ctx, actorID, actorEmail, "create_user", "user", item.ID, map[string]any{"email": item.Email, "role": role, "school_level": level, "referral_code": item.ReferralCode})
	}
	return item, err
}
func (s *AdminService) UpdateUser(ctx context.Context, actorID, actorEmail, userID string, input domain.AdminUpdateUserRequest) (*domain.UserResponse, error) {
	role := strings.ToLower(strings.TrimSpace(input.Role))
	level := strings.ToUpper(strings.TrimSpace(input.SchoolLevel))
	if !validUUID(userID) || !validStaffRole(role) || (level != "SD" && level != "SMP" && level != "SMA") {
		return nil, domain.ErrInvalidInput
	}
	current, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == domain.RoleOwner && role != domain.RoleOwner {
		if actorID == userID {
			return nil, fmt.Errorf("%w: owner cannot remove own role", domain.ErrInvalidInput)
		}
		owners, err := s.repository.CountOwners(ctx)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, domain.ErrLastOwnerGuard
		}
	}
	result, err := s.repository.UpdateUser(ctx, userID, role, level)
	if err == nil {
		if role == domain.RoleAffiliate {
			if code, codeErr := s.ensureAffiliate(ctx, userID); codeErr == nil {
				result.ReferralCode = code
			}
		}
		s.audit(ctx, actorID, actorEmail, "update_user", "user", userID, map[string]any{"from_role": current, "to_role": role, "school_level": level, "referral_code": result.ReferralCode})
	}
	return result, err
}

func (s *AdminService) DeleteUser(ctx context.Context, actorID, actorEmail, userID string) error {
	if !validUUID(userID) {
		return domain.ErrInvalidInput
	}
	if actorID == userID {
		return fmt.Errorf("%w: admin cannot delete own account", domain.ErrInvalidInput)
	}
	current, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}
	if current == domain.RoleOwner {
		owners, err := s.repository.CountOwners(ctx)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return domain.ErrLastOwnerGuard
		}
	}
	if err := s.repository.DeleteUser(ctx, userID); err != nil {
		return err
	}
	s.audit(ctx, actorID, actorEmail, "delete_user", "user", userID, map[string]any{"role": current})
	return nil
}
func validateAdminPackage(input domain.AdminPackageRequest) error {
	if strings.TrimSpace(input.Title) == "" || len(input.Title) > 200 ||
		input.Price < 0 || input.ValidityDays < 1 ||
		!validAdminStatus(input.Status) || !validJenjang(input.Jenjang) ||
		!validExamType(input.ExamType) || !validUUID(input.KategoriID) || !validUUID(input.KelasID) ||
		(input.ExamType == "sell" && (strings.TrimSpace(input.Kode) == "" || len(input.Kode) > 50)) ||
		(input.ExamType == "cbt" && !cbtTokenPattern.MatchString(input.CBTToken)) ||
		(input.ExamType != "cbt" && input.CBTToken != "") {
		return domain.ErrInvalidInput
	}
	if input.ExamType == "cbt" {
		if input.StartDate != "" || input.EndDate != "" {
			if _, err := time.Parse(time.RFC3339, input.StartDate); err != nil {
				return domain.ErrInvalidInput
			}
			if _, err := time.Parse(time.RFC3339, input.EndDate); err != nil {
				return domain.ErrInvalidInput
			}
			start, _ := time.Parse(time.RFC3339, input.StartDate)
			end, _ := time.Parse(time.RFC3339, input.EndDate)
			if !start.Before(end) {
				return domain.ErrInvalidInput
			}
		}
	}
	return nil
}
func applyPackagePolicies(input *domain.AdminPackageRequest) {
	input.ExamType = strings.ToLower(strings.TrimSpace(input.ExamType))
	input.CBTToken = strings.ToUpper(strings.TrimSpace(input.CBTToken))
	input.StartDate = strings.TrimSpace(input.StartDate)
	input.EndDate = strings.TrimSpace(input.EndDate)
	if input.ExamType == "cbt" {
		input.Price = 0
		input.ValidityDays = computeValidityDays(input.StartDate, input.EndDate)
		if input.ValidityDays < 1 {
			input.ValidityDays = 1
		}
	} else {
		input.CBTToken = ""
		input.StartDate = ""
		input.EndDate = ""
	}
}

func computeValidityDays(startStr, endStr string) int {
	start, err1 := time.Parse(time.RFC3339, startStr)
	end, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return 1
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	return days
}
func validExamType(value string) bool {
	return value == "sell" || value == "cbt"
}

var cbtTokenPattern = regexp.MustCompile(`^[A-Z0-9]{4,8}$`)

func validJenjang(value string) bool {
	switch value {
	case "SD", "SMP", "SMA":
		return true
	default:
		return false
	}
}
func (s *AdminService) CreatePackage(ctx context.Context, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Kode = strings.ToUpper(strings.TrimSpace(input.Kode))
	input.Status = normalizeAdminStatus(input.Status)
	applyPackagePolicies(&input)
	if err := validateAdminPackage(input); err != nil {
		return nil, err
	}
	return s.repository.CreatePackage(ctx, input)
}
func (s *AdminService) UpdatePackage(ctx context.Context, id string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Kode = strings.ToUpper(strings.TrimSpace(input.Kode))
	input.ExamType = strings.ToLower(strings.TrimSpace(input.ExamType))
	if !validUUID(id) {
		return nil, domain.ErrInvalidInput
	}
	if input.ExamType == "cbt" && input.StartDate == "" && input.EndDate == "" {
		existing, err := s.repository.GetPackage(ctx, id)
		if err != nil {
			return nil, err
		}
		input.StartDate = strings.TrimSpace(existing.StartDate)
		input.EndDate = strings.TrimSpace(existing.EndDate)
	}
	input.Status = normalizeAdminStatus(input.Status)
	applyPackagePolicies(&input)
	if validateAdminPackage(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdatePackage(ctx, id, input)
}
func (s *AdminService) CreatePackageBundle(ctx context.Context, input domain.PackageBundle) (*domain.AdminPackage, error) {
	input.Package.Title = strings.TrimSpace(input.Package.Title)
	input.Package.Description = strings.TrimSpace(input.Package.Description)
	input.Package.Kode = strings.ToUpper(strings.TrimSpace(input.Package.Kode))
	input.Package.Status = normalizeAdminStatus(input.Package.Status)
	applyPackagePolicies(&input.Package)
	if input.Exam != nil {
		input.Exam.Title = strings.TrimSpace(input.Exam.Title)
		input.Exam.Status = normalizeAdminStatus(input.Exam.Status)
	}
	for index := range input.Questions {
		input.Questions[index] = normalizeQuestion(input.Questions[index])
		input.Questions[index].Status = normalizeAdminStatus(input.Questions[index].Status)
	}
	if validatePackageBundle(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreatePackageBundle(ctx, input)
}
func (s *AdminService) DeletePackage(ctx context.Context, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeletePackage(ctx, id)
}
func validateAdminExam(input domain.AdminExamRequest) error {
	if !validUUID(input.PackageID) || strings.TrimSpace(input.Title) == "" || len(input.Title) > 200 || input.DurationMinutes < 1 || input.TotalQuestions < 1 || input.PassingScore < 0 || input.PassingScore > 100 || !validAdminStatus(input.Status) {
		return domain.ErrInvalidInput
	}
	return nil
}
func (s *AdminService) CreateExam(ctx context.Context, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Status = normalizeAdminStatus(input.Status)
	if err := validateAdminExam(input); err != nil {
		return nil, err
	}
	return s.repository.CreateExam(ctx, input)
}
func (s *AdminService) UpdateExam(ctx context.Context, id string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Status = normalizeAdminStatus(input.Status)
	if !validUUID(id) || validateAdminExam(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateExam(ctx, id, input)
}
func (s *AdminService) DeleteExam(ctx context.Context, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeleteExam(ctx, id)
}
func (s *AdminService) ListCBTPublishSettings(ctx context.Context) ([]domain.CBTPublishSetting, error) {
	return s.repository.ListCBTPublishSettings(ctx)
}
func (s *AdminService) SetExamPublishPembahasan(ctx context.Context, actorID, actorEmail, examID string, publish bool) (*domain.CBTPublishSetting, error) {
	if !validUUID(examID) {
		return nil, domain.ErrInvalidInput
	}
	item, err := s.repository.SetExamPublishPembahasan(ctx, examID, publish)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, actorEmail, "cbt_publish_pembahasan", "exam", examID, map[string]any{"publish_pembahasan": publish})
	return item, nil
}
func (s *AdminService) ListCBTParticipants(ctx context.Context, examID string) ([]domain.CBTParticipant, error) {
	if !validUUID(examID) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ListCBTParticipants(ctx, examID)
}
func validateAdminQuestion(input domain.AdminQuestionRequest) error {
	if !validUUID(input.ExamID) {
		return domain.ErrInvalidInput
	}
	return validateQuestionFields(input)
}
func validateQuestionFields(input domain.AdminQuestionRequest) error {
	isEssay := input.QuestionType == domain.QuestionTypeEssay
	if strings.TrimSpace(input.SubjectName) == "" || (strings.TrimSpace(input.ContentText) == "" && input.QuestionImageURL == "") || input.ScoreWeight <= 0 || !validAdminStatus(input.Status) || !validQuestionImage(input.QuestionImageURL) || !validQuestionImage(input.StimulusImageURL) {
		return domain.ErrInvalidInput
	}
	if input.PresentationType != domain.PresentationTypeSingle {
		return domain.ErrInvalidInput
	}
	if !isEssay {
		if len(input.Options) < 2 {
			return domain.ErrInvalidInput
		}
		seenKeys := map[string]bool{}
		for _, option := range input.Options {
			key := strings.TrimSpace(option.Key)
			if key == "" || (strings.TrimSpace(option.Content) == "" && option.ImageURL == "") || seenKeys[key] || !validQuestionImage(option.ImageURL) {
				return domain.ErrInvalidInput
			}
			seenKeys[key] = true
		}
	}
	if _, valid := canonicalAnswer(input.QuestionType, input.Options, input.CategoryLabels, input.CorrectAnswer, false); !valid {
		return domain.ErrInvalidInput
	}
	return nil
}
func validatePackageBundle(input domain.PackageBundle) error {
	if validateAdminPackage(input.Package) != nil {
		return domain.ErrInvalidInput
	}
	if input.Exam == nil {
		if len(input.Questions) > 0 {
			return domain.ErrInvalidInput
		}
		return nil
	}
	exam := input.Exam
	if strings.TrimSpace(exam.Title) == "" || len(exam.Title) > 200 || exam.DurationMinutes < 1 || exam.TotalQuestions < 1 || exam.PassingScore < 0 || exam.PassingScore > 100 || !validAdminStatus(exam.Status) {
		return domain.ErrInvalidInput
	}
	for _, question := range input.Questions {
		if validateQuestionFields(question) != nil {
			return domain.ErrInvalidInput
		}
	}
	return nil
}
func validAdminStatus(value string) bool {
	return value == domain.StatusActive || value == domain.StatusInactive
}
func normalizeAdminStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return domain.StatusActive
	}
	return value
}
func (s *AdminService) CreateQuestion(ctx context.Context, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	input = normalizeQuestion(input)
	input.Status = normalizeAdminStatus(input.Status)
	if err := validateAdminQuestion(input); err != nil {
		return nil, err
	}
	return s.repository.CreateQuestion(ctx, input)
}
func (s *AdminService) UpdateQuestion(ctx context.Context, id string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	input = normalizeQuestion(input)
	input.Status = normalizeAdminStatus(input.Status)
	if !validUUID(id) || validateAdminQuestion(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateQuestion(ctx, id, input)
}
func (s *AdminService) DeleteQuestion(ctx context.Context, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeleteQuestion(ctx, id)
}
func (s *AdminService) BulkDeleteQuestions(ctx context.Context, ids []string, packageID string) (int, error) {
	if len(ids) == 0 && !validUUID(packageID) && packageID != "" {
		return 0, domain.ErrInvalidInput
	}
	for _, id := range ids {
		if !validUUID(id) {
			return 0, domain.ErrInvalidInput
		}
	}
	if len(ids) == 0 && packageID == "" {
		return 0, domain.ErrInvalidInput
	}
	return s.repository.BulkDeleteQuestions(ctx, ids, packageID)
}

func validateMasterName(nama string) bool {
	nama = strings.TrimSpace(nama)
	return nama != "" && len(nama) <= 120
}

func (s *AdminService) ListMaster(ctx context.Context, category domain.MasterCategory) ([]domain.MasterItem, error) {
	return s.repository.ListMaster(ctx, category)
}

func (s *AdminService) CreateMaster(ctx context.Context, category domain.MasterCategory, nama string) (*domain.MasterItem, error) {
	nama = strings.TrimSpace(nama)
	if !validateMasterName(nama) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreateMaster(ctx, category, nama)
}

func (s *AdminService) UpdateMaster(ctx context.Context, category domain.MasterCategory, id, nama string) (*domain.MasterItem, error) {
	nama = strings.TrimSpace(nama)
	if !validUUID(id) || !validateMasterName(nama) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateMaster(ctx, category, id, nama)
}

func (s *AdminService) DeleteMaster(ctx context.Context, category domain.MasterCategory, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeleteMaster(ctx, category, id)
}

var _ domain.AdminService = (*AdminService)(nil)
