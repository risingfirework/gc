package domain

import "context"

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
	UpdatePackage(ctx context.Context, publisherID, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, publisherID, id string) error
	CreateExam(ctx context.Context, publisherID string, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, publisherID, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, publisherID, id string) error
	CreateQuestion(ctx context.Context, publisherID string, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, publisherID, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, publisherID, id string) error
	UpdatePayoutAccount(ctx context.Context, publisherID string, input TeacherPayoutAccount) (*TeacherPayoutAccount, error)
	CreatePayoutRequest(ctx context.Context, publisherID string) (*TeacherWithdrawalRequest, error)
	CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*TeacherWithdrawalRequest, error)
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
	UpdatePayoutAccount(ctx context.Context, publisherID string, input TeacherPayoutAccount) (*TeacherPayoutAccount, error)
	CreatePayoutRequest(ctx context.Context, publisherID string) (*TeacherWithdrawalRequest, error)
	CancelPayoutRequest(ctx context.Context, publisherID, requestID string) (*TeacherWithdrawalRequest, error)
}
