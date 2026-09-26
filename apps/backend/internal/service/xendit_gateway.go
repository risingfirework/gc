package service

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tka/apps/backend/internal/domain"
)

// XenditGateway uses Xendit's current Payment Sessions and Refund APIs.
type XenditGateway struct {
	secretKey, webhookToken, baseURL, successURL, failureURL string
	httpClient                                               *http.Client
	now                                                      func() time.Time
}

func NewXenditGateway(secretKey, webhookToken, baseURL, successURL, failureURL string) *XenditGateway {
	return &XenditGateway{
		secretKey: strings.TrimSpace(secretKey), webhookToken: strings.TrimSpace(webhookToken),
		baseURL: normalizeXenditBaseURL(baseURL), successURL: strings.TrimSpace(successURL), failureURL: strings.TrimSpace(failureURL),
		httpClient: &http.Client{Timeout: 15 * time.Second}, now: time.Now,
	}
}

func normalizeXenditBaseURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" || value == "https://api.xendit.co/v2" {
		return "https://api.xendit.co"
	}
	return value
}

const xenditResponseBodyLimit = 1 << 20

type xenditSessionRequest struct {
	ReferenceID      string  `json:"reference_id"`
	SessionType      string  `json:"session_type"`
	Mode             string  `json:"mode"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Country          string  `json:"country"`
	CaptureMethod    string  `json:"capture_method"`
	ExpiresAt        string  `json:"expires_at"`
	Description      string  `json:"description"`
	SuccessReturnURL string  `json:"success_return_url,omitempty"`
	CancelReturnURL  string  `json:"cancel_return_url,omitempty"`
}

type xenditSessionResponse struct {
	ID               string `json:"payment_session_id"`
	PaymentRequestID string `json:"payment_request_id"`
	URL              string `json:"payment_link_url"`
}

func (g *XenditGateway) CreatePaymentSession(ctx context.Context, transaction domain.Transaction) (*domain.PaymentSession, error) {
	if g.secretKey == "" {
		return nil, fmt.Errorf("xendit secret key tidak dikonfigurasi")
	}
	if transaction.Amount <= 0 || transaction.ExpiresAt == nil {
		return nil, domain.ErrInvalidPayment
	}
	request := xenditSessionRequest{
		ReferenceID: transaction.InvoiceNumber, SessionType: "PAY", Mode: "PAYMENT_LINK",
		Amount: transaction.Amount, Currency: "IDR", Country: "ID",
		CaptureMethod:    "AUTOMATIC",
		ExpiresAt:        transaction.ExpiresAt.UTC().Format(time.RFC3339),
		Description:      "Pembayaran paket TKA " + transaction.InvoiceNumber,
		SuccessReturnURL: g.successURL, CancelReturnURL: g.failureURL,
	}
	var response xenditSessionResponse
	if err := g.doJSON(ctx, http.MethodPost, "/sessions", request, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.ID) == "" || strings.TrimSpace(response.URL) == "" {
		return nil, fmt.Errorf("xendit session response tidak lengkap")
	}
	return &domain.PaymentSession{ID: response.ID, PaymentRequestID: response.PaymentRequestID, URL: response.URL}, nil
}

type xenditRefundRequest struct {
	ReferenceID      string            `json:"reference_id"`
	PaymentRequestID string            `json:"payment_request_id"`
	Currency         string            `json:"currency"`
	Amount           float64           `json:"amount"`
	Reason           string            `json:"reason"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}
type xenditRefundResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (g *XenditGateway) CreateRefund(ctx context.Context, transaction domain.Transaction, reason string) (*domain.PaymentRefund, error) {
	if g.secretKey == "" {
		return nil, fmt.Errorf("xendit secret key tidak dikonfigurasi")
	}
	if transaction.ProviderPaymentRequestID == nil || strings.TrimSpace(*transaction.ProviderPaymentRequestID) == "" || transaction.Amount <= 0 {
		return nil, domain.ErrTransactionNotRefundable
	}
	request := xenditRefundRequest{
		ReferenceID: transaction.InvoiceNumber, PaymentRequestID: *transaction.ProviderPaymentRequestID,
		Currency: "IDR", Amount: transaction.Amount, Reason: "REQUESTED_BY_CUSTOMER",
		Metadata: map[string]string{"transaction_id": transaction.ID, "admin_reason": truncate(reason, 500)},
	}
	var response xenditRefundResponse
	if err := g.doJSON(ctx, http.MethodPost, "/refunds", request, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, fmt.Errorf("xendit refund response tanpa id")
	}
	return &domain.PaymentRefund{ID: response.ID, Status: strings.ToLower(response.Status)}, nil
}

func (g *XenditGateway) doJSON(ctx context.Context, method, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode xendit request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(g.secretKey+":")))
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call xendit: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, xenditResponseBodyLimit))
	if err != nil {
		return fmt.Errorf("read xendit response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("xendit returned %s: %s", resp.Status, truncate(string(responseBody), 200))
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode xendit response: %w", err)
	}
	return nil
}

func (g *XenditGateway) VerifyWebhook(_ []byte, auth domain.WebhookAuth, _ time.Time) error {
	if g.webhookToken == "" || len(auth.Token) != len(g.webhookToken) || subtle.ConstantTimeCompare([]byte(auth.Token), []byte(g.webhookToken)) != 1 {
		return domain.ErrInvalidWebhookSignature
	}
	return nil
}

type xenditWebhookPayload struct {
	Event   string `json:"event"`
	Created string `json:"created"`
	Data    struct {
		ID                string  `json:"id"`
		PaymentSessionID  string  `json:"payment_session_id"`
		PaymentRequestID  string  `json:"payment_request_id"`
		ReferenceID       string  `json:"reference_id"`
		Amount            float64 `json:"amount"`
		PaymentMethodType string  `json:"payment_method_type"`
		Updated           string  `json:"updated"`
	} `json:"data"`
}

func (g *XenditGateway) ParseWebhook(rawBody []byte) (*domain.PaymentWebhookRequest, error) {
	var payload xenditWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, domain.ErrInvalidPayment
	}
	eventName := strings.ToLower(strings.TrimSpace(payload.Event))
	status := ""
	switch eventName {
	case "payment_session.completed":
		status = "paid"
	case "payment_session.expired":
		status = "expired"
	case "refund.succeeded":
		status = "refunded"
	case "refund.failed":
		status = "paid"
	default:
		return nil, domain.ErrInvalidPayment
	}
	providerID := firstNonEmpty(payload.Data.PaymentSessionID, payload.Data.ID)
	event := &domain.PaymentWebhookRequest{
		EventID: providerID + ":" + eventName, InvoiceNumber: strings.TrimSpace(payload.Data.ReferenceID),
		PaymentStatus: status, PaymentMethod: mapXenditPaymentMethod(payload.Data.PaymentMethodType), Amount: payload.Data.Amount,
		ProviderSessionID: strings.TrimSpace(payload.Data.PaymentSessionID), ProviderPaymentRequestID: strings.TrimSpace(payload.Data.PaymentRequestID),
	}
	if strings.HasPrefix(eventName, "refund.") {
		event.ProviderRefundID = strings.TrimSpace(payload.Data.ID)
	}
	if eventName == "payment_session.completed" {
		// data.updated mencerminkan perubahan status sesi. Top-level created
		// adalah waktu delivery attempt dan tidak aman untuk cek kedaluwarsa.
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.Data.Updated)); err == nil {
			event.PaidAt = &parsed
		}
	}
	if providerID == "" || event.InvoiceNumber == "" {
		return nil, domain.ErrInvalidPayment
	}
	return event, nil
}

func mapXenditPaymentMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "QR_CODE":
		return "qris"
	case "VIRTUAL_ACCOUNT", "BANK_TRANSFER":
		return "virtual_account"
	case "EWALLET":
		return "e_wallet"
	default:
		return ""
	}
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

var _ domain.PaymentGateway = (*XenditGateway)(nil)
