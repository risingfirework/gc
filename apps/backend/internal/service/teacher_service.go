package service

import (
	"context"
	"regexp"
	"strings"

	"tka/apps/backend/internal/domain"
)

var payoutNumberPattern = regexp.MustCompile(`^[0-9]{6,30}$`)
var payoutPhonePattern = regexp.MustCompile(`^(?:\+62|62|0)[0-9]{8,13}$`)

type TeacherService struct{ repository domain.TeacherRepository }

func NewTeacherService(repository domain.TeacherRepository) *TeacherService {
	return &TeacherService{repository: repository}
}
func (s *TeacherService) GetDashboard(ctx context.Context, publisherID string) (*domain.TeacherDashboard, error) {
	return s.repository.GetDashboard(ctx, publisherID)
}
func (s *TeacherService) CreatePackage(ctx context.Context, publisherID string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Kode = strings.ToUpper(strings.TrimSpace(input.Kode))
	input.Status = domain.StatusInactive
	applyPackagePolicies(&input)
	if input.ExamType == "cbt" {
		input.Status = domain.StatusActive
	}
	if validateAdminPackage(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreatePackage(ctx, publisherID, input)
}
func (s *TeacherService) UpdatePackage(ctx context.Context, publisherID, id string, input domain.AdminPackageRequest) (*domain.AdminPackage, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Kode = strings.ToUpper(strings.TrimSpace(input.Kode))
	input.ExamType = strings.ToLower(strings.TrimSpace(input.ExamType))
	if !validUUID(id) {
		return nil, domain.ErrInvalidInput
	}
	if input.ExamType == "cbt" && input.StartDate == "" && input.EndDate == "" {
		existing, err := s.repository.GetPackage(ctx, publisherID, id)
		if err != nil {
			return nil, err
		}
		input.StartDate = strings.TrimSpace(existing.StartDate)
		input.EndDate = strings.TrimSpace(existing.EndDate)
	}
	input.Status = domain.StatusInactive
	applyPackagePolicies(&input)
	if input.ExamType == "cbt" {
		input.Status = domain.StatusActive
	}
	if validateAdminPackage(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdatePackage(ctx, publisherID, id, input)
}
func (s *TeacherService) CreatePackageBundle(ctx context.Context, publisherID string, input domain.PackageBundle) (*domain.AdminPackage, error) {
	input.Package.Title = strings.TrimSpace(input.Package.Title)
	input.Package.Description = strings.TrimSpace(input.Package.Description)
	input.Package.Kode = strings.ToUpper(strings.TrimSpace(input.Package.Kode))
	input.Package.Status = domain.StatusInactive
	applyPackagePolicies(&input.Package)
	if input.Package.ExamType == "cbt" {
		input.Package.Status = domain.StatusActive
	}
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
	return s.repository.CreatePackageBundle(ctx, publisherID, input)
}
func (s *TeacherService) DeletePackage(ctx context.Context, publisherID, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeletePackage(ctx, publisherID, id)
}
func (s *TeacherService) CreateExam(ctx context.Context, publisherID string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Status = normalizeAdminStatus(input.Status)
	if validateAdminExam(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreateExam(ctx, publisherID, input)
}
func (s *TeacherService) UpdateExam(ctx context.Context, publisherID, id string, input domain.AdminExamRequest) (*domain.AdminExam, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Status = normalizeAdminStatus(input.Status)
	if !validUUID(id) || validateAdminExam(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateExam(ctx, publisherID, id, input)
}
func (s *TeacherService) DeleteExam(ctx context.Context, publisherID, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeleteExam(ctx, publisherID, id)
}
func (s *TeacherService) ListCBTPublishSettings(ctx context.Context, publisherID string) ([]domain.CBTPublishSetting, error) {
	if !validUUID(publisherID) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ListCBTPublishSettings(ctx, publisherID)
}
func (s *TeacherService) SetExamPublishPembahasan(ctx context.Context, publisherID, examID string, publish bool) (*domain.CBTPublishSetting, error) {
	if !validUUID(publisherID) || !validUUID(examID) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.SetExamPublishPembahasan(ctx, publisherID, examID, publish)
}
func (s *TeacherService) ListCBTParticipants(ctx context.Context, publisherID, examID string) ([]domain.CBTParticipant, error) {
	if !validUUID(publisherID) || !validUUID(examID) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.ListCBTParticipants(ctx, publisherID, examID)
}
func (s *TeacherService) CreateQuestion(ctx context.Context, publisherID string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	input = normalizeQuestion(input)
	input.Status = normalizeAdminStatus(input.Status)
	if validateAdminQuestion(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreateQuestion(ctx, publisherID, input)
}
func (s *TeacherService) UpdateQuestion(ctx context.Context, publisherID, id string, input domain.AdminQuestionRequest) (*domain.AdminQuestion, error) {
	input = normalizeQuestion(input)
	input.Status = normalizeAdminStatus(input.Status)
	if !validUUID(id) || validateAdminQuestion(input) != nil {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateQuestion(ctx, publisherID, id, input)
}
func (s *TeacherService) DeleteQuestion(ctx context.Context, publisherID, id string) error {
	if !validUUID(id) {
		return domain.ErrInvalidInput
	}
	return s.repository.DeleteQuestion(ctx, publisherID, id)
}
func (s *TeacherService) BulkDeleteQuestions(ctx context.Context, publisherID string, ids []string, packageID string) (int, error) {
	for _, id := range ids {
		if !validUUID(id) {
			return 0, domain.ErrInvalidInput
		}
	}
	if len(ids) == 0 && (!validUUID(packageID) || packageID == "") {
		return 0, domain.ErrInvalidInput
	}
	return s.repository.BulkDeleteQuestions(ctx, publisherID, ids, packageID)
}

func (s *TeacherService) UpdatePayoutAccount(ctx context.Context, publisherID string, input domain.TeacherPayoutAccount) (*domain.TeacherPayoutAccount, error) {
	input.Method = strings.ToLower(strings.TrimSpace(input.Method))
	input.Provider = strings.TrimSpace(input.Provider)
	input.AccountNumber = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(input.AccountNumber), " ", ""), "-", "")
	input.AccountHolderName = strings.Join(strings.Fields(input.AccountHolderName), " ")
	input.Phone = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(input.Phone), " ", ""), "-", "")
	if (input.Method != "bank_transfer" && input.Method != "e_wallet") || input.Provider == "" || len(input.Provider) > 80 || !payoutNumberPattern.MatchString(input.AccountNumber) || input.AccountHolderName == "" || len(input.AccountHolderName) > 120 || (input.Phone != "" && !payoutPhonePattern.MatchString(input.Phone)) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdatePayoutAccount(ctx, publisherID, input)
}

func (s *TeacherService) CreatePayoutRequest(ctx context.Context, publisherID string, amount float64) (*domain.TeacherWithdrawalRequest, error) {
	if !validUUID(publisherID) || amount <= 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CreatePayoutRequest(ctx, publisherID, amount)
}

func (s *TeacherService) CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*domain.TeacherWithdrawalRequest, error) {
	if !validUUID(publisherID) || !validUUID(requestID) {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.CancelPayoutRequest(ctx, publisherID, requestID)
}

// Appeal memvalidasi bukti sanggah (data-URI gambar) lalu mengeksekusinya.
// Hanya guru dengan status rejected yang bisa mengajukan sanggah.
func (s *TeacherService) Appeal(ctx context.Context, userID string, input domain.TeacherAppealRequest) error {
	image := strings.TrimSpace(input.AppealImageDataURL)
	if !strings.HasPrefix(image, "data:image/") || len(image) > 6<<20 {
		return domain.ErrInvalidInput
	}
	return s.repository.Appeal(ctx, userID, image)
}

var _ domain.TeacherService = (*TeacherService)(nil)
