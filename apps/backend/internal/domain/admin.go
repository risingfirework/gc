// Package domain mendefinisikan tipe-tipe inti dan kontrak antarmuka
// (repository & service) yang dipakai seluruh lapisan aplikasi.
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrAdminConflict = errors.New("record is still used by other data")

var ErrPackageHasAttempts = errors.New("paket memiliki ujian yang sudah pernah dikerjakan; hapus paket tidak diperbolehkan")

var ErrPackageInUse = errors.New("paket sudah pernah dibeli/diambil siswa; hapus paket tidak diperbolehkan, nonaktifkan saja agar tidak tersedia")

var ErrExamHasAttempts = errors.New("ujian sudah pernah dikerjakan siswa; hapus ujian tidak diperbolehkan")

var (
	ErrPayoutAccountRequired     = errors.New("lengkapi data rekening pencairan terlebih dahulu")
	ErrPayoutMinimum             = errors.New("saldo belum memenuhi minimum pencairan")
	ErrInsufficientPayoutBalance = errors.New("nominal melebihi saldo dapat dicairkan")
	ErrPayoutProofRequired       = errors.New("lampirkan bukti transfer sebelum mengonfirmasi dana ditransfer")
	ErrPayoutRequestActive       = errors.New("masih ada pengajuan pencairan yang sedang diproses")
	ErrPayoutRequestNotFound     = errors.New("pengajuan pencairan tidak ditemukan")
	ErrPayoutTransition          = errors.New("perubahan status pencairan tidak diperbolehkan")
)

var (
	ErrMasterNotFound = errors.New("rekaman tidak ditemukan")
	ErrDuplicateName  = errors.New("nama sudah digunakan")
	ErrDuplicateKode  = errors.New("kode sudah digunakan")
	ErrLastOwnerGuard = errors.New("tidak dapat menurunkan peran pemilik terakhir")
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
	MasterKategori    MasterCategory = "kategori"
	MasterKelas       MasterCategory = "kelas"
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
	ExamType       string    `json:"exam_type"`
	CBTToken       string    `json:"cbt_token,omitempty"`
	StartDate      string    `json:"start_date,omitempty"`
	EndDate        string    `json:"end_date,omitempty"`
	KategoriID     string    `json:"kategori_id,omitempty"`
	KategoriName   string    `json:"kategori_name,omitempty"`
	KelasID        string    `json:"kelas_id,omitempty"`
	KelasName      string    `json:"kelas_name,omitempty"`
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
	ID           string `json:"id"`
	ImageDataURL string `json:"image_data_url"`
	Title        string `json:"title"`
}

type YouTubeVideo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type SiteSettings struct {
	PlatformName               string         `json:"platform_name"`
	PlatformTagline            string         `json:"platform_tagline"`
	LogoDataURL                string         `json:"logo_data_url"`
	FaviconDataURL             string         `json:"favicon_data_url"`
	SupportEmail               string         `json:"support_email"`
	WhatsApp                   string         `json:"whatsapp"`
	InstagramURL               string         `json:"instagram_url"`
	YouTubeURL                 string         `json:"youtube_url"`
	YouTubeVideos              []YouTubeVideo `json:"youtube_videos"`
	HeroIntervalMS             int            `json:"hero_interval_ms"`
	CatalogIntervalMS          int            `json:"catalog_interval_ms"`
	DefaultPackageValidityDays int            `json:"default_package_validity_days"`
	DefaultExamDurationMinutes int            `json:"default_exam_duration_minutes"`
	DefaultExamTotalQuestions  int            `json:"default_exam_total_questions"`
	DefaultPassingScore        float64        `json:"default_passing_score"`
	HeroSlides                 []HeroSlide    `json:"hero_slides"`
	UpdatedAt                  time.Time      `json:"updated_at"`
}

type AdminExam struct {
	ID                string    `json:"id"`
	PackageID         string    `json:"package_id"`
	Title             string    `json:"title"`
	PackageTitle      string    `json:"package_title"`
	MapelID           string    `json:"mapel_id"`
	TahunAjaranID     string    `json:"tahun_ajaran_id"`
	DurationMinutes   int       `json:"duration_minutes"`
	TotalQuestions    int       `json:"total_questions"`
	PassingScore      float64   `json:"passing_score"`
	Status            string    `json:"status"`
	PublishPembahasan bool      `json:"publish_pembahasan"`
	CreatedAt         time.Time `json:"created_at"`
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

type RefundTransactionRequest struct {
	Reason string `json:"reason"`
}

// Page adalah hasil terpaginasi generik yang dikembalikan endpoint daftar.
type Page[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Count int `json:"count"`
	Total int `json:"total"`
}

func NewPage[T any](items []T, page, perPage, total int) Page[T] {
	return Page[T]{Items: items, Page: page, Count: perPage, Total: total}
}

type UserSummary struct {
	Total    int `json:"total"`
	Students int `json:"students"`
	Teachers int `json:"teachers"`
	Admins   int `json:"admins"`
}

type AdminUsersPage struct {
	Items   []UserResponse `json:"items"`
	Page    int            `json:"page"`
	Count   int            `json:"count"`
	Total   int            `json:"total"`
	Summary UserSummary    `json:"summary"`
}

type PackageStatusCounts struct {
	Active          int `json:"active"`
	Inactive        int `json:"inactive"`
	ActiveTeacher   int `json:"active_teacher"`
	InactiveTeacher int `json:"inactive_teacher"`
}

type AdminPackagesPage struct {
	Items  []AdminPackage      `json:"items"`
	Page   int                 `json:"page"`
	Count  int                 `json:"count"`
	Total  int                 `json:"total"`
	Counts PackageStatusCounts `json:"counts"`
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
	AffiliateRatePercent      float64   `json:"affiliate_rate_percent"`
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
	ProofURL          string     `json:"proof_url"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PayoutRequestReview struct {
	Status    string `json:"status"`
	Note      string `json:"note"`
	Reference string `json:"reference"`
	ProofURL  string `json:"proof_url"`
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

type AuditLogEntry struct {
	ID         string          `json:"id"`
	ActorID    string          `json:"actor_id"`
	ActorEmail string          `json:"actor_email"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Detail     json.RawMessage `json:"detail"`
	CreatedAt  time.Time       `json:"created_at"`
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
	Kategoris     []MasterItem       `json:"kategoris"`
	Kelas         []MasterItem       `json:"kelas"`
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
	ExamType     string  `json:"exam_type"`
	CBTToken     string  `json:"cbt_token"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	KategoriID   string  `json:"kategori_id"`
	KelasID      string  `json:"kelas_id"`
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
	GetPackage(ctx context.Context, id string) (*AdminPackage, error)
	UpdatePackage(ctx context.Context, id string, input AdminPackageRequest) (*AdminPackage, error)
	DeletePackage(ctx context.Context, id string) error
	CreateExam(ctx context.Context, input AdminExamRequest) (*AdminExam, error)
	UpdateExam(ctx context.Context, id string, input AdminExamRequest) (*AdminExam, error)
	DeleteExam(ctx context.Context, id string) error
	CreateQuestion(ctx context.Context, input AdminQuestionRequest) (*AdminQuestion, error)
	UpdateQuestion(ctx context.Context, id string, input AdminQuestionRequest) (*AdminQuestion, error)
	DeleteQuestion(ctx context.Context, id string) error
	BulkDeleteQuestions(ctx context.Context, ids []string, packageID string) (int, error)
	ListMaster(ctx context.Context, category MasterCategory) ([]MasterItem, error)
	CreateMaster(ctx context.Context, category MasterCategory, nama string) (*MasterItem, error)
	UpdateMaster(ctx context.Context, category MasterCategory, id, nama string) (*MasterItem, error)
	DeleteMaster(ctx context.Context, category MasterCategory, id string) error
	GetFinanceDashboard(ctx context.Context) (*FinanceDashboard, error)
	ExportFinanceLedger(ctx context.Context) ([]TeacherCommission, error)
	UpdateFinanceSettings(ctx context.Context, input FinanceSettings) (*FinanceSettings, error)
	UpdateUserFinance(ctx context.Context, userID string, input UserFinanceUpdateRequest) (*UserFinanceProfile, error)
	PayTeacherCommissions(ctx context.Context, teacherID, reference string) (*TeacherPayout, error)
	ReviewPayoutRequest(ctx context.Context, requestID string, input PayoutRequestReview) (*TeacherWithdrawalRequest, error)
	ListPayoutEligibleTeachers(ctx context.Context) ([]string, error)
	ListTransactions(ctx context.Context, page, perPage int, status string) (Page[AdminTransaction], error)
	ListUsers(ctx context.Context, page, perPage int, role, level, q string) (*AdminUsersPage, error)
	ListAdminPackages(ctx context.Context, page, perPage int, status, jenjang, q, examType string) (*AdminPackagesPage, error)
	ListAdminExams(ctx context.Context, page, perPage int, packageID string) (Page[AdminExam], error)
	ListAdminQuestions(ctx context.Context, page, perPage int, packageID string) (Page[AdminQuestion], error)
	ListCBTPublishSettings(ctx context.Context) ([]CBTPublishSetting, error)
	SetExamPublishPembahasan(ctx context.Context, examID string, publish bool) (*CBTPublishSetting, error)
	ListCBTParticipants(ctx context.Context, examID string) ([]CBTParticipant, error)
	RefundTransaction(ctx context.Context, transactionID, reason string, now time.Time) (*AdminTransaction, error)
	CountOwners(ctx context.Context) (int, error)
	GetUserRole(ctx context.Context, userID string) (string, error)
	EnsureAffiliate(ctx context.Context, userID, referralCode string) (string, error)
	InsertAuditLog(ctx context.Context, entry AuditLogEntry) error
	ListAuditLogs(ctx context.Context, limit int) ([]AuditLogEntry, error)
	ListTeacherVerifications(ctx context.Context, page, perPage int, status string) (Page[TeacherVerification], error)
	ApproveTeacher(ctx context.Context, userID string) error
	RejectTeacher(ctx context.Context, userID, reason string) error
	ListPendingTeacherSIMPKBChecks(ctx context.Context, limit int) ([]TeacherSIMPKBJob, error)
	GetTeacherKTP(ctx context.Context, userID string) (string, error)
	SetTeacherSIMPKBChecking(ctx context.Context, userID string) error
	SaveTeacherSIMPKBResult(ctx context.Context, userID, status, image string, checkedAt time.Time) error
}

type AdminService interface {
	GetSiteSettings(ctx context.Context) (*SiteSettings, error)
	UpdateSiteSettings(ctx context.Context, actorID, actorEmail string, input SiteSettings) (*SiteSettings, error)
	GetDashboard(ctx context.Context) (*AdminDashboard, error)
	CreateUser(ctx context.Context, actorID, actorEmail string, input AdminCreateUserRequest) (*UserResponse, error)
	UpdateUser(ctx context.Context, actorID, actorEmail, userID string, input AdminUpdateUserRequest) (*UserResponse, error)
	DeleteUser(ctx context.Context, actorID, actorEmail, userID string) error
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
	BulkDeleteQuestions(ctx context.Context, ids []string, packageID string) (int, error)
	ListMaster(ctx context.Context, category MasterCategory) ([]MasterItem, error)
	CreateMaster(ctx context.Context, category MasterCategory, nama string) (*MasterItem, error)
	UpdateMaster(ctx context.Context, category MasterCategory, id, nama string) (*MasterItem, error)
	DeleteMaster(ctx context.Context, category MasterCategory, id string) error
	GetFinanceDashboard(ctx context.Context) (*FinanceDashboard, error)
	ExportFinanceCSV(ctx context.Context) ([]byte, error)
	UpdateFinanceSettings(ctx context.Context, actorID, actorEmail string, input FinanceSettings) (*FinanceSettings, error)
	UpdateUserFinance(ctx context.Context, actorID, actorEmail, userID string, input UserFinanceUpdateRequest) (*UserFinanceProfile, error)
	PayTeacherCommissions(ctx context.Context, actorID, actorEmail, teacherID, reference string) (*TeacherPayout, error)
	ReviewPayoutRequest(ctx context.Context, actorID, actorEmail, requestID string, input PayoutRequestReview) (*TeacherWithdrawalRequest, error)
	AutoPayTeachers(ctx context.Context) ([]TeacherPayout, error)
	ListTransactions(ctx context.Context, page, perPage int, status string) (Page[AdminTransaction], error)
	ListUsers(ctx context.Context, page, perPage int, role, level, q string) (*AdminUsersPage, error)
	ListAdminPackages(ctx context.Context, page, perPage int, status, jenjang, q, examType string) (*AdminPackagesPage, error)
	ListAdminExams(ctx context.Context, page, perPage int, packageID string) (Page[AdminExam], error)
	ListAdminQuestions(ctx context.Context, page, perPage int, packageID string) (Page[AdminQuestion], error)
	ListCBTPublishSettings(ctx context.Context) ([]CBTPublishSetting, error)
	SetExamPublishPembahasan(ctx context.Context, actorID, actorEmail, examID string, publish bool) (*CBTPublishSetting, error)
	ListCBTParticipants(ctx context.Context, examID string) ([]CBTParticipant, error)
	RefundTransaction(ctx context.Context, actorID, actorEmail, transactionID string, input RefundTransactionRequest) (*AdminTransaction, error)
	ListAuditLogs(ctx context.Context) ([]AuditLogEntry, error)
	ListTeacherVerifications(ctx context.Context, page, perPage int, status string) (Page[TeacherVerification], error)
	ApproveTeacher(ctx context.Context, actorID, actorEmail, userID string) error
	RejectTeacher(ctx context.Context, actorID, actorEmail, userID, reason string) error
	CheckTeacherSIMPKB(ctx context.Context, userID string) error
	SIMPKBWorkerTick(ctx context.Context) error
}
