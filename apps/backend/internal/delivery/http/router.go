package http

import (
	"log/slog"
	"net/http"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"tka/apps/backend/internal/delivery/http/handler"
	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
	"tka/apps/backend/internal/observability"
)

type RouterOptions struct {
	Metrics      *observability.Metrics
	EnableSentry bool
	Admin        domain.AdminService
	Teacher      domain.TeacherService
	Testimonial  domain.TestimonialService
}

func NewRouter(auth domain.AuthService, cbt domain.CBTService, payment domain.PaymentService, analytics domain.ExamAnalyticsService, logger *slog.Logger, allowedOrigin string, options ...RouterOptions) http.Handler {
	var opts RouterOptions
	if len(options) > 0 {
		opts = options[0]
	}
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	if opts.Metrics != nil {
		router.Use(opts.Metrics.Middleware)
	}
	router.Use(chimiddleware.Recoverer)
	if opts.EnableSentry {
		sentryMiddleware := sentryhttp.New(sentryhttp.Options{Repanic: true, WaitForDelivery: false})
		router.Use(sentryMiddleware.Handle)
	}
	router.Use(chimiddleware.Timeout(15 * time.Second))
	router.Use(securityHeaders)
	router.Use(cors(allowedOrigin))
	if opts.Metrics != nil {
		router.Handle("/metrics", opts.Metrics.Handler())
	}

	authHandler := handler.NewAuthHandler(auth, logger)
	cbtHandler := handler.NewCBTHandler(cbt, logger)
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	router.Route("/api/v1/auth", func(router chi.Router) {
		router.Post("/register", authHandler.Register)
		router.Post("/login", authHandler.Login)
		router.Post("/forgot-password", authHandler.ForgotPassword)
		router.Post("/reset-password", authHandler.ResetPassword)
		router.Post("/google", authHandler.GoogleLogin)
		router.With(middleware.RequireAuth(auth)).Get("/me", authHandler.CurrentUser)
		router.With(middleware.RequireAuth(auth)).Patch("/me", authHandler.UpdateProfile)
		router.With(middleware.RequireAuth(auth)).Post("/logout", authHandler.Logout)
	})
	if payment != nil {
		paymentHandler := handler.NewPaymentHandler(payment, logger)
		router.Get("/api/v1/packages", paymentHandler.ListPackages)
		router.Post("/api/v1/packages/{id}/view", paymentHandler.TrackPackageView)
		router.Post("/api/v1/webhooks/payment", paymentHandler.Webhook)
		router.With(middleware.RequireAuth(auth)).Get("/api/v1/packages/mine", paymentHandler.ListMyPackages)
		router.With(middleware.RequireAuth(auth)).Get("/api/v1/finance/me", paymentHandler.PricingPolicy)
		router.With(middleware.RequireAuth(auth)).Get("/api/v1/packages/{id}/exams", cbtHandler.ListPackageExams)
		router.With(middleware.RequireAuth(auth)).Post("/api/v1/packages/{id}/claim", paymentHandler.ClaimFreePackage)
		router.With(middleware.RequireAuth(auth)).Post("/api/v1/transactions/checkout", paymentHandler.Checkout)
	}
	if opts.Admin != nil {
		adminHandler := handler.NewAdminHandler(opts.Admin, logger)
		router.Get("/api/v1/settings", adminHandler.SiteSettings)
		router.Route("/api/v1/admin", func(router chi.Router) {
			router.Use(middleware.RequireAuth(auth))
			router.Use(middleware.RequireAdmin)
			router.Get("/dashboard", adminHandler.Dashboard)
			router.Put("/settings", adminHandler.UpdateSiteSettings)
			router.Get("/finance", adminHandler.FinanceDashboard)
			router.Put("/finance/settings", adminHandler.UpdateFinanceSettings)
			router.Put("/finance/users/{id}", adminHandler.UpdateUserFinance)
			router.Post("/finance/teachers/{id}/payout", adminHandler.PayTeacherCommissions)
			router.Patch("/finance/payout-requests/{id}", adminHandler.ReviewPayoutRequest)
			router.Post("/users", adminHandler.CreateUser)
			router.Patch("/users/{id}", adminHandler.UpdateUser)
			router.Delete("/users/{id}", adminHandler.DeleteUser)
			router.Post("/packages", adminHandler.CreatePackage)
			router.Post("/packages/bundle", adminHandler.CreatePackageBundle)
			router.Put("/packages/{id}", adminHandler.UpdatePackage)
			router.Delete("/packages/{id}", adminHandler.DeletePackage)
			router.Post("/exams", adminHandler.CreateExam)
			router.Put("/exams/{id}", adminHandler.UpdateExam)
			router.Delete("/exams/{id}", adminHandler.DeleteExam)
			router.Post("/questions", adminHandler.CreateQuestion)
			router.Put("/questions/{id}", adminHandler.UpdateQuestion)
			router.Delete("/questions/{id}", adminHandler.DeleteQuestion)
			router.Get("/master/{category}", adminHandler.ListMaster)
			router.Post("/master/{category}", adminHandler.CreateMaster)
			router.Put("/master/{category}/{id}", adminHandler.UpdateMaster)
			router.Delete("/master/{category}/{id}", adminHandler.DeleteMaster)
		})
	}
	if opts.Teacher != nil {
		teacherHandler := handler.NewTeacherHandler(opts.Teacher, logger)
		router.Route("/api/v1/teacher", func(router chi.Router) {
			router.Use(middleware.RequireAuth(auth))
			router.Use(middleware.RequireTeacher)
			router.Get("/dashboard", teacherHandler.Dashboard)
			router.Put("/payout-account", teacherHandler.UpdatePayoutAccount)
			router.Post("/payout-requests", teacherHandler.CreatePayoutRequest)
			router.Patch("/payout-requests/{id}/cancel", teacherHandler.CancelPayoutRequest)
			router.Post("/packages", teacherHandler.CreatePackage)
			router.Post("/packages/bundle", teacherHandler.CreatePackageBundle)
			router.Put("/packages/{id}", teacherHandler.UpdatePackage)
			router.Delete("/packages/{id}", teacherHandler.DeletePackage)
			router.Post("/exams", teacherHandler.CreateExam)
			router.Put("/exams/{id}", teacherHandler.UpdateExam)
			router.Delete("/exams/{id}", teacherHandler.DeleteExam)
			router.Post("/questions", teacherHandler.CreateQuestion)
			router.Put("/questions/{id}", teacherHandler.UpdateQuestion)
			router.Delete("/questions/{id}", teacherHandler.DeleteQuestion)
		})
	}
	router.Group(func(router chi.Router) {
		router.Use(middleware.RequireAuth(auth))
		router.Get("/api/v1/exams/{id}/start", cbtHandler.StartExam)
		router.Post("/api/v1/cbt/answers/sync", cbtHandler.SyncAnswer)
		router.Post("/api/v1/exams/{id}/submit", cbtHandler.SubmitExam)
		if analytics != nil {
			analyticsHandler := handler.NewAnalyticsHandler(analytics, logger)
			router.Get("/api/v1/exams/{id}/result", analyticsHandler.Result)
			router.Get("/api/v1/rankings/global", analyticsHandler.GlobalRanking)
		}
	})
	if opts.Testimonial != nil {
		testimonialHandler := handler.NewTestimonialHandler(opts.Testimonial, logger)
		router.Get("/api/v1/testimonials", testimonialHandler.ListApproved)
		router.With(middleware.RequireAuth(auth)).Post("/api/v1/testimonials", testimonialHandler.Submit)
		router.With(middleware.RequireAuth(auth)).Get("/api/v1/testimonials/mine", testimonialHandler.GetMine)
		router.Route("/api/v1/admin/testimonials", func(router chi.Router) {
			router.Use(middleware.RequireAuth(auth))
			router.Use(middleware.RequireAdmin)
			router.Get("/", testimonialHandler.AdminList)
			router.Patch("/{id}", testimonialHandler.AdminUpdateStatus)
			router.Delete("/{id}", testimonialHandler.AdminDelete)
		})
	}
	return router
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func cors(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
