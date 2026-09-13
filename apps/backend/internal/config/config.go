// Package config memuat konfigurasi runtime dari environment variable
// dan file Docker secrets (pola *_FILE) beserta nilai default bawaan.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	DatabaseURL              string
	RedisAddr                string
	RedisClusterAddrs        []string
	RedisPassword            string
	RedisDB                  int
	JWTSecret                string
	JWTIssuer                string
	AccessTokenTTL           time.Duration
	AllowedOrigin            string
	PaymentWebhookSecret     string
	PaymentCheckoutURL       string
	SentryDSN                string
	Environment              string
	Release                  string
	SentryTracesRate         float64
	GoogleClientID           string
	PublicWebURL             string
	SMTPHost                 string
	SMTPPort                 string
	SMTPUsername             string
	SMTPPassword             string
	SMTPFrom                 string
	LoginRateLimit           int
	LoginRateWindow          time.Duration
	ForgotPasswordRateLimit  int
	ForgotPasswordRateWindow time.Duration
	RegisterRateLimit        int
	RegisterRateWindow       time.Duration
	GoogleRateLimit          int
	GoogleRateWindow         time.Duration
	RefreshTokenTTL          time.Duration
	TrustProxy               bool
	AutoSubmitInterval       time.Duration
	AutoPayoutInterval       time.Duration
	PaymentExpiryInterval    time.Duration
	PaymentViewRateLimit     int
	PaymentViewRateWindow    time.Duration
	SIMPKBCheckURL           string
	SIMPKBCheckInterval      time.Duration
}

func Load() (Config, error) {
	// Environment variables remain authoritative; .env is only a local convenience.
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	redisDB, err := strconv.Atoi(value("REDIS_DB", "0"))
	if err != nil || redisDB < 0 {
		return Config{}, fmt.Errorf("REDIS_DB must be a non-negative integer")
	}
	ttl, err := time.ParseDuration(value("JWT_ACCESS_TTL", "15m"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("JWT_ACCESS_TTL must be a positive duration")
	}

	databaseURL, err := secretValue("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	redisPassword, err := secretValue("REDIS_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	jwtSecret, err := secretValue("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	paymentWebhookSecret, err := secretValue("PAYMENT_WEBHOOK_SECRET")
	if err != nil {
		return Config{}, err
	}
	sentryDSN, err := secretValue("SENTRY_DSN")
	if err != nil {
		return Config{}, err
	}
	smtpPassword, err := secretValue("SMTP_PASSWORD")
	if err != nil {
		return Config{}, err
	}
	sentryTracesRate, err := strconv.ParseFloat(value("SENTRY_TRACES_SAMPLE_RATE", "0.05"), 64)
	if err != nil || sentryTracesRate < 0 || sentryTracesRate > 1 {
		return Config{}, fmt.Errorf("SENTRY_TRACES_SAMPLE_RATE must be between 0 and 1")
	}
	loginRateLimit, err := strconv.Atoi(value("LOGIN_RATE_LIMIT", "10"))
	if err != nil || loginRateLimit <= 0 {
		return Config{}, fmt.Errorf("LOGIN_RATE_LIMIT must be a positive integer")
	}
	loginRateWindow, err := time.ParseDuration(value("LOGIN_RATE_WINDOW", "15m"))
	if err != nil || loginRateWindow <= 0 {
		return Config{}, fmt.Errorf("LOGIN_RATE_WINDOW must be a positive duration")
	}
	forgotRateLimit, err := strconv.Atoi(value("FORGOT_RATE_LIMIT", "5"))
	if err != nil || forgotRateLimit <= 0 {
		return Config{}, fmt.Errorf("FORGOT_RATE_LIMIT must be a positive integer")
	}
	forgotRateWindow, err := time.ParseDuration(value("FORGOT_RATE_WINDOW", "1h"))
	if err != nil || forgotRateWindow <= 0 {
		return Config{}, fmt.Errorf("FORGOT_RATE_WINDOW must be a positive duration")
	}
	registerRateLimit, err := strconv.Atoi(value("REGISTER_RATE_LIMIT", "10"))
	if err != nil || registerRateLimit <= 0 {
		return Config{}, fmt.Errorf("REGISTER_RATE_LIMIT must be a positive integer")
	}
	registerRateWindow, err := time.ParseDuration(value("REGISTER_RATE_WINDOW", "15m"))
	if err != nil || registerRateWindow <= 0 {
		return Config{}, fmt.Errorf("REGISTER_RATE_WINDOW must be a positive duration")
	}
	googleRateLimit, err := strconv.Atoi(value("GOOGLE_RATE_LIMIT", "10"))
	if err != nil || googleRateLimit <= 0 {
		return Config{}, fmt.Errorf("GOOGLE_RATE_LIMIT must be a positive integer")
	}
	googleRateWindow, err := time.ParseDuration(value("GOOGLE_RATE_WINDOW", "15m"))
	if err != nil || googleRateWindow <= 0 {
		return Config{}, fmt.Errorf("GOOGLE_RATE_WINDOW must be a positive duration")
	}
	refreshTokenTTL, err := time.ParseDuration(value("JWT_REFRESH_TTL", "720h"))
	if err != nil || refreshTokenTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_REFRESH_TTL must be a positive duration")
	}
	trustProxy, err := strconv.ParseBool(value("TRUST_PROXY", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("TRUST_PROXY must be a boolean")
	}
	autoSubmitInterval, err := time.ParseDuration(value("AUTO_SUBMIT_INTERVAL", "5s"))
	if err != nil || autoSubmitInterval <= 0 {
		return Config{}, fmt.Errorf("AUTO_SUBMIT_INTERVAL must be a positive duration")
	}
	autoPayoutInterval, err := time.ParseDuration(value("AUTO_PAYOUT_INTERVAL", "6h"))
	if err != nil || autoPayoutInterval <= 0 {
		return Config{}, fmt.Errorf("AUTO_PAYOUT_INTERVAL must be a positive duration")
	}
	paymentExpiryInterval, err := time.ParseDuration(value("PAYMENT_EXPIRY_INTERVAL", "1m"))
	if err != nil || paymentExpiryInterval <= 0 {
		return Config{}, fmt.Errorf("PAYMENT_EXPIRY_INTERVAL must be a positive duration")
	}
	paymentViewRateLimit, err := strconv.Atoi(value("PAYMENT_VIEW_RATE_LIMIT", "60"))
	if err != nil || paymentViewRateLimit <= 0 {
		return Config{}, fmt.Errorf("PAYMENT_VIEW_RATE_LIMIT must be a positive integer")
	}
	paymentViewRateWindow, err := time.ParseDuration(value("PAYMENT_VIEW_RATE_WINDOW", "1m"))
	if err != nil || paymentViewRateWindow <= 0 {
		return Config{}, fmt.Errorf("PAYMENT_VIEW_RATE_WINDOW must be a positive duration")
	}
	simpkbCheckInterval, err := time.ParseDuration(value("SIMPKB_CHECK_INTERVAL", "5m"))
	if err != nil || simpkbCheckInterval <= 0 {
		return Config{}, fmt.Errorf("SIMPKB_CHECK_INTERVAL must be a positive duration")
	}

	cfg := Config{
		AppPort:       strings.TrimPrefix(value("APP_PORT", "8080"), ":"),
		DatabaseURL:   databaseURL,
		RedisAddr:     value("REDIS_ADDR", "localhost:6379"),
		RedisPassword: redisPassword, RedisDB: redisDB,
		JWTSecret: jwtSecret, JWTIssuer: value("JWT_ISSUER", "tka-api"),
		AccessTokenTTL: ttl, AllowedOrigin: value("ALLOWED_ORIGIN", "http://localhost:3000"),
		PaymentWebhookSecret:     paymentWebhookSecret,
		PaymentCheckoutURL:       value("PAYMENT_CHECKOUT_URL", "https://payment-gateway.example/checkout"),
		SentryDSN:                sentryDSN,
		Environment:              value("APP_ENVIRONMENT", "development"),
		Release:                  os.Getenv("APP_RELEASE"),
		SentryTracesRate:         sentryTracesRate,
		GoogleClientID:           strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		PublicWebURL:             strings.TrimRight(value("PUBLIC_WEB_URL", value("ALLOWED_ORIGIN", "http://localhost:3000")), "/"),
		SMTPHost:                 strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                 value("SMTP_PORT", "587"),
		SMTPUsername:             strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:             smtpPassword,
		SMTPFrom:                 strings.TrimSpace(os.Getenv("SMTP_FROM")),
		LoginRateLimit:           loginRateLimit,
		LoginRateWindow:          loginRateWindow,
		ForgotPasswordRateLimit:  forgotRateLimit,
		ForgotPasswordRateWindow: forgotRateWindow,
		RegisterRateLimit:        registerRateLimit,
		RegisterRateWindow:       registerRateWindow,
		GoogleRateLimit:          googleRateLimit,
		GoogleRateWindow:         googleRateWindow,
		RefreshTokenTTL:          refreshTokenTTL,
		TrustProxy:               trustProxy,
		AutoSubmitInterval:       autoSubmitInterval,
		AutoPayoutInterval:       autoPayoutInterval,
		PaymentExpiryInterval:    paymentExpiryInterval,
		PaymentViewRateLimit:     paymentViewRateLimit,
		PaymentViewRateWindow:    paymentViewRateWindow,
		SIMPKBCheckURL:           strings.TrimSpace(value("SIMPKB_CHECK_URL", "http://simpkb-check:8080")),
		SIMPKBCheckInterval:      simpkbCheckInterval,
	}
	if clusterAddrs := strings.TrimSpace(os.Getenv("REDIS_CLUSTER_ADDRS")); clusterAddrs != "" {
		for _, address := range strings.Split(clusterAddrs, ",") {
			if address = strings.TrimSpace(address); address != "" {
				cfg.RedisClusterAddrs = append(cfg.RedisClusterAddrs, address)
			}
		}
		if len(cfg.RedisClusterAddrs) == 0 {
			return Config{}, fmt.Errorf("REDIS_CLUSTER_ADDRS must contain at least one address")
		}
		if cfg.RedisDB != 0 {
			return Config{}, fmt.Errorf("REDIS_DB must be 0 when REDIS_CLUSTER_ADDRS is set")
		}
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if cfg.AppPort == "" {
		return Config{}, fmt.Errorf("APP_PORT cannot be empty")
	}
	if len(cfg.PaymentWebhookSecret) < 32 {
		return Config{}, fmt.Errorf("PAYMENT_WEBHOOK_SECRET must contain at least 32 characters")
	}
	return cfg, nil
}

func (c Config) HTTPAddress() string { return ":" + c.AppPort }

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// secretValue supports Docker secrets without weakening local .env ergonomics.
// KEY_FILE takes precedence over KEY when both are defined.
func secretValue(key string) (string, error) {
	if path := strings.TrimSpace(os.Getenv(key + "_FILE")); path != "" {
		contents, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", key, err)
		}
		return strings.TrimSpace(string(contents)), nil
	}
	return strings.TrimSpace(os.Getenv(key)), nil
}
