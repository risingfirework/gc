package service

import (
	"context"
	"testing"

	"tka/apps/backend/internal/domain"
	"tka/apps/backend/internal/security"
)

type fakeSettingsStore struct {
	stored *domain.EncryptedPaymentSettings
	err    error
}

func TestValidatePaymentBaseURLProduction(t *testing.T) {
	if err := validatePaymentBaseURL("https://api.xendit.co", "production"); err != nil {
		t.Fatalf("official Xendit URL rejected: %v", err)
	}
	if err := validatePaymentBaseURL("https://attacker.example", "production"); err == nil {
		t.Fatal("custom production payment URL must be rejected")
	}
}

func (s *fakeSettingsStore) GetPaymentSettings(_ context.Context) (*domain.EncryptedPaymentSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.stored == nil {
		return nil, domain.ErrPaymentSettingsNotFound
	}
	copyStored := *s.stored
	return &copyStored, nil
}

func (s *fakeSettingsStore) UpsertPaymentSettings(_ context.Context, settings *domain.EncryptedPaymentSettings) error {
	if s.err != nil {
		return s.err
	}
	copyStored := *settings
	s.stored = &copyStored
	return nil
}

func newSettingsTestSvc(store *fakeSettingsStore, env PaymentSettingsEnv) *PaymentService {
	cipher, err := security.NewPaymentCipher("0123456789012345678901234567890123456789")
	if err != nil {
		panic(err)
	}
	svc := &PaymentService{
		repository:    &fakePaymentRepo{},
		now:           nil,
		settingsStore: store,
		cipher:        cipher,
		envDefaults:   env,
		environment:   "development",
	}
	svc.gateway = newPaymentGatewayGate(&fakePaymentGateway{})
	return svc
}

func updateSettingsSvc(t *testing.T, input domain.PaymentSettingsInput) (*PaymentService, *fakeSettingsStore, *domain.PaymentSettings, error) {
	t.Helper()
	store := &fakeSettingsStore{}
	svc := newSettingsTestSvc(store, PaymentSettingsEnv{})
	settings, err := svc.UpdatePaymentSettings(context.Background(), input)
	return svc, store, settings, err
}

func paymentSettingString(value string) *string { return &value }

func TestUpdatePaymentSettingsPersistsEncryptedAndMasks(t *testing.T) {
	cipher, _ := security.NewPaymentCipher("0123456789012345678901234567890123456789")
	const secret = "xnd_development_sk_rahasia_123456789"
	svc, store, settings, err := updateSettingsSvc(t, domain.PaymentSettingsInput{
		Provider:           "xendit",
		SecretKey:          secret,
		WebhookToken:       "callbacktoken-panjang-32-karakter-xyz",
		SuccessRedirectURL: paymentSettingString("https://app.example.com/payment/success"),
		FailureRedirectURL: paymentSettingString("https://app.example.com/payment/failure"),
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if store.stored == nil {
		t.Fatalf("expected settings persisted")
	}
	plain, err := cipher.Decrypt(store.stored.SecretKeyCiphertext)
	if err != nil {
		t.Fatalf("decrypt stored secret: %v", err)
	}
	if string(plain) != secret {
		t.Fatalf("stored secret mismatch: got %q", plain)
	}
	if !settings.SecretKeyConfigured {
		t.Fatalf("expected secret key configured")
	}
	if settings.SecretKeyMasked == secret {
		t.Fatalf("masked secret must not equal plaintext")
	}
	if settings.Provider != "xendit" || settings.Environment != "development" {
		t.Fatalf("unexpected settings metadata: %+v", settings)
	}
	if _, ok := svc.gateway.Get().(*XenditGateway); !ok {
		t.Fatalf("expected live gateway swapped to XenditGateway, got %T", svc.gateway.Get())
	}
}

func TestUpdatePaymentSettingsIgnoresBlankCredentialsToKeepStored(t *testing.T) {
	store := &fakeSettingsStore{}
	svc := newSettingsTestSvc(store, PaymentSettingsEnv{})
	first := domain.PaymentSettingsInput{SecretKey: "xnd_development_sk_lama_12345678", WebhookToken: "callbacktoken-panjang-32-karakter-abc"}
	if _, err := svc.UpdatePaymentSettings(context.Background(), first); err != nil {
		t.Fatalf("first update: %v", err)
	}
	second := domain.PaymentSettingsInput{WebhookToken: "callbacktoken-panjang-32-karakter-def", SecretKey: ""}
	if _, err := svc.UpdatePaymentSettings(context.Background(), second); err != nil {
		t.Fatalf("second update: %v", err)
	}
	plain, err := svc.cipher.Decrypt(store.stored.SecretKeyCiphertext)
	if err != nil || string(plain) != "xnd_development_sk_lama_12345678" {
		t.Fatalf("stored secret should be unchanged, got %q err=%v", plain, err)
	}
}

func TestUpdatePaymentSettingsRequiresToken(t *testing.T) {
	_, _, _, err := updateSettingsSvc(t, domain.PaymentSettingsInput{SecretKey: "xnd_development_sk_1"})
	if err == nil {
		t.Fatalf("expected error when webhook token empty")
	}
}

func TestUpdatePaymentSettingsRejectsShortToken(t *testing.T) {
	_, _, _, err := updateSettingsSvc(t, domain.PaymentSettingsInput{SecretKey: "xnd_development_sk_1", WebhookToken: "pendek"})
	if err == nil {
		t.Fatalf("expected error for short webhook token")
	}
}

func TestUpdatePaymentSettingsRejectsBadRedirectURL(t *testing.T) {
	_, _, _, err := updateSettingsSvc(t, domain.PaymentSettingsInput{
		SecretKey:          "xnd_development_sk_1",
		WebhookToken:       "callbacktoken-panjang-32-karakter-abc",
		SuccessRedirectURL: paymentSettingString("http://app.example.com/success"),
	})
	if err == nil {
		t.Fatalf("expected error for non-https redirect url outside localhost")
	}
}

func TestUpdatePaymentSettingsAcceptsLocalhostHTTP(t *testing.T) {
	_, _, _, err := updateSettingsSvc(t, domain.PaymentSettingsInput{
		SecretKey:          "xnd_development_sk_1",
		WebhookToken:       "callbacktoken-panjang-32-karakter-abc",
		BaseURL:            paymentSettingString("http://localhost:8080/v2"),
		SuccessRedirectURL: paymentSettingString("http://localhost:5173/payment/success"),
	})
	if err != nil {
		t.Fatalf("expected localhost http to be allowed, got %v", err)
	}
}

func TestUpdatePaymentSettingsRejectsUnknownProvider(t *testing.T) {
	_, _, _, err := updateSettingsSvc(t, domain.PaymentSettingsInput{
		Provider:     "midtrans",
		SecretKey:    "xnd_development_sk_1",
		WebhookToken: "callbacktoken-panjang-32-karakter-abc",
	})
	if err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}

func TestGetPaymentSettingsDefaultsWhenUnset(t *testing.T) {
	svc := newSettingsTestSvc(&fakeSettingsStore{}, PaymentSettingsEnv{})
	settings, err := svc.GetPaymentSettings(context.Background())
	if err != nil {
		t.Fatalf("get default settings: %v", err)
	}
	if settings.SecretKeyConfigured || settings.WebhookTokenConfigured {
		t.Fatalf("expected nothing configured by default")
	}
	if settings.BaseURL != "https://api.xendit.co" {
		t.Fatalf("unexpected default base url %q", settings.BaseURL)
	}
}

func TestGetPaymentSettingsMergesEnvFallbackBeforeStored(t *testing.T) {
	store := &fakeSettingsStore{}
	svc := newSettingsTestSvc(store, PaymentSettingsEnv{
		SecretKey: "xnd_development_sk_env_12345678",
		BaseURL:   "https://api.example.dev/v2",
	})
	settings, err := svc.GetPaymentSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if !settings.SecretKeyConfigured {
		t.Fatalf("expected env secret reported as configured")
	}
	if settings.BaseURL != "https://api.example.dev/v2" {
		t.Fatalf("expected env base url, got %q", settings.BaseURL)
	}
	if store.stored != nil {
		t.Fatalf("env fallback should not persist anything to storage")
	}
}

func TestLoadPaymentSettingsActivatesStoredGateway(t *testing.T) {
	store := &fakeSettingsStore{}
	svc := newSettingsTestSvc(store, PaymentSettingsEnv{
		SecretKey:    "xnd_development_sk_env_12345678",
		WebhookToken: "callbacktoken-env-panjang-32-karakter",
		BaseURL:      "https://api.env.example/v2",
	})
	secretCiphertext, err := svc.cipher.Encrypt([]byte("xnd_development_sk_database_12345678"))
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	tokenCiphertext, err := svc.cipher.Encrypt([]byte("callbacktoken-database-panjang-32-karakter"))
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}
	store.stored = &domain.EncryptedPaymentSettings{
		SecretKeyCiphertext:    secretCiphertext,
		WebhookTokenCiphertext: tokenCiphertext,
		BaseURL:                "https://api.database.example/v2",
		SuccessRedirectURL:     "https://app.example/success",
	}

	if err := svc.LoadPaymentSettings(context.Background()); err != nil {
		t.Fatalf("load payment settings: %v", err)
	}
	gateway, ok := svc.gateway.Get().(*XenditGateway)
	if !ok {
		t.Fatalf("expected XenditGateway, got %T", svc.gateway.Get())
	}
	if gateway.secretKey != "xnd_development_sk_database_12345678" || gateway.webhookToken != "callbacktoken-database-panjang-32-karakter" {
		t.Fatalf("stored credentials were not activated")
	}
	if gateway.baseURL != "https://api.database.example/v2" || gateway.successURL != "https://app.example/success" {
		t.Fatalf("stored endpoints were not activated: %+v", gateway)
	}
}

func TestUpdatePaymentSettingsCanClearRedirectURLs(t *testing.T) {
	store := &fakeSettingsStore{}
	svc := newSettingsTestSvc(store, PaymentSettingsEnv{
		SuccessRedirectURL: "https://env.example/success",
		FailureRedirectURL: "https://env.example/failure",
	})
	first := domain.PaymentSettingsInput{
		SecretKey:          "xnd_development_sk_lama_12345678",
		WebhookToken:       "callbacktoken-panjang-32-karakter-abc",
		SuccessRedirectURL: paymentSettingString("https://app.example/success"),
		FailureRedirectURL: paymentSettingString("https://app.example/failure"),
	}
	if _, err := svc.UpdatePaymentSettings(context.Background(), first); err != nil {
		t.Fatalf("first update: %v", err)
	}
	cleared := ""
	settings, err := svc.UpdatePaymentSettings(context.Background(), domain.PaymentSettingsInput{
		SuccessRedirectURL: &cleared,
		FailureRedirectURL: &cleared,
	})
	if err != nil {
		t.Fatalf("clear redirects: %v", err)
	}
	if settings.SuccessRedirectURL != "" || settings.FailureRedirectURL != "" {
		t.Fatalf("redirects should be empty: %+v", settings)
	}
	if store.stored.SuccessRedirectURL != "" || store.stored.FailureRedirectURL != "" {
		t.Fatalf("cleared redirects were not persisted: %+v", store.stored)
	}
	if err := svc.LoadPaymentSettings(context.Background()); err != nil {
		t.Fatalf("reload cleared redirects: %v", err)
	}
	gateway := svc.gateway.Get().(*XenditGateway)
	if gateway.successURL != "" || gateway.failureURL != "" {
		t.Fatalf("environment fallback restored cleared redirects after reload: %+v", gateway)
	}
	reloaded, err := svc.GetPaymentSettings(context.Background())
	if err != nil {
		t.Fatalf("get cleared redirects: %v", err)
	}
	if reloaded.SuccessRedirectURL != "" || reloaded.FailureRedirectURL != "" {
		t.Fatalf("environment fallback restored cleared redirects in response: %+v", reloaded)
	}
}
