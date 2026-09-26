package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"tka/apps/backend/internal/domain"
)

const defaultXenditBaseURL = "https://api.xendit.co"

// GetPaymentSettings mengembalikan tampilan ter-mask dari konfigurasi
// pembayaran. Nilai rahasia (secret key & webhook token) tidak pernah
// dikembalikan secara utuh.
func (s *PaymentService) GetPaymentSettings(ctx context.Context) (*domain.PaymentSettings, error) {
	if s.settingsStore == nil || s.cipher == nil {
		return nil, domain.ErrInvalidPayment
	}
	current := &domain.EncryptedPaymentSettings{}
	stored, err := s.settingsStore.GetPaymentSettings(ctx)
	switch {
	case err == nil:
		current = stored
	case errors.Is(err, domain.ErrPaymentSettingsNotFound):
		// Belum tersimpan: tampilkan nilai environment.
	default:
		return nil, err
	}
	secret, err := s.effectiveSecret(current.SecretKeyCiphertext, "", s.envDefaults.SecretKey)
	if err != nil {
		return nil, err
	}
	token, err := s.effectiveSecret(current.WebhookTokenCiphertext, "", s.envDefaults.WebhookToken)
	if err != nil {
		return nil, err
	}
	storedConfigured := hasStoredPaymentConfig(current)
	merged := &domain.EncryptedPaymentSettings{
		BaseURL:            normalizeXenditBaseURL(firstNonEmpty(current.BaseURL, s.envDefaults.BaseURL, defaultXenditBaseURL)),
		SuccessRedirectURL: effectiveOptionalURL(nil, current.SuccessRedirectURL, s.envDefaults.SuccessRedirectURL, storedConfigured),
		FailureRedirectURL: effectiveOptionalURL(nil, current.FailureRedirectURL, s.envDefaults.FailureRedirectURL, storedConfigured),
		UpdatedAt:          current.UpdatedAt,
	}
	result := s.maskSettings(merged)
	result.SecretKeyConfigured = secret != ""
	result.SecretKeyMasked = maskedSecret(secret)
	result.WebhookTokenConfigured = token != ""
	result.WebhookTokenMasked = maskedSecret(token)
	return result, nil
}

// LoadPaymentSettings mengaktifkan konfigurasi efektif dari database saat
// proses aplikasi dimulai. Nilai environment tetap menjadi fallback untuk
// field yang belum pernah disimpan oleh owner.
func (s *PaymentService) LoadPaymentSettings(ctx context.Context) error {
	if s.settingsStore == nil || s.cipher == nil {
		return domain.ErrInvalidPayment
	}
	current := &domain.EncryptedPaymentSettings{}
	stored, err := s.settingsStore.GetPaymentSettings(ctx)
	switch {
	case err == nil:
		current = stored
	case errors.Is(err, domain.ErrPaymentSettingsNotFound):
		// Instalasi baru memakai nilai environment sampai owner menyimpan
		// konfigurasi pertamanya.
	default:
		return err
	}
	secret, err := s.effectiveSecret(current.SecretKeyCiphertext, "", s.envDefaults.SecretKey)
	if err != nil {
		return err
	}
	token, err := s.effectiveSecret(current.WebhookTokenCiphertext, "", s.envDefaults.WebhookToken)
	if err != nil {
		return err
	}
	storedConfigured := hasStoredPaymentConfig(current)
	baseURL := normalizeXenditBaseURL(firstNonEmpty(current.BaseURL, s.envDefaults.BaseURL, defaultXenditBaseURL))
	if err := validatePaymentBaseURL(baseURL, s.environment); err != nil {
		return err
	}
	successURL := effectiveOptionalURL(nil, current.SuccessRedirectURL, s.envDefaults.SuccessRedirectURL, storedConfigured)
	failureURL := effectiveOptionalURL(nil, current.FailureRedirectURL, s.envDefaults.FailureRedirectURL, storedConfigured)
	s.gateway.Set(NewXenditGateway(secret, token, baseURL, successURL, failureURL))
	return nil
}

// UpdatePaymentSettings menyimpan konfigurasi terenkripsi lalu
// mengganti gateway aktif secara atomik (hot-reload tanpa restart).
func (s *PaymentService) UpdatePaymentSettings(ctx context.Context, input domain.PaymentSettingsInput) (*domain.PaymentSettings, error) {
	if s.settingsStore == nil || s.cipher == nil {
		return nil, domain.ErrInvalidPayment
	}
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	if provider == "" {
		provider = "xendit"
	}
	if provider != "xendit" {
		return nil, fmt.Errorf("provider pembayaran %q tidak didukung", provider)
	}
	current := &domain.EncryptedPaymentSettings{}
	stored, err := s.settingsStore.GetPaymentSettings(ctx)
	switch {
	case err == nil:
		current = stored
	case errors.Is(err, domain.ErrPaymentSettingsNotFound):
		// Belum pernah disimpan: mulai dari kosong.
	default:
		return nil, err
	}

	storedConfigured := hasStoredPaymentConfig(current)
	baseURL := normalizeXenditBaseURL(effectiveBaseURL(input.BaseURL, current.BaseURL, s.envDefaults.BaseURL))
	successURL := effectiveOptionalURL(input.SuccessRedirectURL, current.SuccessRedirectURL, s.envDefaults.SuccessRedirectURL, storedConfigured)
	failureURL := effectiveOptionalURL(input.FailureRedirectURL, current.FailureRedirectURL, s.envDefaults.FailureRedirectURL, storedConfigured)
	for label, value := range map[string]string{"base url": baseURL, "redirect sukses": successURL, "redirect gagal": failureURL} {
		if err := validatePaymentEndpointURL(value); err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
	}
	if err := validatePaymentBaseURL(baseURL, s.environment); err != nil {
		return nil, fmt.Errorf("base url: %w", err)
	}

	secret, err := s.effectiveSecret(current.SecretKeyCiphertext, input.SecretKey, s.envDefaults.SecretKey)
	if err != nil {
		return nil, err
	}
	token, err := s.effectiveSecret(current.WebhookTokenCiphertext, input.WebhookToken, s.envDefaults.WebhookToken)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, domain.ErrInvalidPayment
	}
	if len(token) < 32 {
		return nil, fmt.Errorf("webhook token minimal 32 karakter")
	}
	secretCiphertext, err := s.cipher.Encrypt([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("enkripsi secret key: %w", err)
	}
	tokenCiphertext, err := s.cipher.Encrypt([]byte(token))
	if err != nil {
		return nil, fmt.Errorf("enkripsi webhook token: %w", err)
	}

	persisted := &domain.EncryptedPaymentSettings{
		SecretKeyCiphertext:    secretCiphertext,
		WebhookTokenCiphertext: tokenCiphertext,
		BaseURL:                baseURL,
		SuccessRedirectURL:     successURL,
		FailureRedirectURL:     failureURL,
	}
	if err := s.settingsStore.UpsertPaymentSettings(ctx, persisted); err != nil {
		return nil, err
	}
	s.gateway.Set(NewXenditGateway(secret, token, baseURL, successURL, failureURL))
	result := s.maskSettings(persisted)
	result.SecretKeyConfigured = secret != ""
	result.SecretKeyMasked = maskedSecret(secret)
	result.WebhookTokenConfigured = token != ""
	result.WebhookTokenMasked = maskedSecret(token)
	return result, nil
}

func (s *PaymentService) maskSettings(settings *domain.EncryptedPaymentSettings) *domain.PaymentSettings {
	return &domain.PaymentSettings{
		Provider:           "xendit",
		Environment:        s.environment,
		BaseURL:            settings.BaseURL,
		SuccessRedirectURL: settings.SuccessRedirectURL,
		FailureRedirectURL: settings.FailureRedirectURL,
		UpdatedAt:          settings.UpdatedAt,
	}
}

func (s *PaymentService) effectiveSecret(ciphertext []byte, input, envFallback string) (string, error) {
	if value := strings.TrimSpace(input); value != "" {
		return value, nil
	}
	if len(ciphertext) > 0 {
		plain, err := s.cipher.Decrypt(ciphertext)
		if err != nil {
			return "", fmt.Errorf("dekripsi credential pembayaran: %w", err)
		}
		return string(plain), nil
	}
	return envFallback, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func effectiveBaseURL(input *string, stored, envFallback string) string {
	if input != nil {
		return firstNonEmpty(*input, envFallback, defaultXenditBaseURL)
	}
	return firstNonEmpty(stored, envFallback, defaultXenditBaseURL)
}

// effectiveOptionalURL membedakan field yang tidak dikirim (pertahankan
// nilai lama) dari string kosong yang sengaja dikirim (hapus redirect).
func effectiveOptionalURL(input *string, stored, envFallback string, storedConfigured bool) string {
	if input != nil {
		return strings.TrimSpace(*input)
	}
	if storedConfigured {
		return strings.TrimSpace(stored)
	}
	return firstNonEmpty(stored, envFallback)
}

func hasStoredPaymentConfig(settings *domain.EncryptedPaymentSettings) bool {
	// UpdatePaymentSettings selalu mensyaratkan dan mengenkripsi webhook token.
	// Keberadaan ciphertext ini membedakan row bootstrap migrasi dari
	// konfigurasi yang benar-benar pernah disimpan owner.
	return len(settings.WebhookTokenCiphertext) > 0
}

func maskedSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "••••••"
	}
	return value[:3] + "••••••" + value[len(value)-4:]
}

// validatePaymentEndpointURL menerima URL https di luar localhost, atau http
// untuk localhost/127.0.0.1 (pengembangan).
func validatePaymentEndpointURL(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("URL tidak valid")
	}
	host := parsed.Hostname()
	localHTTP := parsed.Scheme == "http" && (host == "localhost" || host == "127.0.0.1")
	if parsed.Scheme != "https" && !localHTTP {
		return fmt.Errorf("URL harus memakai https di luar localhost")
	}
	return nil
}

func validatePaymentBaseURL(value, environment string) error {
	if err := validatePaymentEndpointURL(value); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("URL tidak valid")
	}
	if strings.EqualFold(strings.TrimSpace(environment), "production") {
		if parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "api.xendit.co") || parsed.Port() != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return fmt.Errorf("production hanya mengizinkan https://api.xendit.co")
		}
	}
	return nil
}
