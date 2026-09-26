package security

import (
	"bytes"
	"testing"
)

func TestPaymentCipherRoundTrip(t *testing.T) {
	cipherBox, err := NewPaymentCipher("0123456789012345678901234567890123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	secrets := []string{"xnd_development_sk_secret_1234567890", "webhook-token-panjang-untuk-uji", ""}
	for _, value := range secrets {
		encrypted, err := cipherBox.Encrypt([]byte(value))
		if err != nil {
			t.Fatalf("encrypt %q: %v", value, err)
		}
		decrypted, err := cipherBox.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decrypt %q: %v", value, err)
		}
		if !bytes.Equal(decrypted, []byte(value)) {
			t.Fatalf("round trip mismatch for %q", value)
		}
	}
}

func TestPaymentCipherNonDeterministic(t *testing.T) {
	cipherBox, _ := NewPaymentCipher("0123456789012345678901234567890123456789")
	first, _ := cipherBox.Encrypt([]byte("rahasia-berulang"))
	second, _ := cipherBox.Encrypt([]byte("rahasia-berulang"))
	if bytes.Equal(first, second) {
		t.Fatalf("expected random nonce to produce distinct ciphertext")
	}
}

func TestPaymentCipherTamperDetected(t *testing.T) {
	cipherBox, _ := NewPaymentCipher("0123456789012345678901234567890123456789")
	encrypted, _ := cipherBox.Encrypt([]byte("rahasia"))
	encrypted[len(encrypted)-1] ^= 0x01
	if _, err := cipherBox.Decrypt(encrypted); err == nil {
		t.Fatalf("expected tamper detection")
	}
}

func TestPaymentCipherWeakMasterKey(t *testing.T) {
	if _, err := NewPaymentCipher("pendek"); err == nil {
		t.Fatalf("expected error for short master key")
	}
}

func TestPaymentCipherDecryptGarbage(t *testing.T) {
	cipherBox, _ := NewPaymentCipher("0123456789012345678901234567890123456789")
	if _, err := cipherBox.Decrypt([]byte{0x01, 0x02}); err == nil {
		t.Fatalf("expected error for short ciphertext")
	}
}
