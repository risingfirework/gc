package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"tka/apps/backend/internal/config"
	httpdelivery "tka/apps/backend/internal/delivery/http"
	"tka/apps/backend/internal/domain"
	"tka/apps/backend/internal/observability"
	postgresrepo "tka/apps/backend/internal/repository/postgres"
	redisrepo "tka/apps/backend/internal/repository/redis"
	"tka/apps/backend/internal/scheduler"
	"tka/apps/backend/internal/service"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "create-owner" {
		if err := runCreateOwner(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "create-owner:", err)
			os.Exit(1)
		}
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
	refreshRepository := postgresrepo.NewRefreshTokenRepository(db)
	authService.ConfigureRefreshTokens(refreshRepository, cfg.RefreshTokenTTL)
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
	adminService := service.NewAdminService(postgresrepo.NewAdminRepository(db), service.NewSIMPKBClient(cfg.SIMPKBCheckURL))
	teacherService := service.NewTeacherService(postgresrepo.NewTeacherRepository(db))
	affiliateService := service.NewAffiliateService(postgresrepo.NewAffiliateRepository(db), postgresrepo.NewTeacherRepository(db), cfg.PublicWebURL)
	testimonialService := service.NewTestimonialService(postgresrepo.NewTestimonialRepository(db))
	notificationService := domain.NewNotificationService(postgresrepo.NewNotificationRepository(db))

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go scheduler.Start(signalCtx, logger,
		scheduler.New("auto-submit", cfg.AutoSubmitInterval, 30*time.Second, func(ctx context.Context) error {
			return cbtService.AutoSubmitTask(ctx)
		}),
		scheduler.New("auto-payout", cfg.AutoPayoutInterval, 30*time.Second, func(ctx context.Context) error {
			dashboard, err := adminService.GetFinanceDashboard(ctx)
			if err != nil {
				return err
			}
			if !dashboard.Settings.AutoPayout || dashboard.Settings.PayoutCycle == "manual" {
				return nil
			}
			payments, err := adminService.AutoPayTeachers(ctx)
			if err == nil && len(payments) > 0 {
				logger.Info("auto-payout processed", "count", len(payments))
			}
			return err
		}),
		scheduler.New("simpkb-teacher-check", cfg.SIMPKBCheckInterval, 30*time.Minute, func(ctx context.Context) error {
			return adminService.SIMPKBWorkerTick(ctx)
		}),
		scheduler.New("expire-pending", cfg.PaymentExpiryInterval, 30*time.Second, func(ctx context.Context) error {
			count, err := paymentService.ExpirePendingTransactions(ctx)
			if err == nil && count > 0 {
				logger.Info("expired pending transactions", "count", count)
			}
			return err
		}),
	)
	// Cek SIMPKB langsung saat aplikasi baru mulai agar screenshot guru pending
	// sudah tersedia begitu panel verifikasi pertama kali dibuka.
	go func() {
		workerCtx, cancel := context.WithTimeout(signalCtx, 20*time.Minute)
		defer cancel()
		if err := adminService.SIMPKBWorkerTick(workerCtx); err != nil {
			logger.Warn("initial simpkb check stopped", "error", err)
		}
	}()

	server := &http.Server{
		Addr: cfg.HTTPAddress(),
		Handler: httpdelivery.NewRouter(authService, cbtService, paymentService, examService, logger, cfg.AllowedOrigin,
			httpdelivery.RouterOptions{
				Metrics: metrics, EnableSentry: sentryEnabled,
				RateLimiter:              redisrepo.NewRateLimiter(redisClient),
				LoginRateLimit:           cfg.LoginRateLimit,
				LoginRateWindow:          cfg.LoginRateWindow,
				ForgotPasswordRateLimit:  cfg.ForgotPasswordRateLimit,
				ForgotPasswordRateWindow: cfg.ForgotPasswordRateWindow,
				RegisterRateLimit:        cfg.RegisterRateLimit,
				RegisterRateWindow:       cfg.RegisterRateWindow,
				GoogleRateLimit:          cfg.GoogleRateLimit,
				GoogleRateWindow:         cfg.GoogleRateWindow,
				TrustProxy:               cfg.TrustProxy,
				ViewRateLimit:            cfg.PaymentViewRateLimit,
				ViewRateWindow:           cfg.PaymentViewRateWindow,
				Admin:                    adminService, Teacher: teacherService, Testimonial: testimonialService, Notification: notificationService, Affiliate: affiliateService,
			}),
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

// runCreateOwner membuat akun owner pertama kali untuk lingkungan produksi.
// Pemakaian: create-owner --email admin@example.com --password sekretkuat.
func runCreateOwner(args []string) error {
	email, password := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email":
			if i+1 < len(args) {
				i++
				email = args[i]
			}
		case "--password":
			if i+1 < len(args) {
				i++
				password = args[i]
			}
		}
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") || len(email) > 320 {
		return errors.New("--email wajib diisi dengan alamat email valid")
	}
	password = strings.TrimSpace(password)
	if len([]byte(password)) < 8 || len([]byte(password)) > 72 {
		return errors.New("--password wajib diisi dengan 8-72 karakter")
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, school_level, name)
		VALUES ($1, $2, 'owner', 'SMA', '')
		ON CONFLICT DO NOTHING
		RETURNING id`, email, string(hash)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("akun dengan email %s sudah terdaftar", email)
	}
	if err != nil {
		return fmt.Errorf("insert owner: %w", err)
	}
	fmt.Printf("Owner berhasil dibuat: %s (%s)\n", email, userID)
	return nil
}
