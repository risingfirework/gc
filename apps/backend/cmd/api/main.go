package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"tka/apps/backend/internal/config"
	httpdelivery "tka/apps/backend/internal/delivery/http"
	"tka/apps/backend/internal/domain"
	"tka/apps/backend/internal/observability"
	postgresrepo "tka/apps/backend/internal/repository/postgres"
	redisrepo "tka/apps/backend/internal/repository/redis"
	"tka/apps/backend/internal/service"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	sentryEnabled := cfg.SentryDSN != ""
	if sentryEnabled {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.SentryDSN, Environment: cfg.Environment, Release: cfg.Release,
			EnableTracing: cfg.SentryTracesRate > 0, TracesSampleRate: cfg.SentryTracesRate,
			AttachStacktrace: true, SendDefaultPII: false,
		}); err != nil {
			logger.Error("initialize Sentry", "error", err)
			os.Exit(1)
		}
		defer sentry.Flush(2 * time.Second)
	}
	metrics := observability.NewMetrics()

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("create postgres pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(startupCtx); err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}

	redisAddrs := cfg.RedisClusterAddrs
	if len(redisAddrs) == 0 {
		redisAddrs = []string{cfg.RedisAddr}
	}
	redisClient := goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: redisAddrs, Password: cfg.RedisPassword, DB: cfg.RedisDB,
		DialTimeout: 5 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("close Redis client", "error", err)
		}
	}()
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		logger.Warn("Redis unavailable at startup; CBT answer writes will use PostgreSQL fallback", "error", err)
	}

	userRepository := postgresrepo.NewUserRepository(db)
	sessionRepository := redisrepo.NewSessionRepository(redisClient)
	var googleVerifier domain.GoogleIDTokenVerifier
	if cfg.GoogleClientID != "" {
		googleVerifier = service.NewGoogleVerifier(cfg.GoogleClientID)
	}
	authService := service.NewAuthService(userRepository, sessionRepository, cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL, googleVerifier)
	var passwordMailer domain.PasswordResetMailer
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" {
		passwordMailer = service.NewSMTPPasswordResetMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	}
	authService.ConfigurePasswordReset(cfg.PublicWebURL, cfg.Environment, passwordMailer)
	examRepository := postgresrepo.NewExamRepository(db)
	cbtRepository := redisrepo.NewCBTRepository(redisClient)
	cbtService := service.NewCBTService(examRepository, cbtRepository, logger, examRepository)
	paymentRepository := postgresrepo.NewPaymentRepository(db)
	paymentGateway := service.NewHMACPaymentGateway(cfg.PaymentWebhookSecret, cfg.PaymentCheckoutURL)
	paymentService := service.NewPaymentService(paymentRepository, paymentGateway)
	analyticsRepository := postgresrepo.NewAnalyticsRepository(db)
	examService := service.NewExamService(analyticsRepository)
	adminService := service.NewAdminService(postgresrepo.NewAdminRepository(db))
	teacherService := service.NewTeacherService(postgresrepo.NewTeacherRepository(db))
	testimonialService := service.NewTestimonialService(postgresrepo.NewTestimonialRepository(db))

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go runAutoSubmit(signalCtx, cbtService, logger)

	server := &http.Server{
		Addr: cfg.HTTPAddress(),
		Handler: httpdelivery.NewRouter(authService, cbtService, paymentService, examService, logger, cfg.AllowedOrigin,
			httpdelivery.RouterOptions{Metrics: metrics, EnableSentry: sentryEnabled, Admin: adminService, Teacher: teacherService, Testimonial: testimonialService}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          log.New(os.Stderr, "http: ", log.LstdFlags),
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("TKA API listening", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
		os.Exit(1)
	}
}

func runHealthcheck() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		os.Exit(1)
	}
	statusCode := response.StatusCode
	if err := response.Body.Close(); err != nil {
		os.Exit(1)
	}
	if statusCode != http.StatusOK {
		os.Exit(1)
	}
}

func runAutoSubmit(ctx context.Context, cbt domain.CBTService, logger *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := cbt.AutoSubmitTask(taskCtx)
			cancel()
			if err != nil {
				logger.Error("auto-submit task", "error", err)
			}
		}
	}
}
