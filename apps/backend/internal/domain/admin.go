package domain

import (
	"context"
	"errors"
	"time"
)

var ErrAdminConflict = errors.New("record is still used by other data")

var (
	ErrPayoutAccountRequired = errors.New("lengkapi data rekening pencairan terlebih dahulu")
	ErrPayoutMinimum         = errors.New("saldo belum memenuhi minimum pencairan")
	ErrPayoutRequestActive   = errors.New("masih ada pengajuan pencairan yang sedang diproses")
	ErrPayoutRequestNotFound = errors.New("pengajuan pencairan tidak ditemukan")
	ErrPayoutTransition      = errors.New("perubahan status pencairan tidak diperbolehkan")
)

var (
	ErrMasterNotFound = errors.New("rekaman tidak ditemukan")
	ErrDuplicateName  = errors.New("nama sudah digunakan")
	ErrDuplicateKode  = errors.New("kode sudah digunakan")
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type MasterCategory = string

const (
	MasterMapel       MasterCategory = "mapel"
	MasterJenjang     MasterCategory = "jenjang"
	MasterTahunAjaran MasterCategory = "tahun-ajaran"
)

type MasterItem struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`
}

type AdminPackage struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Kode           string    `json:"kode"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	ValidityDays   int       `json:"validity_days"`
	Status         string    `json:"status"`
	Jenjang        string    `json:"jenjang"`
	PublisherID    string    `json:"publisher_id,omitempty"`
	PublisherEmail string    `json:"publisher_email,omitempty"`
	SalesCount     int64     `json:"sales_count"`
	ViewCount      int64     `json:"view_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminOverview struct {
	TotalUsers        int64 `json:"total_users"`
	TotalStudents     int64 `json:"total_students"`
	TotalPackages     int64 `json:"total_packages"`
	TotalExams        int64 `json:"total_exams"`
	TotalTransactions int64 `json:"total_transactions"`
	PaidTransactions  int64 `json:"paid_transactions"`
}

type HeroSlide struct {
	ID      string     `json:"id"`
	Eyebrow string     `json:"eyebrow"`
	Title   string     `json:"title"`
	Lead    string     `json:"lead"`
	Stats   [][]string `json:"stats"`
}

type SiteSettings struct {
	PlatformName               string      `json:"platform_name"`
	PlatformTagline            string      `json:"platform_tagline"`
	LogoDataURL                string      `json:"logo_data_url"`
	SupportEmail               string      `json:"support_email"`
	WhatsApp                   string      `json:"whatsapp"`
	InstagramURL               string      `json:"instagram_url"`
	YouTubeURL                 string      `json:"youtube_url"`
	HeroIntervalMS             int         `json:"hero_interval_ms"`
	CatalogIntervalMS          int         `json:"catalog_interval_ms"`
	DefaultPackageValidityDays int         `json:"default_package_validity_days"`
	DefaultExamDurationMinutes int         `json:"default_exam_duration_minutes"`
	DefaultExamTotalQuestions  int         `json:"default_exam_total_questions"`
	DefaultPassingScore        float64     `json:"default_passing_score"`
	HeroSlides                 []HeroSlide `json:"hero_slides"`
	UpdatedAt                  time.Time   `json:"updated_at"`
}

type AdminExam struct {
	ID              string    `json:"id"`
	PackageID       string    `json:"package_id"`
	Title           string    `json:"title"`
	PackageTitle    string    `json:"package_title"`
	MapelID         string    `json:"mapel_id"`
	TahunAjaranID   string    `json:"tahun_ajaran_id"`
	DurationMinutes int       `json:"duration_minutes"`
	TotalQuestions  int       `json:"total_questions"`
	PassingScore    float64   `json:"passing_score"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type AdminTransaction struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	UserEmail     string    `json:"user_email"`
	PackageTitle  string    `json:"package_title"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type FinanceSettings struct {
	PlatformCommissionPercent float64   `json:"platform_commission_percent"`
	DefaultDiscountPercent    float64   `json:"default_discount_percent"`
	TaxPercent                float64   `json:"tax_percent"`
	MinimumPayout             float64   `json:"minimum_payout"`
	PayoutCycle               string    `json:"payout_cycle"`
	AutoPayout                bool      `json:"auto_payout"`
	TeacherUploadFee          float64   `json:"teacher_upload_fee"`
	TeacherSalesBonusPercent  float64   `json:"teacher_sales_bonus_percent"`
	CommissionHoldDays        int       `json:"commission_hold_days"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type TeacherCommission struct {
	ID              string     `json:"id"`
	TeacherID       string     `json:"teacher_id"`
	TeacherEmail    string     `json:"teacher_email"`
	PackageID       string     `json:"package_id"`
	PackageTitle    string     `json:"package_title"`
	InvoiceNumber   string     `json:"invoice_number,omitempty"`
	Kind            string     `json:"kind"`
	BaseAmount      float64    `json:"base_amount"`
	RatePercent     float64    `json:"rate_percent"`
	Amount          float64    `json:"amount"`
	Status          string     `json:"status"`
	AvailableAt     time.Time  `json:"available_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	PayoutReference string     `json:"payout_reference,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type TeacherPayout struct {
	ID           string    `json:"id"`
	TeacherID    string    `json:"teacher_id"`
	TeacherEmail string    `json:"teacher_email"`
	Amount       float64   `json:"amount"`
	Reference    string    `json:"reference"`
	PaidAt       time.Time `json:"paid_at"`
}

type TeacherWithdrawalRequest struct {
	ID                string     `json:"id"`
	TeacherID         string     `json:"teacher_id"`
	TeacherEmail      string     `json:"teacher_email"`
	Amount            float64    `json:"amount"`
	Status            string     `json:"status"`
	PayoutMethod      string     `json:"payout_method"`
	Provider          string     `json:"provider"`
	AccountNumber     string     `json:"account_number"`
	AccountHolderName string     `json:"account_holder_name"`
	Phone             string     `json:"phone"`
	AdminNote         string     `json:"admin_note"`
	TransferReference string     `json:"transfer_reference"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PayoutRequestReview struct {
	Status    string `json:"status"`
	Note      string `json:"note"`
	Reference string `json:"reference"`
}

type TeacherPayoutAccount struct {
	TeacherID         string     `json:"teacher_id"`
	Method            string     `json:"method"`
	Provider          string     `json:"provider"`
	AccountNumber     string     `json:"account_number"`
	AccountHolderName string     `json:"account_holder_name"`
	Phone             string     `json:"phone"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type TeacherCommissionSummary struct {
	UploadFees float64 `json:"upload_fees"`
	SaleBonus  float64 `json:"sales_bonus"`
	Held       float64 `json:"held"`
	Available  float64 `json:"available"`
	Paid       float64 `json:"paid"`
}

type UserFinanceProfile struct {
	UserID                     string    `json:"user_id"`
	Email                      string    `json:"email"`
	Name                       string    `json:"name"`
	Role                       string    `json:"role"`
	CommissionPercent          *float64  `json:"commission_percent,omitempty"`
	DiscountPercent            *float64  `json:"discount_percent,omitempty"`
	EffectiveCommissionPercent float64   `json:"effective_commission_percent"`
	EffectiveDiscountPercent   float64   `json:"effective_discount_percent"`
	AccountStatus              string    `json:"account_status"`
	Notes                      string    `json:"notes"`
	TotalSpend                 float64   `json:"total_spend"`
	GrossRevenue               float64   `json:"gross_revenue"`
	TransactionCount           int64     `json:"transaction_count"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type FinanceOverview struct {
	GrossRevenue       float64 `json:"gross_revenue"`
	PendingRevenue     float64 `json:"pending_revenue"`
	PlatformCommission float64 `json:"platform_commission"`
	EstimatedPayouts   float64 `json:"estimated_payouts"`
	PaidTransactions   int64   `json:"paid_transactions"`
}

type FinanceDashboard struct {
	Overview       FinanceOverview            `json:"overview"`
	Settings       FinanceSettings            `json:"settings"`
	Users          []UserFinanceProfile       `json:"users"`
	Commissions    []TeacherCommission        `json:"commissions"`
	Payouts        []TeacherPayout            `json:"payouts"`
	PayoutAccounts []TeacherPayoutAccount     `json:"payout_accounts"`
	Summary        TeacherCommissionSummary   `json:"commission_summary"`
	PayoutRequests []TeacherWithdrawalRequest `json:"payout_requests"`
}

type TeacherPayoutRequest struct {
	Reference string `json:"reference"`
}

type UserFinanceUpdateRequest struct {
	CommissionPercent *float64 `json:"commission_percent"`
	DiscountPercent   *float64 `json:"discount_percent"`
	AccountStatus     string   `json:"account_status"`
	Notes             string   `json:"notes"`
}

type AdminQuestion struct {
	ID               string           `json:"id"`
	ExamID           string           `json:"exam_id"`
	ExamTitle        string           `json:"exam_title"`
	SubjectName      string           `json:"subject_name"`
	ContentText      string           `json:"content_text"`
	QuestionType     string           `json:"question_type"`
	PresentationType string           `json:"presentation_type"`
	GroupCode        string           `json:"group_code"`
	StimulusText     string           `json:"stimulus_text"`
	QuestionImageURL string           `json:"question_image_url"`
	StimulusImageURL string           `json:"stimulus_image_url"`
	CategoryLabels   []string         `json:"category_labels"`
	Options          []QuestionOption `json:"options"`
	CorrectAnswer    string           `json:"correct_answer"`
	ScoreWeight      float64          `json:"score_weight"`
	Explanation      string           `json:"explanation_text"`
	Status           string           `json:"status"`
}

type AdminDashboard struct {
	Overview      AdminOverview      `json:"overview"`
	Levels        []string           `json:"levels"`
	Jenjangs      []MasterItem       `json:"jenjangs"`
	Mapels        []MasterItem       `json:"mapels"`
	AcademicYears []MasterItem       `json:"academic_years"`
	Users         []UserResponse     `json:"users"`
	Packages      []AdminPackage     `json:"packages"`
	Exams         []AdminExam        `json:"exams"`
	Transactions  []AdminTransaction `json:"transactions"`
	Questions     []AdminQuestion    `json:"questions"`
}

type AdminUpdateUserRequest struct {
	Role        string `json:"role"`
	SchoolLevel string `json:"school_level"`
}

type AdminCreateUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	SchoolLevel string `json:"school_level"`
}

type AdminPackageRequest struct {
	Title        string  `json:"title"`
	Kode         string  `json:"kode"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	ValidityDays int     `json:"validity_days"`
	Status       string  `json:"status"`
	Jenjang      string  `json:"jenjang"`
}

type AdminExamRequest struct {
	PackageID       string  `json:"package_id"`
	Title           string  `json:"title"`
	MapelID         string  `json:"mapel_id"`
	TahunAjaranID   string  `json:"tahun_ajaran_id"`
	DurationMinutes int     `json:"duration_minutes"`
	TotalQuestions  int     `json:"total_questions"`
	PassingScore    float64 `json:"passing_score"`
	Status          string  `json:"status"`
}

type AdminQuestionRequest struct {
	ExamID           string           `json:"exam_id"`
	SubjectName      string           `json:"subject_name"`
	ContentText      string           `json:"content_text"`
	QuestionType     string           `json:"question_type"`
	PresentationType string           `json:"presentation_type"`
	GroupCode        string           `json:"group_code"`
	StimulusText     string           `json:"stimulus_text"`
	QuestionImageURL string           `json:"question_image_url"`
	StimulusImageURL string           `json:"stimulus_image_url"`
	CategoryLabels   []string         `json:"category_labels"`
	Options          []QuestionOption `json:"options"`
	CorrectAnswer    string           `json:"correct_answer"`
	ScoreWeight      float64          `json:"score_weight"`
	Explanation      string           `json:"explanation_text"`
	Status           string           `json:"status"`
}

// PackageBundle adalah pembuatan satu paket sekaligus ujian pertama dan
// daftar soal inline dalam satu transaksi.
type PackageBundle struct {
	Package   AdminPackageRequest    `json:"package"`
	Exam      *AdminExamRequest      `json:"exam,omitempty"`
	Questions []AdminQuestionRequest `json:"questions,omitempty"`
}

type AdminRepository interface {
	GetSiteSettings(ctx context.Context) (*SiteSettings, error)
	UpdateSiteSettings(ctx context.Context, input SiteSettings) (*SiteSettings, error)
	GetDashboard(ctx context.Context) (*AdminDashboard, error)
	CreateUser(ctx context.Context, user User) (*UserResponse, error)
	UpdateUser(ctx context.Context, userID, role, schoolLevel string) (*UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	CreatePackage(ctx context.Context, input AdminPackageRequest) (*AdminPackage, error)
	CreatePackageBundle(ctx context.Context, input PackageBundle) (*AdminPackage, error)
	UpdatePackage(ctx context.Context, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, id string) error
	CreateExam(ctx context.Context, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, id string) error
	CreateQuestion(ctx context.Context, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, id string) error
	ListMaster(ctx context.Context, category MasterCategory) ([]MasterItem, error)
	CreateMaster(ctx context.Context, category MasterCategory, nama string) (*MasterItem, error)
	UpdateMaster(ctx context.Context, category MasterCategory, id, nama string) (*MasterItem, error)
	DeleteMaster(ctx context.Context, category MasterCategory, id string) error
	GetFinanceDashboard(ctx context.Context) (*FinanceDashboard, error)
	UpdateFinanceSettings(ctx context.Context, input FinanceSettings) (*FinanceSettings, error)
	UpdateUserFinance(ctx context.Context, userID string, input UserFinanceUpdateRequest) (*UserFinanceProfile, error)
	PayTeacherCommissions(ctx context.Context, teacherID, reference string) (*TeacherPayout, error)
	ReviewPayoutRequest(ctx context.Context, requestID string, input PayoutRequestReview) (*TeacherWithdrawalRequest, error)
}

type AdminService interface {
	GetSiteSettings(ctx context.Context) (*SiteSettings, error)
	UpdateSiteSettings(ctx context.Context, input SiteSettings) (*SiteSettings, error)
	GetDashboard(ctx context.Context) (*AdminDashboard, error)
	CreateUser(ctx context.Context, input AdminCreateUserRequest) (*UserResponse, error)
	UpdateUser(ctx context.Context, actorID, userID string, input AdminUpdateUserRequest) (*UserResponse, error)
	DeleteUser(ctx context.Context, actorID, userID string) error
	CreatePackage(ctx context.Context, input AdminPackageRequest) (*AdminPackage, error)
	CreatePackageBundle(ctx context.Context, input PackageBundle) (*AdminPackage, error)
	UpdatePackage(ctx context.Context, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, id string) error
	CreateExam(ctx context.Context, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, id string) error
	CreateQuestion(ctx context.Context, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, id string) error
	ListMaster(ctx context.Context, category MasterCategory) ([]MasterItem, error)
	CreateMaster(ctx context.Context, category MasterCategory, nama string) (*MasterItem, error)
	UpdateMaster(ctx context.Context, category MasterCategory, id, nama string) (*MasterItem, error)
	DeleteMaster(ctx context.Context, category MasterCategory, id string) error
	GetFinanceDashboard(ctx context.Context) (*FinanceDashboard, error)
	UpdateFinanceSettings(ctx context.Context, input FinanceSettings) (*FinanceSettings, error)
	UpdateUserFinance(ctx context.Context, userID string, input UserFinanceUpdateRequest) (*UserFinanceProfile, error)
	PayTeacherCommissions(ctx context.Context, teacherID, reference string) (*TeacherPayout, error)
	ReviewPayoutRequest(ctx context.Context, requestID string, input PayoutRequestReview) (*TeacherWithdrawalRequest, error)
}
