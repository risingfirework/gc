package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

type fakePaymentRepo struct {
	packages        []domain.Package
	pkgOwned        bool
	pendingCheckout bool
	affiliates      map[string]string // code → affiliateID
	inserted        [][2]string       // (affiliateID, referredUserID)
	referralError   error
}

func (f *fakePaymentRepo) ListPackages(_ context.Context, _, _ int) ([]domain.Package, error) {
	return f.packages, nil
}
func (f *fakePaymentRepo) GetPackage(_ context.Context, packageID string) (*domain.Package, error) {
	for _, p := range f.packages {
		if p.ID == packageID {
			return &p, nil
		}
	}
	return nil, domain.ErrPackageNotFound
}
func (f *fakePaymentRepo) ListMyPackages(_ context.Context, _ string) ([]domain.OwnedPackage, error) {
	if f.pkgOwned {
		return []domain.OwnedPackage{{Package: domain.Package{ID: "22222222-2222-2222-2222-222222222222"}}}, nil
	}
	return nil, nil
}
func (f *fakePaymentRepo) ClaimFreePackage(_ context.Context, _, _ string) (*domain.OwnedPackage, error) {
	return nil, nil
}
func (f *fakePaymentRepo) ReserveTransaction(_ context.Context, tx domain.Transaction, _ string) (*domain.Transaction, bool, error) {
	return &tx, true, nil
}
func (f *fakePaymentRepo) AttachPaymentSession(_ context.Context, txID string, session domain.PaymentSession) (*domain.Transaction, error) {
	url := session.URL
	expires := time.Now().Add(time.Hour)
	return &domain.Transaction{ID: txID, PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: ptrString("qris"), PaymentURL: &url, ExpiresAt: &expires}, nil
}
func (f *fakePaymentRepo) FailPaymentInitialization(context.Context, string) error { return nil }
func ptrString(v string) *string                                                   { return &v }
func (f *fakePaymentRepo) ProcessWebhook(_ context.Context, _ domain.PaymentWebhookRequest, _ string, _ time.Time) (bool, error) {
	return true, nil
}
func (f *fakePaymentRepo) GetUserFinancePolicy(_ context.Context, _ string) (float64, bool, error) {
	return 0, true, nil
}
func (f *fakePaymentRepo) TrackPackageView(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}
func (f *fakePaymentRepo) HasEverOwnedPackage(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (f *fakePaymentRepo) HasPendingCheckout(_ context.Context, _, _, _ string) (bool, error) {
	return f.pendingCheckout, nil
}
func (f *fakePaymentRepo) ListMyTransactions(_ context.Context, _ string) ([]domain.AdminTransaction, error) {
	return nil, nil
}
func (f *fakePaymentRepo) ListPendingTransactions(_ context.Context, _ string) ([]domain.PendingTransaction, error) {
	return nil, nil
}
func (f *fakePaymentRepo) ExpirePendingTransactions(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (f *fakePaymentRepo) GetInvoice(_ context.Context, _, _ string, _ bool) (*domain.Invoice, error) {
	return nil, nil
}
func (f *fakePaymentRepo) FindReferralAffiliate(_ context.Context, code string) (string, error) {
	if f.referralError != nil {
		return "", f.referralError
	}
	if id, ok := f.affiliates[code]; ok {
		return id, nil
	}
	return "", domain.ErrReferralNotFound
}
func (f *fakePaymentRepo) InsertReferral(_ context.Context, affiliateID, referredUserID string) error {
	f.inserted = append(f.inserted, [2]string{affiliateID, referredUserID})
	return nil
}

type fakePaymentGateway struct{}

func (f *fakePaymentGateway) CreatePaymentSession(_ context.Context, _ domain.Transaction) (*domain.PaymentSession, error) {
	return &domain.PaymentSession{ID: "ps-1", URL: "https://pay.example.test/invoice"}, nil
}
func (f *fakePaymentGateway) CreateRefund(context.Context, domain.Transaction, string) (*domain.PaymentRefund, error) {
	return nil, domain.ErrTransactionNotRefundable
}
func (f *fakePaymentGateway) VerifyWebhook(_ []byte, _ domain.WebhookAuth, _ time.Time) error {
	return nil
}
func (f *fakePaymentGateway) ParseWebhook(_ []byte) (*domain.PaymentWebhookRequest, error) {
	return &domain.PaymentWebhookRequest{EventID: "evt-1", InvoiceNumber: "TKA-1", PaymentStatus: "paid", Amount: 1}, nil
}

func newCheckoutSvc(repo *fakePaymentRepo) *PaymentService {
	svc := &PaymentService{repository: repo, now: time.Now}
	svc.gateway = newPaymentGatewayGate(&fakePaymentGateway{})
	return svc
}

func TestCheckoutAppliesReferralCode(t *testing.T) {
	repo := &fakePaymentRepo{
		packages:   []domain.Package{{ID: "22222222-2222-2222-2222-222222222222", Price: 5000, Status: domain.StatusActive}},
		affiliates: map[string]string{"MITRA-ABC": "33333333-3333-3333-3333-333333333333"},
	}
	svc := newCheckoutSvc(repo)
	_, err := svc.Checkout(context.Background(), "11111111-1111-1111-1111-111111111111", "test-idempotency-key-abc12345", domain.CheckoutRequest{PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: "qris", ReferralCode: "MITRA-ABC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inserted) != 1 || repo.inserted[0] != [2]string{"33333333-3333-3333-3333-333333333333", "11111111-1111-1111-1111-111111111111"} {
		t.Fatalf("expected referral inserted for student-1, got %v", repo.inserted)
	}
}

func TestCheckoutRejectsUnknownReferralCode(t *testing.T) {
	repo := &fakePaymentRepo{
		packages: []domain.Package{{ID: "22222222-2222-2222-2222-222222222222", Price: 5000, Status: domain.StatusActive}},
	}
	svc := newCheckoutSvc(repo)
	_, err := svc.Checkout(context.Background(), "11111111-1111-1111-1111-111111111111", "test-idempotency-key-abc12345", domain.CheckoutRequest{PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: "qris", ReferralCode: "MITRA-MISSING"})
	if err != domain.ErrInvalidReferralCode {
		t.Fatalf("expected ErrInvalidReferralCode, got %v", err)
	}
}

func TestCheckoutIgnoresEmptyReferralCode(t *testing.T) {
	repo := &fakePaymentRepo{
		packages: []domain.Package{{ID: "22222222-2222-2222-2222-222222222222", Price: 5000, Status: domain.StatusActive}},
	}
	svc := newCheckoutSvc(repo)
	_, err := svc.Checkout(context.Background(), "11111111-1111-1111-1111-111111111111", "test-idempotency-key-abc12345", domain.CheckoutRequest{PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: "qris"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("expected no referral inserted, got %d", len(repo.inserted))
	}
}

func TestCheckoutRejectsFreePackage(t *testing.T) {
	repo := &fakePaymentRepo{
		packages: []domain.Package{{ID: "22222222-2222-2222-2222-222222222222", Price: 0, Status: domain.StatusActive}},
	}
	svc := newCheckoutSvc(repo)
	_, err := svc.Checkout(context.Background(), "11111111-1111-1111-1111-111111111111", "test-idempotency-key-abc12345", domain.CheckoutRequest{PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: "qris"})
	if err != domain.ErrNotFreePackage {
		t.Fatalf("expected ErrNotFreePackage, got %v", err)
	}
}

func TestCheckoutRejectsDuplicatePendingPackage(t *testing.T) {
	repo := &fakePaymentRepo{
		packages:        []domain.Package{{ID: "22222222-2222-2222-2222-222222222222", Price: 5000, Status: domain.StatusActive}},
		pendingCheckout: true,
	}
	svc := newCheckoutSvc(repo)
	_, err := svc.Checkout(context.Background(), "11111111-1111-1111-1111-111111111111", "test-idempotency-key-abc12345", domain.CheckoutRequest{PackageID: "22222222-2222-2222-2222-222222222222", PaymentMethod: "qris"})
	if err != domain.ErrPendingPaymentExists {
		t.Fatalf("expected ErrPendingPaymentExists, got %v", err)
	}
}

func TestHMACPaymentGatewayVerifyWebhook(t *testing.T) {
	secret := "01234567890123456789012345678901"
	gateway := NewHMACPaymentGateway(secret, "https://pay.example.test/checkout")
	now := time.Unix(1_800_000_000, 0).UTC()
	timestamp := "1800000000"
	body := []byte(`{"event_id":"evt-1"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	auth := domain.WebhookAuth{Timestamp: timestamp, Signature: signature}
	if err := gateway.VerifyWebhook(body, auth, now); err != nil {
		t.Fatalf("expected valid signature: %v", err)
	}
	if err := gateway.VerifyWebhook([]byte(`{}`), auth, now); err != domain.ErrInvalidWebhookSignature {
		t.Fatalf("expected invalid signature, got %v", err)
	}
}

func TestHMACPaymentGatewayParseWebhookStrict(t *testing.T) {
	gateway := NewHMACPaymentGateway("01234567890123456789012345678901", "https://pay.example.test/checkout")
	event, err := gateway.ParseWebhook([]byte(`{"event_id":"evt-1","invoice_number":"TKA-1","payment_status":"paid","amount":5000}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.EventID != "evt-1" || event.InvoiceNumber != "TKA-1" || event.PaymentStatus != "paid" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if _, err := gateway.ParseWebhook([]byte(`{"event_id":"evt-1","invoice_number":"TKA-1","payment_status":"paid","amount":5000,"extra":"field"}`)); err != domain.ErrInvalidPayment {
		t.Fatalf("expected reject unknown field, got %v", err)
	}
}
