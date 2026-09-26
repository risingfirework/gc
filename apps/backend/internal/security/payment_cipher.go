// Package security berisi utilitas kriptografi untuk data sensitif aplikasi.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// PaymentCipher mengenkripsi credential pembayaran (mis. secret key Xendit)
// sebelum disimpan ke database. Kunci diturunkan dari master key runtime
// (JWT secret) sehingga tidak ada kunci tersimpan dalam penyimpanan.
type PaymentCipher struct {
	aead cipher.AEAD
}

func NewPaymentCipher(masterKey string) (*PaymentCipher, error) {
	if len(masterKey) < 32 {
		return nil, fmt.Errorf("payment cipher master key must contain at least 32 characters")
	}
	derived := sha256.Sum256([]byte(masterKey))
	block, err := aes.NewCipher(derived[:])
	if err != nil {
		return nil, fmt.Errorf("create payment cipher block: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create payment cipher aead: %w", err)
	}
	return &PaymentCipher{aead: aead}, nil
}

// Encrypt mengenkripsi plaintext (AES-256-GCM). Nilai kosong dienkripsi
// menjadi ciphertext kosong untuk memudahkan merge "jangan ubah" di lapisan
// service.
func (c *PaymentCipher) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read cipher nonce: %w", err)
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *PaymentCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("payment ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payment ciphertext: %w", err)
	}
	return plaintext, nil
}
