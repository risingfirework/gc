package domain

import (
	"context"
	"time"
)

// TeacherVerification adalah baris data guru yang sedang/belum disetujui dan
// tampil pada panel verifikasi operator/admin.
type TeacherVerification struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	TeacherKTP      string    `json:"teacher_ktp"`
	SchoolLevel     string    `json:"school_level"`
	Status          string    `json:"status"`
	RejectionReason string    `json:"rejection_reason"`
	AppealImage     string    `json:"appeal_image"`
	SIMPKBStatus    string    `json:"simpkb_status"`
	SIMPKBImage     string    `json:"simpkb_image"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TeacherSIMPKBJob adalah data minimum guru pending yang menunggu dicek ke SIMPKB.
type TeacherSIMPKBJob struct {
	UserID     string
	TeacherKTP string
}

// TeacherAppealRequest berisi bukti sanggah guru saat pendaftarannya ditolak.
// AppealImageDataURL berupa data-URI gambar (atau URL jika disediakan klien).
type TeacherAppealRequest struct {
	AppealImageDataURL string `json:"appeal_image_data_url"`
}

type TeacherOverview struct {
	TotalPackages  int64   `json:"total_packages"`
	TotalExams     int64   `json:"total_exams"`
	TotalQuestions int64   `json:"total_questions"`
	TotalSales     int64   `json:"total_sales"`
	TotalRevenue   float64 `json:"total_revenue"`
}

type TeacherDashboard struct {
	Overview          TeacherOverview            `json:"overview"`
	Levels            []string                   `json:"levels"`
	Mapels            []MasterItem               `json:"mapels"`
	AcademicYears     []MasterItem               `json:"academic_years"`
	Kategoris         []MasterItem               `json:"kategoris"`
	Kelas             []MasterItem               `json:"kelas"`
	Packages          []AdminPackage             `json:"packages"`
	Exams             []AdminExam                `json:"exams"`
	Questions         []AdminQuestion            `json:"questions"`
	Transactions      []AdminTransaction         `json:"transactions"`
	CommissionSummary TeacherCommissionSummary   `json:"commission_summary"`
	Commissions       []TeacherCommission        `json:"commissions"`
	Payouts           []TeacherPayout            `json:"payouts"`
	PayoutAccount     TeacherPayoutAccount       `json:"payout_account"`
	FinanceSettings   FinanceSettings            `json:"finance_settings"`
	PayoutRequests    []TeacherWithdrawalRequest `json:"payout_requests"`
}

type TeacherRepository interface {
	GetDashboard(ctx context.Context, publisherID string) (*TeacherDashboard, error)
	CreatePackage(ctx context.Context, publisherID string, input AdminPackageRequest) (*AdminPackage, error)
	CreatePackageBundle(ctx context.Context, publisherID string, input PackageBundle) (*AdminPackage, error)
	GetPackage(ctx context.Context, publisherID, id string) (*AdminPackage, error)
	UpdatePackage(ctx context.Context, publisherID, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, publisherID, id string) error
	CreateExam(ctx context.Context, publisherID string, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, publisherID, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, publisherID, id string) error
	CreateQuestion(ctx context.Context, publisherID string, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, publisherID, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, publisherID, id string) error
	BulkDeleteQuestions(ctx context.Context, publisherID string, ids []string, packageID string) (int, error)
	ListCBTPublishSettings(ctx context.Context, publisherID string) ([]CBTPublishSetting, error)
	SetExamPublishPembahasan(ctx context.Context, publisherID, examID string, publish bool) (*CBTPublishSetting, error)
	ListCBTParticipants(ctx context.Context, publisherID, examID string) ([]CBTParticipant, error)
	UpdatePayoutAccount(ctx context.Context, publisherID string, input TeacherPayoutAccount) (*TeacherPayoutAccount, error)
	CreatePayoutRequest(ctx context.Context, publisherID string, amount float64) (*TeacherWithdrawalRequest, error)
	CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*TeacherWithdrawalRequest, error)
	Appeal(ctx context.Context, userID, appealImageDataURL string) error
}

type TeacherService interface {
	GetDashboard(ctx context.Context, publisherID string) (*TeacherDashboard, error)
	CreatePackage(ctx context.Context, publisherID string, input AdminPackageRequest) (*AdminPackage, error)
	CreatePackageBundle(ctx context.Context, publisherID string, input PackageBundle) (*AdminPackage, error)
	UpdatePackage(ctx context.Context, publisherID, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, publisherID, id string) error
	CreateExam(ctx context.Context, publisherID string, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, publisherID, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, publisherID, id string) error
	CreateQuestion(ctx context.Context, publisherID string, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, publisherID, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, publisherID, id string) error
	BulkDeleteQuestions(ctx context.Context, publisherID string, ids []string, packageID string) (int, error)
	ListCBTPublishSettings(ctx context.Context, publisherID string) ([]CBTPublishSetting, error)
	SetExamPublishPembahasan(ctx context.Context, publisherID, examID string, publish bool) (*CBTPublishSetting, error)
	ListCBTParticipants(ctx context.Context, publisherID, examID string) ([]CBTParticipant, error)
	UpdatePayoutAccount(ctx context.Context, publisherID string, input TeacherPayoutAccount) (*TeacherPayoutAccount, error)
	CreatePayoutRequest(ctx context.Context, publisherID string, amount float64) (*TeacherWithdrawalRequest, error)
	CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*TeacherWithdrawalRequest, error)
	Appeal(ctx context.Context, userID string, input TeacherAppealRequest) error
}
