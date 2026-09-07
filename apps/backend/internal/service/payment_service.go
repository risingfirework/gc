package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"tka/apps/backend/internal/domain"
)

const webhookTolerance = 5 * time.Minute

type HMACPaymentGateway struct {
	secret      []byte
	checkoutURL string
}

func NewHMACPaymentGateway(secret, checkoutURL string) *HMACPaymentGateway {
	return &HMACPaymentGateway{secret: []byte(secret), checkoutURL: strings.TrimRight(checkoutURL, "/")}
}

func (g *HMACPaymentGateway) CreatePaymentURL(transaction domain.Transaction) (string, error) {
	base, err := url.Parse(g.checkoutURL)
	if err != nil {
		return "", fmt.Errorf("parse payment checkout URL: %w", err)
	}
	localHTTP := base.Scheme == "http" && (base.Hostname() == "localhost" || base.Hostname() == "127.0.0.1")
	if base.Scheme != "https" && !localHTTP {
		return "", fmt.Errorf("payment checkout URL must use https outside localhost")
	}
	amount := strconv.FormatFloat(transaction.Amount, 'f', 2, 64)
	mac := hmac.New(sha256.New, g.secret)
	_, _ = mac.Write([]byte(transaction.InvoiceNumber + "." + amount))
	query := base.Query()
	query.Set("invoice_number", transaction.InvoiceNumber)
	query.Set("amount", amount)
	query.Set("signature", hex.EncodeToString(mac.Sum(nil)))
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (g *HMACPaymentGateway) VerifyWebhook(rawBody []byte, timestamp, signature string, now time.Time) error {
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return domain.ErrInvalidWebhookSignature
	}
	sentAt := time.Unix(unix, 0)
	if delta := now.Sub(sentAt); delta > webhookTolerance || delta < -webhookTolerance {
		return domain.ErrInvalidWebhookSignature
	}
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return domain.ErrInvalidWebhookSignature
	}
	mac := hmac.New(sha256.New, g.secret)
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(rawBody)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return domain.ErrInvalidWebhookSignature
	}
	return nil
}

type PaymentService struct {
	repository domain.PaymentRepository
	gateway    domain.PaymentGateway
	now        func() time.Time
}

func NewPaymentService(repository domain.PaymentRepository, gateway domain.PaymentGateway) *PaymentService {
	return &PaymentService{repository: repository, gateway: gateway, now: time.Now}
}

func (s *PaymentService) ListPackages(ctx context.Context, page, perPage int) ([]domain.Package, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return s.repository.ListPackages(ctx, perPage, (page-1)*perPage)
}

func (s *PaymentService) TrackPackageView(ctx context.Context, packageID, visitorKey string) (int64, error) {
	if !validUUID(packageID) || !validUUID(visitorKey) {
		return 0, domain.ErrInvalidPayment
	}
	return s.repository.TrackPackageView(ctx, packageID, visitorKey)
}

func (s *PaymentService) ListMyPackages(ctx context.Context, userID string) ([]domain.OwnedPackage, error) {
	if !validUUID(userID) {
		return nil, domain.ErrInvalidPayment
	}
	items, err := s.repository.ListMyPackages(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, domain.ErrNoPurchasedPackage
	}
	return items, nil
}

func (s *PaymentService) GetPricingPolicy(ctx context.Context, userID string) (*domain.UserPricingPolicy, error) {
	if !validUUID(userID) {
		return nil, domain.ErrInvalidPayment
	}
	discountPercent, accountActive, err := s.repository.GetUserFinancePolicy(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.UserPricingPolicy{DiscountPercent: discountPercent, AccountActive: accountActive}, nil
}

func (s *PaymentService) ClaimFreePackage(ctx context.Context, userID, packageID string) (*domain.OwnedPackage, error) {
	if !validUUID(userID) || !validUUID(packageID) {
		return nil, domain.ErrInvalidPayment
	}
	item, err := s.repository.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if item.Status != domain.StatusActive {
		return nil, domain.ErrPackageNotFound
	}
	if item.Price != 0 {
		return nil, domain.ErrNotFreePackage
	}
	if err := s.ensurePackageNotOwned(ctx, userID, packageID); err != nil {
		return nil, err
	}
	return s.repository.ClaimFreePackage(ctx, userID, packageID)
}

func (s *PaymentService) Checkout(ctx context.Context, userID, idempotencyKey string, input domain.CheckoutRequest) (*domain.CheckoutResponse, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.PaymentMethod = strings.ToLower(strings.TrimSpace(input.PaymentMethod))
	if !validUUID(userID) || !validUUID(input.PackageID) || len(idempotencyKey) < 16 || len(idempotencyKey) > 128 || !validPaymentMethod(input.PaymentMethod) {
		return nil, domain.ErrInvalidPayment
	}
	item, err := s.repository.GetPackage(ctx, input.PackageID)
	if err != nil {
		return nil, err
	}
	if item.Status != domain.StatusActive {
		return nil, domain.ErrPackageNotFound
	}
	if err := s.ensurePackageNotOwned(ctx, userID, input.PackageID); err != nil {
		return nil, err
	}
	discountPercent, accountActive, err := s.repository.GetUserFinancePolicy(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !accountActive {
		return nil, domain.ErrInvalidPayment
	}
	discountedAmount := math.Round(item.Price*(100-discountPercent)) / 100
	now := s.now().UTC()
	expiresAt := now.Add(30 * time.Minute)
	method := input.PaymentMethod
	transaction := domain.Transaction{
		ID: uuid.NewString(), UserID: userID, PackageID: item.ID,
		InvoiceNumber: "TKA-" + now.Format("20060102-150405") + "-" + strings.ToUpper(uuid.NewString()[:8]),
		Amount:        discountedAmount, PaymentStatus: "pending", PaymentMethod: &method, ExpiresAt: &expiresAt, CreatedAt: now,
	}
	paymentURL, err := s.gateway.CreatePaymentURL(transaction)
	if err != nil {
		return nil, err
	}
	transaction.PaymentURL = &paymentURL
	created, err := s.repository.CreateOrGetTransaction(ctx, transaction, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if created.PaymentURL == nil || created.ExpiresAt == nil {
		return nil, fmt.Errorf("stored checkout is incomplete")
	}
	if created.PackageID != input.PackageID || created.PaymentMethod == nil || *created.PaymentMethod != input.PaymentMethod {
		return nil, domain.ErrInvalidPayment
	}
	return &domain.CheckoutResponse{Transaction: *created, PaymentURL: *created.PaymentURL, ExpiresAt: *created.ExpiresAt}, nil
}

func (s *PaymentService) ensurePackageNotOwned(ctx context.Context, userID, packageID string) error {
	items, err := s.repository.ListMyPackages(ctx, userID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == packageID {
			return domain.ErrPackageAlreadyOwned
		}
	}
	return nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, rawBody []byte, timestamp, signature string) (bool, error) {
	now := s.now().UTC()
	if err := s.gateway.VerifyWebhook(rawBody, timestamp, signature, now); err != nil {
		return false, err
	}
	var event domain.PaymentWebhookRequest
	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return false, domain.ErrInvalidPayment
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return false, domain.ErrInvalidPayment
	}
	event.EventID = strings.TrimSpace(event.EventID)
	event.InvoiceNumber = strings.TrimSpace(event.InvoiceNumber)
	event.PaymentStatus = strings.ToLower(strings.TrimSpace(event.PaymentStatus))
	event.PaymentMethod = strings.ToLower(strings.TrimSpace(event.PaymentMethod))
	if event.EventID == "" || len(event.EventID) > 200 || event.InvoiceNumber == "" || event.Amount < 0 || !validPaymentStatus(event.PaymentStatus) {
		return false, domain.ErrInvalidPayment
	}
	digest := sha256.Sum256(rawBody)
	return s.repository.ProcessWebhook(ctx, event, hex.EncodeToString(digest[:]), now)
}

func validPaymentMethod(value string) bool {
	return value == "qris" || value == "virtual_account" || value == "e_wallet"
}
func validPaymentStatus(value string) bool {
	return value == "paid" || value == "failed" || value == "expired" || value == "refunded"
}

var _ domain.PaymentGateway = (*HMACPaymentGateway)(nil)
var _ domain.PaymentService = (*PaymentService)(nil)
