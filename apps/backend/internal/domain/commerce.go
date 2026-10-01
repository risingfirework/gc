package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPackageNotFound          = errors.New("package not found")
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrInvalidPayment           = errors.New("invalid payment request")
	ErrInvalidWebhookSignature  = errors.New("invalid webhook signature")
	ErrInvalidPaymentTransition = errors.New("invalid payment status transition")
	ErrNoPurchasedPackage       = errors.New("no purchased package")
	ErrNotFreePackage           = errors.New("package is not free")
	ErrPackageAlreadyOwned      = errors.New("package already owned")
	ErrTransactionNotRefundable = errors.New("transaction cannot be refunded")
	ErrPaymentExpired           = errors.New("payment window expired")
	ErrPendingPaymentExists     = errors.New("pending payment for this package already exists")
	ErrPaymentSettingsNotFound  = errors.New("payment settings not found")
	ErrPaymentInitializing      = errors.New("payment session is being initialized")
)

type Package struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Kode           string    `json:"kode"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	ValidityDays   int       `json:"validity_days"`
	Status         string    `json:"status"`
	Jenjang        string    `json:"jenjang"`
	ExamType       string    `json:"exam_type"`
	PublisherID    string    `json:"publisher_id,omitempty"`
	PublisherEmail string    `json:"publisher_email,omitempty"`
	ExamCount      int       `json:"exam_count"`
	QuestionCount  int       `json:"question_count"`
	SalesCount     int64     `json:"sales_count"`
	ViewCount      int64     `json:"view_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type OwnedPackage struct {
	Package
	ExpiredAt time.Time `json:"expired_at"`
	PaidAt    time.Time `json:"paid_at"`
}

type Transaction struct {
	ID                       string     `json:"id"`
	UserID                   string     `json:"user_id"`
	PackageID                string     `json:"package_id"`
	InvoiceNumber            string     `json:"invoice_number"`
	Amount                   float64    `json:"amount"`
	PlatformCommission       float64    `json:"platform_commission"`
	PaymentStatus            string     `json:"payment_status"`
	PaymentMethod            *string    `json:"payment_method,omitempty"`
	PaymentURL               *string    `json:"payment_url,omitempty"`
	ProviderSessionID        *string    `json:"provider_session_id,omitempty"`
	ProviderPaymentRequestID *string    `json:"provider_payment_request_id,omitempty"`
	ProviderRefundID         *string    `json:"provider_refund_id,omitempty"`
	ExpiresAt                *time.Time `json:"expires_at,omitempty"`
	PaidAt                   *time.Time `json:"paid_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
}

type CheckoutRequest struct {
	PackageID     string `json:"package_id"`
	PaymentMethod string `json:"payment_method"`
	ReferralCode  string `json:"referral_code,omitempty"`
}

type PackageViewRequest struct {
	VisitorKey string `json:"visitor_key"`
}

type CheckoutResponse struct {
	Transaction Transaction `json:"transaction"`
	PaymentURL  string      `json:"payment_url"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

type PendingTransaction struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	PackageTitle  string    `json:"package_title"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	PaymentURL    string    `json:"payment_url"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

type UserPricingPolicy struct {
	DiscountPercent float64 `json:"discount_percent"`
	AccountActive   bool    `json:"account_active"`
}

type PaymentWebhookRequest struct {
	EventID                  string     `json:"event_id"`
	InvoiceNumber            string     `json:"invoice_number"`
	PaymentStatus            string     `json:"payment_status"`
	PaymentMethod            string     `json:"payment_method,omitempty"`
	Amount                   float64    `json:"amount"`
	PaidAt                   *time.Time `json:"paid_at,omitempty"`
	ProviderSessionID        string     `json:"provider_session_id,omitempty"`
	ProviderPaymentRequestID string     `json:"provider_payment_request_id,omitempty"`
	ProviderRefundID         string     `json:"provider_refund_id,omitempty"`
}

type PaymentSession struct {
	ID               string
	PaymentRequestID string
	URL              string
}

type PaymentRefund struct {
	ID     string
	Status string
}

type Invoice struct {
	PlatformName    string
	PlatformTagline string
	InvoiceNumber   string
	BuyerName       string
	BuyerEmail      string
	BuyerSchool     string
	PackageTitle    string
	PackageKode     string
	PackageJenjang  string
	PackageValidity int
	Price           float64
	PaymentMethod   string
	PaymentStatus   string
	PaidAt          *time.Time
	CreatedAt       time.Time
	TransactionID   string
}

type PaymentRepository interface {
	ListPackages(ctx context.Context, limit, offset int) ([]Package, error)
	GetPackage(ctx context.Context, packageID string) (*Package, error)
	ListMyPackages(ctx context.Context, userID string) ([]OwnedPackage, error)
	ClaimFreePackage(ctx context.Context, userID, packageID string) (*OwnedPackage, error)
	ReserveTransaction(ctx context.Context, transaction Transaction, idempotencyKey string) (*Transaction, bool, error)
	AttachPaymentSession(ctx context.Context, transactionID string, session PaymentSession) (*Transaction, error)
	FailPaymentInitialization(ctx context.Context, transactionID string) error
	ProcessWebhook(ctx context.Context, event PaymentWebhookRequest, payloadSHA256 string, now time.Time) (bool, error)
	GetUserFinancePolicy(ctx context.Context, userID string) (discountPercent float64, accountActive bool, err error)
	TrackPackageView(ctx context.Context, packageID, visitorKey string) (int64, error)
	HasEverOwnedPackage(ctx context.Context, userID, packageID string) (bool, error)
	HasPendingCheckout(ctx context.Context, userID, packageID, idempotencyKey string) (bool, error)
	ListMyTransactions(ctx context.Context, userID string) ([]AdminTransaction, error)
	ListPendingTransactions(ctx context.Context, userID string) ([]PendingTransaction, error)
	ExpirePendingTransactions(ctx context.Context, now time.Time) (int64, error)
	GetInvoice(ctx context.Context, transactionID, requesterID string, isAdmin bool) (*Invoice, error)
	FindReferralAffiliate(ctx context.Context, code string) (string, error)
	InsertReferral(ctx context.Context, affiliateID, referredUserID string) error
}

// WebhookAuth membawa header webhook yang relevan. Xendit memakai Token
// (x-callback-token); adapter HMAC lokal memakai Timestamp/Signature.
type WebhookAuth struct {
	Token     string
	Timestamp string
	Signature string
}

// PaymentSettingsInput adalah input admin untuk konfigurasi integrasi
// pembayaran. Credential kosong berarti "pertahankan nilai tersimpan".
// Pointer URL membedakan field yang tidak dikirim dari redirect kosong yang
// sengaja dihapus.
type PaymentSettingsInput struct {
	Provider           string  `json:"provider"`
	SecretKey          string  `json:"secret_key"`
	WebhookToken       string  `json:"webhook_token"`
	BaseURL            *string `json:"base_url"`
	SuccessRedirectURL *string `json:"success_redirect_url"`
	FailureRedirectURL *string `json:"failure_redirect_url"`
}

// PaymentSettings adalah tampilan aman konfigurasi pembayaran untuk admin;
// nilai rahasia tidak pernah dikembalikan secara utuh, hanya flag + mask.
type PaymentSettings struct {
	Provider               string    `json:"provider"`
	Environment            string    `json:"environment"`
	SecretKeyConfigured    bool      `json:"secret_key_configured"`
	SecretKeyMasked        string    `json:"secret_key_masked,omitempty"`
	WebhookTokenConfigured bool      `json:"webhook_token_configured"`
	WebhookTokenMasked     string    `json:"webhook_token_masked,omitempty"`
	BaseURL                string    `json:"base_url"`
	SuccessRedirectURL     string    `json:"success_redirect_url"`
	FailureRedirectURL     string    `json:"failure_redirect_url"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// EncryptedPaymentSettings adalah bentuk penyimpanan (row DB) sebelum
// didekripsi oleh service.
type EncryptedPaymentSettings struct {
	SecretKeyCiphertext    []byte
	WebhookTokenCiphertext []byte
	BaseURL                string
	SuccessRedirectURL     string
	FailureRedirectURL     string
	UpdatedAt              time.Time
}

type PaymentSettingsStore interface {
	GetPaymentSettings(ctx context.Context) (*EncryptedPaymentSettings, error)
	UpsertPaymentSettings(ctx context.Context, settings *EncryptedPaymentSettings) error
}

// PaymentGateway adalah seam provider pembayaran. Penerapan bertanggung jawab
// membuat URL invoice dan menerjemahkan webhook provider ke representasi
// kanonis aplikasi.
type PaymentGateway interface {
	CreatePaymentSession(ctx context.Context, transaction Transaction) (*PaymentSession, error)
	CreateRefund(ctx context.Context, transaction Transaction, reason string) (*PaymentRefund, error)
	VerifyWebhook(rawBody []byte, auth WebhookAuth, now time.Time) error
	ParseWebhook(rawBody []byte) (*PaymentWebhookRequest, error)
}

type PaymentRefundService interface {
	CreateRefund(ctx context.Context, transaction Transaction, reason string) (*PaymentRefund, error)
}

type PaymentService interface {
	ListPackages(ctx context.Context, page, perPage int) ([]Package, error)
	ListMyPackages(ctx context.Context, userID string) ([]OwnedPackage, error)
	GetPricingPolicy(ctx context.Context, userID string) (*UserPricingPolicy, error)
	ClaimFreePackage(ctx context.Context, userID, packageID string) (*OwnedPackage, error)
	Checkout(ctx context.Context, userID, idempotencyKey string, input CheckoutRequest) (*CheckoutResponse, error)
	HandleWebhook(ctx context.Context, rawBody []byte, auth WebhookAuth) (bool, error)
	GetPaymentSettings(ctx context.Context) (*PaymentSettings, error)
	UpdatePaymentSettings(ctx context.Context, input PaymentSettingsInput) (*PaymentSettings, error)
	TrackPackageView(ctx context.Context, packageID, visitorKey string) (int64, error)
	ListMyTransactions(ctx context.Context, userID string) ([]AdminTransaction, error)
	ListPendingTransactions(ctx context.Context, userID string) ([]PendingTransaction, error)
	ExpirePendingTransactions(ctx context.Context) (int64, error)
	GenerateInvoicePDF(ctx context.Context, transactionID, requesterID string, isAdmin bool) ([]byte, string, error)
}
