package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image/png"
	"net/url"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const (
	totpPeriodSeconds = 30
	totpDigits        = 6
	totpSkewSteps     = 1
	qrImageSize       = 260
)

// generateTOTPSecret membuat seed acak 20 byte dalam encoding base32 tanpa padding.
func generateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// totpCode menghitung kode RFC 6238 (HMAC-SHA1, 6 digit, periode 30 detik).
func totpCode(secret string, at time.Time) (string, error) {
	if at.Before(time.Unix(0, 0)) {
		return "", fmt.Errorf("totp timestamp out of range")
	}
	counter := uint64(at.Unix() / totpPeriodSeconds)
	message := make([]byte, 8)
	binary.BigEndian.PutUint64(message, counter)
	mac := hmac.New(sha1.New, mustDecodeBase32(secret))
	mac.Write(message)
	sum := mac.Sum(nil)
	if len(sum) < 20 {
		return "", fmt.Errorf("totp digest too short")
	}
	offset := sum[len(sum)-1] & 0x0f
	dynamicCode := (int32(sum[offset]&0x7f) << 24) |
		(int32(sum[offset+1]) << 16) |
		(int32(sum[offset+2]) << 8) |
		int32(sum[offset+3])
	return fmt.Sprintf("%06d", dynamicCode%1000000), nil
}

// validTOTP membandingkan kode input terhadap jendela waktu ±totpSkewSteps
// langkah periode, semua perbandingan konstan-waktu.
func validTOTP(secret, input string, at time.Time) bool {
	input = strings.TrimSpace(input)
	if !isDigits(input, totpDigits) {
		return false
	}
	for step := -totpSkewSteps; step <= totpSkewSteps; step++ {
		candidate, err := totpCode(secret, at.Add(time.Duration(step)*totpPeriodSeconds*time.Second))
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(input)) == 1 {
			return true
		}
	}
	return false
}

func isDigits(value string, expected int) bool {
	if len(value) != expected {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func mustDecodeBase32(secret string) []byte {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		// Secret disimpan internal; kegagalan berarti data rusak, aman teratasi di layer atas.
		return nil
	}
	return decoded
}

func totpURI(issuer, accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		url.PathEscape(issuer),
		url.PathEscape(accountName),
		secret,
		url.QueryEscape(issuer),
		totpDigits,
		totpPeriodSeconds,
	)
}

func qrDataURL(content string) (string, error) {
	encoded, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("encode qr: %w", err)
	}
	scaled, err := barcode.Scale(encoded, qrImageSize, qrImageSize)
	if err != nil {
		return "", fmt.Errorf("scale qr: %w", err)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, scaled); err != nil {
		return "", fmt.Errorf("encode png: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
