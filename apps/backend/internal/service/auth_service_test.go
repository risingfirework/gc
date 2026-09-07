package service

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

type fakeUsers struct {
	mu          sync.Mutex
	byEmail     map[string]domain.User
	resetHash   []byte
	resetUser   string
	resetExpiry time.Time
}

func (f *fakeUsers) CreatePasswordResetToken(_ context.Context, email string, tokenHash []byte, expiresAt time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	user, exists := f.byEmail[email]
	if !exists {
		return false, nil
	}
	f.resetHash = append([]byte(nil), tokenHash...)
	f.resetUser = user.ID
	f.resetExpiry = expiresAt
	return true, nil
}

func (f *fakeUsers) ResetPassword(_ context.Context, tokenHash []byte, passwordHash string, now time.Time) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !bytes.Equal(tokenHash, f.resetHash) || !now.Before(f.resetExpiry) {
		return "", domain.ErrInvalidResetToken
	}
	for email, user := range f.byEmail {
		if user.ID == f.resetUser {
			user.PasswordHash = passwordHash
			f.byEmail[email] = user
			f.resetHash = nil
			return user.ID, nil
		}
	}
	return "", domain.ErrInvalidResetToken
}

func (f *fakeUsers) Create(_ context.Context, user *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.byEmail[user.Email]; exists {
		return domain.ErrEmailAlreadyExists
	}
	user.ID = "user-1"
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	f.byEmail[user.Email] = *user
	return nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	user, exists := f.byEmail[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return &user, nil
}

func (f *fakeUsers) FindByID(_ context.Context, id string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, user := range f.byEmail {
		if user.ID == id {
			copy := user
			return &copy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUsers) UpdateProfile(_ context.Context, userID string, update domain.ProfileUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, user := range f.byEmail {
		if user.ID == userID {
			user.Name = update.Name
			user.Phone = update.Phone
			user.SchoolLevel = update.SchoolLevel
			if update.BirthDate != nil {
				user.BirthDate = *update.BirthDate
			}
			if update.PasswordHash != nil {
				user.PasswordHash = *update.PasswordHash
			}
			f.byEmail[email] = user
			return nil
		}
	}
	return domain.ErrUserNotFound
}

type fakeSessions struct {
	mu     sync.Mutex
	byUser map[string]string
}

func (f *fakeSessions) SetSession(_ context.Context, userID, jti string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byUser[userID] = jti
	return nil
}
func (f *fakeSessions) GetSession(_ context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	jti, ok := f.byUser[userID]
	if !ok {
		return "", domain.ErrSessionNotFound
	}
	return jti, nil
}
func (f *fakeSessions) DeleteSession(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byUser, userID)
	return nil
}

func newTestAuthService() *AuthService {
	return NewAuthService(
		&fakeUsers{byEmail: make(map[string]domain.User)},
		&fakeSessions{byUser: make(map[string]string)},
		"a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil,
	)
}

func TestRegisterAndLogin(t *testing.T) {
	service := newTestAuthService()
	registered, err := service.Register(context.Background(), domain.RegisterRequest{Email: " STUDENT@example.com ", Password: "strong-password", SchoolLevel: "sma"})
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.Email != "student@example.com" || registered.User.SchoolLevel != "SMA" {
		t.Fatalf("unexpected user: %+v", registered.User)
	}

	loggedIn, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.AccessToken == "" || loggedIn.TokenType != "Bearer" {
		t.Fatalf("unexpected login response: %+v", loggedIn)
	}
	if _, err := service.ValidateAccessToken(context.Background(), loggedIn.AccessToken); err != nil {
		t.Fatalf("validate token: %v", err)
	}
}

func TestForgotAndResetPasswordRevokesSession(t *testing.T) {
	users := &fakeUsers{byEmail: make(map[string]domain.User)}
	sessions := &fakeSessions{byUser: make(map[string]string)}
	service := NewAuthService(users, sessions, "a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil)
	service.ConfigurePasswordReset("http://localhost:3000", "development", nil)
	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "student@example.com", Password: "old-password", SchoolLevel: "SMA"}); err != nil {
		t.Fatal(err)
	}
	login, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "old-password"})
	if err != nil {
		t.Fatal(err)
	}
	forgot, err := service.ForgotPassword(context.Background(), domain.ForgotPasswordRequest{Email: "student@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	resetURL, err := url.Parse(forgot.ResetURL)
	if err != nil || resetURL.Query().Get("token") == "" {
		t.Fatalf("invalid reset URL: %q", forgot.ResetURL)
	}
	if err := service.ResetPassword(context.Background(), domain.ResetPasswordRequest{Token: resetURL.Query().Get("token"), NewPassword: "new-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(context.Background(), login.AccessToken); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("old session error = %v", err)
	}
	if _, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "old-password"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("old password error = %v", err)
	}
	if _, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "new-password"}); err != nil {
		t.Fatalf("new password login: %v", err)
	}
}

func TestLoginReplacesPreviousSession(t *testing.T) {
	service := newTestAuthService()
	_, err := service.Register(context.Background(), domain.RegisterRequest{Email: "student@example.com", Password: "strong-password", SchoolLevel: "SMA"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(context.Background(), first.AccessToken); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("first token error = %v", err)
	}
	claims, err := service.ValidateAccessToken(context.Background(), second.AccessToken)
	if err != nil {
		t.Fatalf("second token: %v", err)
	}
	if err := service.Logout(context.Background(), *claims); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(context.Background(), second.AccessToken); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("logged-out token error = %v", err)
	}
}

func TestInvalidRegistrationAndCredentials(t *testing.T) {
	service := newTestAuthService()
	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "invalid", Password: "short", SchoolLevel: "college"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("register error = %v", err)
	}
	if _, err := service.Login(context.Background(), domain.LoginRequest{Email: "missing@example.com", Password: "anything"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("login error = %v", err)
	}
}
