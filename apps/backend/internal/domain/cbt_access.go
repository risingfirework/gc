package domain

import (
	"context"
	"errors"
	"time"
)

// Mode akses fitur CBT untuk guru.
const (
	CBTAccessModeFree          = "free"
	CBTAccessModeTrialThenPaid = "trial_then_paid"
	CBTAccessModePaid          = "paid"
)

const (
	CBTAccessLockCreateNew = "create_new_cbt"
	CBTAccessLockAll       = "all_cbt_features"
)

const (
	CBTAccessPaymentManual  = "manual"
	CBTAccessPaymentGateway = "gateway"
)

const (
	CBTAccessQuotaNever  = "never"
	CBTAccessQuotaYearly = "yearly"
)

// Status entitlement guru.
const (
	CBTAccessEntitlementTrialAvailable = "trial_available"
	CBTAccessEntitlementTrialUsed      = "trial_used"
	CBTAccessEntitlementActive         = "active"
	CBTAccessEntitlementExpired        = "expired"
	CBTAccessEntitlementPending        = "pending"
	CBTAccessEntitlementBlocked        = "blocked"
)

// Sumber entitlement.
const (
	CBTAccessSourceFree       = "free"
	CBTAccessSourceTrial      = "trial"
	CBTAccessSourcePayment    = "payment"
	CBTAccessSourceAdminGrant = "admin_grant"
)

// Alasan status akses yang dikirim ke frontend.
const (
	CBTAccessReasonFreeMode          = "free_mode"
	CBTAccessReasonActive            = "active"
	CBTAccessReasonTrialAvailable    = "trial_available"
	CBTAccessReasonTrialUsed         = "trial_used"
	CBTAccessReasonGracePeriod       = "grace_period"
	CBTAccessReasonPaymentRequired   = "payment_required"
	CBTAccessReasonPendingReview     = "pending_review"
	CBTAccessReasonExpired           = "expired"
	CBTAccessReasonVerifiedRequired  = "verified_required"
	CBTAccessReasonManageExisting    = "manage_existing"
	CBTAccessReasonAccessDenied      = "access_denied"
	CBTAccessReasonSettingsInvalid   = "cbt_settings_invalid"
	CBTAccessReasonPaymentNotFound   = "cbt_payment_not_found"
	CBTAccessReasonPaymentTransition = "cbt_payment_transition"
)

// Status pembayaran akses CBT.
const (
	CBTPaymentPending   = "pending"
	CBTPaymentApproved  = "approved"
	CBTPaymentRejected  = "rejected"
	CBTPaymentCancelled = "cancelled"
)

var (
	ErrCBTAccessDenied         = errors.New("akses CBT guru dibatasi")
	ErrCBTAccessSettings       = errors.New("pengaturan akses CBT tidak valid")
	ErrCBTPaymentNotFound      = errors.New("pembayaran akses CBT tidak ditemukan")
	ErrCBTPaymentTransition    = errors.New("status pembayaran akses CBT tidak dapat diubah")
	ErrCBTPaymentPending       = errors.New("masih ada pengajuan pembayaran yang menunggu verifikasi")
	ErrCBTPaymentNotRequired   = errors.New("akses CBT saat ini tidak memerlukan pembayaran")
	ErrCBTTeacherNotFound      = errors.New("guru tidak ditemukan")
	ErrCBTAccessTrialExhausted = errors.New("jatah gratis CBT sudah habis")
)

// CBTAccessSettings adalah kebijakan global akses CBT guru (satu baris).
type CBTAccessSettings struct {
	AccessMode             string    `json:"access_mode"`
	PriceAmount            int64     `json:"price_amount"`
	Currency               string    `json:"currency"`
	ValidityDays           int       `json:"validity_days"`
	FreeQuota              int       `json:"free_quota"`
	FreeQuotaReset         string    `json:"free_quota_reset"`
	PaymentMode            string    `json:"payment_mode"`
	PaymentInstructions    string    `json:"payment_instructions"`
	ContactInfo            string    `json:"contact_info"`
	LockScope              string    `json:"lock_scope"`
	AllowManageExisting    bool      `json:"allow_manage_existing"`
	GraceDays              int       `json:"grace_days"`
	RequireVerifiedTeacher bool      `json:"require_verified_teacher"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// TeacherCBTEntitlement adalah status akses CBT seorang guru.
type TeacherCBTEntitlement struct {
	UserID        string     `json:"user_id"`
	Status        string     `json:"status"`
	FreeQuotaUsed int        `json:"free_quota_used"`
	ActiveFrom    *time.Time `json:"active_from,omitempty"`
	ActiveUntil   *time.Time `json:"active_until,omitempty"`
	Source        string     `json:"source,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TeacherCBTPayment adalah pengajuan pembayaran akses CBT oleh guru.
type TeacherCBTPayment struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	UserEmail       string     `json:"user_email,omitempty"`
	UserName        string     `json:"user_name,omitempty"`
	Amount          int64      `json:"amount"`
	Currency        string     `json:"currency"`
	ProofImage      string     `json:"proof_image,omitempty"`
	Note            string     `json:"note"`
	Status          string     `json:"status"`
	PeriodStart     *time.Time `json:"period_start,omitempty"`
	PeriodEnd       *time.Time `json:"period_end,omitempty"`
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason string     `json:"rejection_reason"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CBTAccessStatus adalah gabungan kebijakan + entitlement + keputusan akses
// yang dikirim ke frontend guru.
type CBTAccessStatus struct {
	Settings       CBTAccessSettings     `json:"settings"`
	Entitlement    TeacherCBTEntitlement `json:"entitlement"`
	CanCreateCBT   bool                  `json:"can_create_cbt"`
	Reason         string                `json:"reason"`
	Message        string                `json:"message"`
	PendingPayment *TeacherCBTPayment    `json:"pending_payment,omitempty"`
}

// TeacherCBTAccessRow adalah baris panel admin untuk mengelola akses guru.
type TeacherCBTAccessRow struct {
	UserID             string                `json:"user_id"`
	Email              string                `json:"email"`
	Name               string                `json:"name"`
	VerificationStatus string                `json:"verification_status"`
	Entitlement        TeacherCBTEntitlement `json:"entitlement"`
	CanCreateCBT       bool                  `json:"can_create_cbt"`
	Reason             string                `json:"reason"`
	Message            string                `json:"message"`
	HasPendingPayment  bool                  `json:"has_pending_payment"`
}

type CBTPaymentRequest struct {
	ProofImage string `json:"proof_image"`
	Note       string `json:"note"`
}

type CBTPaymentReviewRequest struct {
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`
	Note        string     `json:"note"`
	Reason      string     `json:"reason"`
}

type CBTAccessGrantRequest struct {
	ValidityDays int    `json:"validity_days"`
	Reason       string `json:"reason"`
}

// CBTAccessDeniedError membawa payload status agar frontend dapat langsung
// menampilkan modal tarif tanpa permintaan tambahan.
type CBTAccessDeniedError struct {
	Reason  string           `json:"reason"`
	Message string           `json:"message"`
	Access  *CBTAccessStatus `json:"access,omitempty"`
}

func (e *CBTAccessDeniedError) Error() string {
	if e == nil {
		return ErrCBTAccessDenied.Error()
	}
	if e.Message != "" {
		return e.Message
	}
	return ErrCBTAccessDenied.Error()
}

func (e *CBTAccessDeniedError) Unwrap() error { return ErrCBTAccessDenied }

type CBTAccessRepository interface {
	GetSettings(ctx context.Context) (*CBTAccessSettings, error)
	UpdateSettings(ctx context.Context, input CBTAccessSettings, updatedBy string) (*CBTAccessSettings, error)
	GetEntitlement(ctx context.Context, userID string) (*TeacherCBTEntitlement, error)
	EnsureEntitlement(ctx context.Context, userID string) (*TeacherCBTEntitlement, error)
	GetTeacherVerification(ctx context.Context, userID string) (string, error)
	ConsumeTrial(ctx context.Context, userID string, freeQuota int, now time.Time) (bool, error)
	RefundTrial(ctx context.Context, userID string, freeQuota int) error
	ActivateEntitlement(ctx context.Context, userID, source string, from time.Time, until *time.Time) error
	RevokeEntitlement(ctx context.Context, userID string, now time.Time) error
	ListPayments(ctx context.Context, status string, page, perPage int) (Page[TeacherCBTPayment], error)
	ListUserPayments(ctx context.Context, userID string, limit int) ([]TeacherCBTPayment, error)
	GetPendingPayment(ctx context.Context, userID string) (*TeacherCBTPayment, error)
	GetPayment(ctx context.Context, id string) (*TeacherCBTPayment, error)
	CreatePayment(ctx context.Context, payment TeacherCBTPayment) (*TeacherCBTPayment, error)
	ApprovePayment(ctx context.Context, id, reviewerID string, now time.Time, periodStart, periodEnd *time.Time) (*TeacherCBTPayment, error)
	RejectPayment(ctx context.Context, id, reviewerID, reason string, now time.Time) (*TeacherCBTPayment, error)
	ListTeacherAccess(ctx context.Context, status, q string, page, perPage int) (Page[TeacherCBTAccessRow], error)
	InsertAuditLog(ctx context.Context, entry AuditLogEntry) error
}

type CBTAccessService interface {
	GetSettings(ctx context.Context) (*CBTAccessSettings, error)
	UpdateSettings(ctx context.Context, actorID, actorEmail string, input CBTAccessSettings) (*CBTAccessSettings, error)
	GetTeacherStatus(ctx context.Context, userID string) (*CBTAccessStatus, error)
	BeginCBTCreation(ctx context.Context, userID string) (string, error)
	RefundCBTCreation(ctx context.Context, userID, token string) error
	SubmitPayment(ctx context.Context, userID string, input CBTPaymentRequest) (*TeacherCBTPayment, error)
	ListMyPayments(ctx context.Context, userID string) ([]TeacherCBTPayment, error)
	ListPayments(ctx context.Context, status string, page, perPage int) (Page[TeacherCBTPayment], error)
	ApprovePayment(ctx context.Context, actorID, actorEmail, paymentID string, input CBTPaymentReviewRequest) (*TeacherCBTPayment, error)
	RejectPayment(ctx context.Context, actorID, actorEmail, paymentID string, input CBTPaymentReviewRequest) (*TeacherCBTPayment, error)
	ListTeacherAccess(ctx context.Context, status, q string, page, perPage int) (Page[TeacherCBTAccessRow], error)
	GrantAccess(ctx context.Context, actorID, actorEmail, userID string, input CBTAccessGrantRequest) error
	RevokeAccess(ctx context.Context, actorID, actorEmail, userID, reason string) error
}
