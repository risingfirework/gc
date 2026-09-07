package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"tka/apps/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type AdminService struct{ repository domain.AdminRepository }

func NewAdminService(repository domain.AdminRepository) *AdminService {
	return &AdminService{repository: repository}
}
func (s *AdminService) GetDashboard(ctx context.Context) (*domain.AdminDashboard, error) {
	return s.repository.GetDashboard(ctx)
}

func (s *AdminService) GetSiteSettings(ctx context.Context) (*domain.SiteSettings, error) {
	return s.repository.GetSiteSettings(ctx)
}

func (s *AdminService) UpdateSiteSettings(ctx context.Context, input domain.SiteSettings) (*domain.SiteSettings, error) {
	input.PlatformName = strings.TrimSpace(input.PlatformName)
	input.PlatformTagline = strings.TrimSpace(input.PlatformTagline)
	input.LogoDataURL = strings.TrimSpace(input.LogoDataURL)
	input.SupportEmail = strings.TrimSpace(input.SupportEmail)
	input.WhatsApp = strings.TrimSpace(input.WhatsApp)
	input.InstagramURL = strings.TrimSpace(input.InstagramURL)
	input.YouTubeURL = strings.TrimSpace(input.YouTubeURL)
	validLogo := input.LogoDataURL == "" || strings.HasPrefix(input.LogoDataURL, "data:image/png;base64,") || strings.HasPrefix(input.LogoDataURL, "data:image/jpeg;base64,") || strings.HasPrefix(input.LogoDataURL, "data:image/webp;base64,")
	if input.PlatformName == "" || len(input.PlatformName) > 80 || len(input.PlatformTagline) > 180 || !validLogo || len(input.LogoDataURL) > 1400000 || len(input.SupportEmail) > 160 || len(input.WhatsApp) > 32 || len(input.InstagramURL) > 300 || len(input.YouTubeURL) > 300 || input.HeroIntervalMS < 2000 || input.HeroIntervalMS > 30000 || input.CatalogIntervalMS < 2000 || input.CatalogIntervalMS > 30000 || input.DefaultPackageValidityDays < 1 || input.DefaultPackageValidityDays > 3650 || input.DefaultExamDurationMinutes < 1 || input.DefaultExamDurationMinutes > 1440 || input.DefaultExamTotalQuestions < 1 || input.DefaultExamTotalQuestions > 1000 || !validPercent(input.DefaultPassingScore) || len(input.HeroSlides) < 1 || len(input.HeroSlides) > 10 {
		return nil, domain.ErrInvalidInput
	}
	for index := range input.HeroSlides {
		slide := &input.HeroSlides[index]
		slide.ID = strings.TrimSpace(slide.ID)
		slide.Eyebrow = strings.TrimSpace(slide.Eyebrow)
		slide.Title = strings.TrimSpace(slide.Title)
		slide.Lead = strings.TrimSpace(slide.Lead)
		if slide.ID == "" || slide.Title == "" || len(slide.ID) > 60 || len(slide.Eyebrow) > 100 || len(slide.Title) > 180 || len(slide.Lead) > 500 || len(slide.Stats) > 4 {
			return nil, domain.ErrInvalidInput
		}
		for _, stat := range slide.Stats {
			if len(stat) != 2 || strings.TrimSpace(stat[0]) == "" || strings.TrimSpace(stat[1]) == "" || len(stat[0]) > 30 || len(stat[1]) > 60 {
				return nil, domain.ErrInvalidInput
			}
		}
	}
	return s.repository.UpdateSiteSettings(ctx, input)
}

func validPercent(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}

func (s *AdminService) GetFinanceDashboard(ctx context.Context) (*domain.FinanceDashboard, error) {
	return s.repository.GetFinanceDashboard(ctx)
}

func (s *AdminService) UpdateFinanceSettings(ctx context.Context, input domain.FinanceSettings) (*domain.FinanceSettings, error) {
	input.PayoutCycle = strings.ToLower(strings.TrimSpace(input.PayoutCycle))
	if !validPercent(input.PlatformCommissionPercent) || !validPercent(input.DefaultDiscountPercent) || !validPercent(input.TaxPercent) || !validPercent(input.TeacherSalesBonusPercent) || input.TeacherUploadFee < 0 || math.IsNaN(input.TeacherUploadFee) || math.IsInf(input.TeacherUploadFee, 0) || input.CommissionHoldDays < 0 || input.CommissionHoldDays > 90 || input.PlatformCommissionPercent+input.TaxPercent > 100 || input.MinimumPayout < 0 || math.IsNaN(input.MinimumPayout) || math.IsInf(input.MinimumPayout, 0) || (input.PayoutCycle != "weekly" && input.PayoutCycle != "monthly" && input.PayoutCycle != "manual") {
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
	return s.repository.UpdateFinanceSettings(ctx, input)
}

func (s *AdminService) PayTeacherCommissions(ctx context.Context, teacherID, reference string) (*domain.TeacherPayout, error) {
	reference = strings.TrimSpace(reference)
	if !validUUID(teacherID) || reference == "" || len(reference) > 120 {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.PayTeacherCommissions(ctx, teacherID, reference)
}

func (s *AdminService) ReviewPayoutRequest(ctx context.Context, requestID string, input domain.PayoutRequestReview) (*domain.TeacherWithdrawalRequest, error) {
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Note = strings.TrimSpace(input.Note)
	input.Reference = strings.TrimSpace(input.Reference)
	if !validUUID(requestID) || (input.Status != "approved" && input.Status != "rejected" && input.Status != "paid") || len(input.Note) > 500 || len(input.Reference) > 120 || (input.Status == "rejected" && input.Note == "") || (input.Status == "paid" && input.Reference == "") {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ReviewPayoutRequest(ctx, requestID, input)
}

func (s *AdminService) UpdateUserFinance(ctx context.Context, userID string, input domain.UserFinanceUpdateRequest) (*domain.UserFinanceProfile, error) {
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
	return s.repository.UpdateUserFinance(ctx, userID, input)
}
func (s *AdminService) CreateUser(ctx context.Context, input domain.AdminCreateUserRequest) (*domain.UserResponse, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	if err = validatePassword(input.Password); err != nil {
		return nil, err
	}
	role := strings.ToLower(strings.TrimSpace(input.Role))
	level := strings.ToUpper(strings.TrimSpace(input.SchoolLevel))
	if (role != domain.RoleStudent && role != domain.RoleAdmin && role != domain.RoleTeacher) || (level != "SD" && level != "SMP" && level != "SMA") {
		return nil, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash admin-created password: %w", err)
	}
	return s.repository.CreateUser(ctx, domain.User{Email: email, PasswordHash: string(hash), Role: role, SchoolLevel: level})
}
func (s *AdminService) UpdateUser(ctx context.Context, actorID, userID string, input domain.AdminUpdateUserRequest) (*domain.UserResponse, error) {
	role := strings.ToLower(strings.TrimSpace(input.Role))
	level := strings.ToUpper(strings.TrimSpace(input.SchoolLevel))
	if !validUUID(userID) || (role != domain.RoleStudent && role != domain.RoleAdmin && role != domain.RoleTeacher) || (level != "SD" && level != "SMP" && level != "SMA") {
		return nil, domain.ErrInvalidInput
	}
	if actorID == userID && role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: admin cannot remove own role", domain.ErrInvalidInput)
	}
	return s.repository.UpdateUser(ctx, userID, role, level)
}

func (s *AdminService) DeleteUser(ctx context.Context, actorID, userID string) error {
	if !validUUID(userID) {
		return domain.ErrInvalidInput
	}
	if actorID == userID {
		return fmt.Errorf("%w: admin cannot delete own account", domain.ErrInvalidInput)
	}
	return s.repository.DeleteUser(ctx, userID)
}
func validateAdminPackage(input domain.AdminPackageRequest) error {
	if strings.TrimSpace(input.Title) == "" || len(input.Title) > 200 ||
		strings.TrimSpace(input.Kode) == "" || len(input.Kode) > 50 ||
		input.Price < 0 || input.ValidityDays < 1 ||
		!validAdminStatus(input.Status) || !validJenjang(input.Jenjang) {
		return domain.ErrInvalidInput
	}
	return nil
}
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
	if err := validateAdminPackage(input); err != nil {
		return nil, err
	}
	return s.repository.CreatePackage(ctx, input)
}
func (s *AdminService) UpdatePackage(ctx context.Context, id string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Kode = strings.ToUpper(strings.TrimSpace(input.Kode))
	input.Status = normalizeAdminStatus(input.Status)
	if !validUUID(id) || validateAdminPackage(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdatePackage(ctx, id, input)
}
func (s *AdminService) CreatePackageBundle(ctx context.Context, input domain.PackageBundle) (*domain.AdminPackage, error) {
	input.Package.Title = strings.TrimSpace(input.Package.Title)
	input.Package.Description = strings.TrimSpace(input.Package.Description)
	input.Package.Kode = strings.ToUpper(strings.TrimSpace(input.Package.Kode))
	input.Package.Status = normalizeAdminStatus(input.Package.Status)
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
func validateAdminQuestion(input domain.AdminQuestionRequest) error {
	if !validUUID(input.ExamID) {
		return domain.ErrInvalidInput
	}
	return validateQuestionFields(input)
}
func validateQuestionFields(input domain.AdminQuestionRequest) error {
	if strings.TrimSpace(input.SubjectName) == "" || (strings.TrimSpace(input.ContentText) == "" && input.QuestionImageURL == "") || input.ScoreWeight <= 0 || len(input.Options) < 2 || !validAdminStatus(input.Status) || !validQuestionImage(input.QuestionImageURL) || !validQuestionImage(input.StimulusImageURL) {
		return domain.ErrInvalidInput
	}
	if input.PresentationType != domain.PresentationTypeSingle {
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
