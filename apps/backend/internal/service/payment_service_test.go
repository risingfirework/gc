package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

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
	if err := gateway.VerifyWebhook(body, timestamp, signature, now); err != nil {
		t.Fatalf("expected valid signature: %v", err)
	}
	if err := gateway.VerifyWebhook([]byte(`{}`), timestamp, signature, now); err != domain.ErrInvalidWebhookSignature {
		t.Fatalf("expected invalid signature, got %v", err)
	}
}
