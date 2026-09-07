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
	AppPort              string
	DatabaseURL          string
	RedisAddr            string
	RedisClusterAddrs    []string
	RedisPassword        string
	RedisDB              int
	JWTSecret            string
	JWTIssuer            string
	AccessTokenTTL       time.Duration
	AllowedOrigin        string
	PaymentWebhookSecret string
	PaymentCheckoutURL   string
	SentryDSN            string
	Environment          string
	Release              string
	SentryTracesRate     float64
	GoogleClientID       string
	PublicWebURL         string
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	SMTPFrom             string
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

	cfg := Config{
		AppPort:       strings.TrimPrefix(value("APP_PORT", "8080"), ":"),
		DatabaseURL:   databaseURL,
		RedisAddr:     value("REDIS_ADDR", "localhost:6379"),
		RedisPassword: redisPassword, RedisDB: redisDB,
		JWTSecret: jwtSecret, JWTIssuer: value("JWT_ISSUER", "tka-api"),
		AccessTokenTTL: ttl, AllowedOrigin: value("ALLOWED_ORIGIN", "http://localhost:3000"),
		PaymentWebhookSecret: paymentWebhookSecret,
		PaymentCheckoutURL:   value("PAYMENT_CHECKOUT_URL", "https://payment-gateway.example/checkout"),
		SentryDSN:            sentryDSN,
		Environment:          value("APP_ENVIRONMENT", "development"),
		Release:              os.Getenv("APP_RELEASE"),
		SentryTracesRate:     sentryTracesRate,
		GoogleClientID:       strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		PublicWebURL:         strings.TrimRight(value("PUBLIC_WEB_URL", value("ALLOWED_ORIGIN", "http://localhost:3000")), "/"),
		SMTPHost:             strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:             value("SMTP_PORT", "587"),
		SMTPUsername:         strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:         smtpPassword,
		SMTPFrom:             strings.TrimSpace(os.Getenv("SMTP_FROM")),
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
