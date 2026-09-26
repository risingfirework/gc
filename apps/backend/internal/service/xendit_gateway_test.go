package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

const xenditTestSecretKey = "xnd_development_secret_key_for_tests"
const xenditTestWebhookToken = "callback-secret-token-for-tests"

func xenditTestTransaction() domain.Transaction {
	expires := time.Now().Add(30 * time.Minute)
	return domain.Transaction{ID: "11111111-1111-1111-1111-111111111111", InvoiceNumber: "TKA-20260921-120000-ABCDEF12", Amount: 50000, ExpiresAt: &expires}
}

func TestXenditCreatePaymentSession(t *testing.T) {
	var path, auth string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"payment_session_id":"ps-1","payment_link_url":"https://checkout.xendit.test/abc"}`))
	}))
	defer server.Close()
	gateway := NewXenditGateway(xenditTestSecretKey, xenditTestWebhookToken, server.URL, "https://app.test/success", "https://app.test/cancel")
	session, err := gateway.CreatePaymentSession(context.Background(), xenditTestTransaction())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/sessions" || session.ID != "ps-1" || session.URL != "https://checkout.xendit.test/abc" {
		t.Fatalf("unexpected session: path=%s result=%+v", path, session)
	}
	if auth != "Basic "+base64.StdEncoding.EncodeToString([]byte(xenditTestSecretKey+":")) {
		t.Fatalf("unexpected auth: %s", auth)
	}
	if body["reference_id"] != "TKA-20260921-120000-ABCDEF12" || body["session_type"] != "PAY" || body["mode"] != "PAYMENT_LINK" || body["country"] != "ID" || body["capture_method"] != "AUTOMATIC" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if _, ok := body["allowed_payment_channels"]; ok {
		t.Fatal("channels must be selected from activated Xendit channels")
	}
}

func TestXenditCreateRefund(t *testing.T) {
	var path string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"id":"rfd-1","status":"PENDING"}`))
	}))
	defer server.Close()
	gateway := NewXenditGateway(xenditTestSecretKey, xenditTestWebhookToken, server.URL, "", "")
	tx := xenditTestTransaction()
	requestID := "pr-1"
	tx.ProviderPaymentRequestID = &requestID
	refund, err := gateway.CreateRefund(context.Background(), tx, "permintaan pengguna")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/refunds" || refund.ID != "rfd-1" || refund.Status != "pending" || body["payment_request_id"] != "pr-1" {
		t.Fatalf("unexpected refund: path=%s result=%+v body=%#v", path, refund, body)
	}
}

func TestXenditWebhookAuthentication(t *testing.T) {
	g := NewXenditGateway("key", xenditTestWebhookToken, "https://api.xendit.test", "", "")
	if err := g.VerifyWebhook(nil, domain.WebhookAuth{Token: xenditTestWebhookToken}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := g.VerifyWebhook(nil, domain.WebhookAuth{Token: "wrong"}, time.Now()); err != domain.ErrInvalidWebhookSignature {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestXenditParseSessionAndRefundWebhooks(t *testing.T) {
	g := NewXenditGateway("key", xenditTestWebhookToken, "https://api.xendit.test", "", "")
	session := []byte(`{"event":"payment_session.completed","created":"2026-09-21T05:20:00Z","data":{"payment_session_id":"ps-1","payment_request_id":"pr-1","reference_id":"TKA-1","amount":50000,"updated":"2026-09-21T05:10:00Z"}}`)
	event, err := g.ParseWebhook(session)
	if err != nil {
		t.Fatal(err)
	}
	if event.EventID != "ps-1:payment_session.completed" || event.PaymentStatus != "paid" || event.ProviderPaymentRequestID != "pr-1" || event.PaidAt == nil {
		t.Fatalf("unexpected event: %+v", event)
	}
	refund := []byte(`{"event":"refund.succeeded","data":{"id":"rfd-1","payment_request_id":"pr-1","reference_id":"TKA-1","amount":50000}}`)
	event, err = g.ParseWebhook(refund)
	if err != nil {
		t.Fatal(err)
	}
	if event.PaymentStatus != "refunded" || event.ProviderRefundID != "rfd-1" {
		t.Fatalf("unexpected refund event: %+v", event)
	}
}
