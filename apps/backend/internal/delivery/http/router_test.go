package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tka/apps/backend/internal/domain"
)

type authStub struct {
	register func(context.Context, domain.RegisterRequest) (*domain.RegisterResponse, error)
	login    func(context.Context, domain.LoginRequest) (*domain.LoginResponse, error)
	logout   func(context.Context, domain.AuthClaims) error
	current  func(context.Context, domain.AuthClaims) (*domain.UserResponse, error)
	update   func(context.Context, domain.AuthClaims, domain.UpdateProfileRequest) (*domain.UserResponse, error)
	validate func(context.Context, string) (*domain.AuthClaims, error)
	forgot   func(context.Context, domain.ForgotPasswordRequest) (*domain.ForgotPasswordResponse, error)
	reset    func(context.Context, domain.ResetPasswordRequest) error
}

type cbtStub struct {
	start func(context.Context, string, string) (*domain.ExamStartResponse, error)
}

func (s cbtStub) StartExam(ctx context.Context, userID, examID string) (*domain.ExamStartResponse, error) {
	return s.start(ctx, userID, examID)
}
func (s cbtStub) ListPackageExams(context.Context, string, string) ([]domain.ExamSummary, error) {
	return nil, nil
}
func (s cbtStub) SyncAnswer(context.Context, string, domain.SyncAnswerRequest) (*domain.SyncAnswerResponse, error) {
	return nil, nil
}
func (s cbtStub) SubmitExam(context.Context, string, string) (*domain.SubmitExamResponse, error) {
	return nil, nil
}
func (s cbtStub) AutoSubmitTask(context.Context) error { return nil }

func (s authStub) Register(ctx context.Context, input domain.RegisterRequest) (*domain.RegisterResponse, error) {
	return s.register(ctx, input)
}
func (s authStub) Login(ctx context.Context, input domain.LoginRequest) (*domain.LoginResponse, error) {
	return s.login(ctx, input)
}
func (s authStub) LoginWithGoogle(context.Context, domain.GoogleAuthRequest) (*domain.LoginResponse, error) {
	return nil, domain.ErrGoogleAuthUnavailable
}
func (s authStub) Logout(ctx context.Context, claims domain.AuthClaims) error {
	return s.logout(ctx, claims)
}
func (s authStub) CurrentUser(ctx context.Context, claims domain.AuthClaims) (*domain.UserResponse, error) {
	if s.current == nil {
		return &domain.UserResponse{ID: claims.UserID}, nil
	}
	return s.current(ctx, claims)
}
func (s authStub) UpdateProfile(ctx context.Context, claims domain.AuthClaims, input domain.UpdateProfileRequest) (*domain.UserResponse, error) {
	if s.update == nil {
		return &domain.UserResponse{ID: claims.UserID}, nil
	}
	return s.update(ctx, claims, input)
}
func (s authStub) ValidateAccessToken(ctx context.Context, token string) (*domain.AuthClaims, error) {
	return s.validate(ctx, token)
}
func (s authStub) ForgotPassword(ctx context.Context, input domain.ForgotPasswordRequest) (*domain.ForgotPasswordResponse, error) {
	if s.forgot == nil {
		return &domain.ForgotPasswordResponse{Message: "ok"}, nil
	}
	return s.forgot(ctx, input)
}
func (s authStub) ResetPassword(ctx context.Context, input domain.ResetPasswordRequest) error {
	if s.reset == nil {
		return nil
	}
	return s.reset(ctx, input)
}

func TestAuthRoutes(t *testing.T) {
	loggedOut := false
	stub := authStub{
		register: func(_ context.Context, input domain.RegisterRequest) (*domain.RegisterResponse, error) {
			return &domain.RegisterResponse{User: domain.UserResponse{ID: "user-1", Email: input.Email, Role: domain.RoleStudent, SchoolLevel: input.SchoolLevel}}, nil
		},
		login: func(_ context.Context, _ domain.LoginRequest) (*domain.LoginResponse, error) {
			return nil, domain.ErrInvalidCredentials
		},
		logout: func(_ context.Context, _ domain.AuthClaims) error { loggedOut = true; return nil },
		validate: func(_ context.Context, token string) (*domain.AuthClaims, error) {
			if token != "valid-token" {
				return nil, domain.ErrSessionInvalid
			}
			return &domain.AuthClaims{UserID: "user-1", JTI: "jti-1"}, nil
		},
	}
	router := NewRouter(stub, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:3000")

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"student@example.com","password":"strong-password","school_level":"SMA"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"student@example.com","password":"wrong-password"}`))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !loggedOut {
		t.Fatalf("logout status = %d, called = %v", response.Code, loggedOut)
	}
}

func TestLogoutRejectsReplacedSession(t *testing.T) {
	stub := authStub{
		register: func(context.Context, domain.RegisterRequest) (*domain.RegisterResponse, error) { return nil, nil },
		login:    func(context.Context, domain.LoginRequest) (*domain.LoginResponse, error) { return nil, nil },
		logout:   func(context.Context, domain.AuthClaims) error { t.Fatal("logout must not be called"); return nil },
		validate: func(context.Context, string) (*domain.AuthClaims, error) { return nil, domain.ErrSessionInvalid },
	}
	router := NewRouter(stub, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer replaced-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestCBTRouteRequiresJWTAndPassesAuthenticatedUser(t *testing.T) {
	const userID = "11111111-1111-4111-8111-111111111111"
	const examID = "22222222-2222-4222-8222-222222222222"
	auth := authStub{
		register: func(context.Context, domain.RegisterRequest) (*domain.RegisterResponse, error) { return nil, nil },
		login:    func(context.Context, domain.LoginRequest) (*domain.LoginResponse, error) { return nil, nil },
		logout:   func(context.Context, domain.AuthClaims) error { return nil },
		validate: func(_ context.Context, token string) (*domain.AuthClaims, error) {
			if token != "valid-token" {
				return nil, domain.ErrSessionInvalid
			}
			return &domain.AuthClaims{UserID: userID, JTI: "jti"}, nil
		},
	}
	cbt := cbtStub{start: func(_ context.Context, gotUserID, gotExamID string) (*domain.ExamStartResponse, error) {
		if gotUserID != userID || gotExamID != examID {
			t.Fatalf("user=%s exam=%s", gotUserID, gotExamID)
		}
		return &domain.ExamStartResponse{ExamID: examID, UserExamID: "attempt"}, nil
	}}
	router := NewRouter(auth, cbt, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "http://localhost:3000")

	request := httptest.NewRequest(http.MethodGet, "/api/v1/exams/"+examID+"/start", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/exams/"+examID+"/start", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, body=%s", response.Code, response.Body.String())
	}
}
