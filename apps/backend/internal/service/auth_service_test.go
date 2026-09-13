package service

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

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

func (f *fakeUsers) FindReferralAffiliate(_ context.Context, code string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if code != "MITRA-TEST" {
		return "", domain.ErrReferralNotFound
	}
	return "affiliate-1", nil
}

func (f *fakeUsers) InsertReferral(_ context.Context, affiliateID, referredUserID string) error {
	return nil
}

func (f *fakeUsers) SetTOTPSecret(_ context.Context, userID, secretBase32 string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, user := range f.byEmail {
		if user.ID == userID {
			user.TOTPSecretBase32 = secretBase32
			f.byEmail[email] = user
			return nil
		}
	}
	return domain.ErrUserNotFound
}

func (f *fakeUsers) ConfirmTOTPSecret(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, user := range f.byEmail {
		if user.ID == userID {
			user.TOTPEnabled = true
			f.byEmail[email] = user
			return nil
		}
	}
	return domain.ErrUserNotFound
}

func (f *fakeUsers) ClearTOTP(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, user := range f.byEmail {
		if user.ID == userID {
			user.TOTPSecretBase32 = ""
			user.TOTPEnabled = false
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

func TestRegisterAppliesReferralCode(t *testing.T) {
	service := newTestAuthService()
	registered, err := service.Register(context.Background(), domain.RegisterRequest{Email: "referred@example.com", Password: "strong-password", SchoolLevel: "SMP", ReferralCode: "MITRA-TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.Role != domain.RoleStudent {
		t.Fatalf("unexpected role: %s", registered.User.Role)
	}
}

func TestRegisterRejectsUnknownReferralCode(t *testing.T) {
	service := newTestAuthService()
	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "nobody@example.com", Password: "strong-password", SchoolLevel: "SMA", ReferralCode: "MITRA-MISSING"}); !errors.Is(err, domain.ErrInvalidReferralCode) {
		t.Fatalf("register error = %v, want ErrInvalidReferralCode", err)
	}
}

func TestRegisterRejectsReferralForTeacher(t *testing.T) {
	service := newTestAuthService()
	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "teacher@example.com", Password: "strong-password", SchoolLevel: "SMA", Role: "teacher", ReferralCode: "MITRA-TEST"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("register error = %v, want ErrInvalidInput", err)
	}
}

func addUser(t *testing.T, users *fakeUsers, email, password, role string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{Email: email, PasswordHash: string(hash), Role: role, SchoolLevel: "SMA"}
	if err := users.Create(context.Background(), &user); err != nil {
		t.Fatal(err)
	}
}

func computeCurrentTOTP(t *testing.T, secret string) string {
	t.Helper()
	code, err := totpCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func Test2FALoginFlow(t *testing.T) {
	users := &fakeUsers{byEmail: make(map[string]domain.User)}
	sessions := &fakeSessions{byUser: make(map[string]string)}
	service := NewAuthService(users, sessions, "a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil)
	addUser(t, users, "finance@example.com", "strong-password", domain.RoleFinance)

	setup, err := service.Setup2FA(context.Background(), domain.AuthClaims{UserID: "user-1"})
	if err != nil {
		t.Fatalf("setup 2fa: %v", err)
	}
	if setup.Secret == "" || setup.QRDataURL == "" || setup.OTPAuthURL == "" {
		t.Fatalf("unexpected setup response: %+v", setup)
	}

	if err := service.Enable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, setup.Secret[:1]); err != nil && !errors.Is(err, domain.ErrInvalid2FACode) {
		t.Fatalf("enable with bad code error = %v", err)
	}
	if err := service.Enable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, computeCurrentTOTP(t, setup.Secret)); err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}
	if err := service.Enable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, computeCurrentTOTP(t, setup.Secret)); !errors.Is(err, domain.Err2FAAlreadyEnabled) {
		t.Fatalf("double enable error = %v", err)
	}

	first, err := service.Login(context.Background(), domain.LoginRequest{Email: "finance@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatalf("step-one login: %v", err)
	}
	if !first.Requires2FA || first.MFAToken == "" || first.AccessToken != "" {
		t.Fatalf("unexpected step-one response: %+v", first)
	}

	if _, err := service.Verify2FA(context.Background(), first.MFAToken, "000000"); !errors.Is(err, domain.ErrInvalid2FACode) {
		t.Fatalf("verify with wrong code error = %v", err)
	}
	if _, err := service.Verify2FA(context.Background(), "not-a-token", computeCurrentTOTP(t, setup.Secret)); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("verify with bad mfa token error = %v", err)
	}

	second, err := service.Verify2FA(context.Background(), first.MFAToken, computeCurrentTOTP(t, setup.Secret))
	if err != nil {
		t.Fatalf("verify 2fa: %v", err)
	}
	if second.AccessToken == "" {
		t.Fatalf("missing access token after 2fa: %+v", second)
	}
	if _, err := service.ValidateAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("validate token after 2fa: %v", err)
	}

	// Disable membutuhkan kode valid saat ini.
	if err := service.Disable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, "000000"); !errors.Is(err, domain.ErrInvalid2FACode) {
		t.Fatalf("disable with wrong code error = %v", err)
	}
	if err := service.Disable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, computeCurrentTOTP(t, setup.Secret)); err != nil {
		t.Fatalf("disable 2fa: %v", err)
	}
	after, err := service.Login(context.Background(), domain.LoginRequest{Email: "finance@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatalf("login after disable: %v", err)
	}
	if after.Requires2FA || after.AccessToken == "" {
		t.Fatalf("login after disable returned = %+v", after)
	}
}

func Test2FARestrictedToPrivilegedRoles(t *testing.T) {
	users := &fakeUsers{byEmail: make(map[string]domain.User)}
	sessions := &fakeSessions{byUser: make(map[string]string)}
	service := NewAuthService(users, sessions, "a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil)
	addUser(t, users, "student@example.com", "strong-password", domain.RoleStudent)

	if _, err := service.Setup2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}); !errors.Is(err, domain.Err2FAUnsupportedRole) {
		t.Fatalf("setup2fa for student error = %v", err)
	}
	if err := service.Enable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, "123456"); !errors.Is(err, domain.Err2FAUnsupportedRole) {
		t.Fatalf("enable2fa for student error = %v", err)
	}
	if err := service.Disable2FA(context.Background(), domain.AuthClaims{UserID: "user-1"}, "123456"); !errors.Is(err, domain.Err2FAUnsupportedRole) {
		t.Fatalf("disable2fa for student error = %v", err)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	users := &fakeUsers{byEmail: make(map[string]domain.User)}
	sessions := &fakeSessions{byUser: make(map[string]string)}
	refreshes := &fakeRefreshTokens{byHash: make(map[string]domain.RefreshToken), userIDs: make(map[string][]string)}
	service := NewAuthService(users, sessions, "a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil)
	service.ConfigureRefreshTokens(refreshes, 30*24*time.Hour)

	if _, err := service.Refresh(context.Background(), "not-a-token"); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("refresh before any login error = %v", err)
	}

	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "student@example.com", Password: "strong-password", SchoolLevel: "SMA"}); err != nil {
		t.Fatal(err)
	}
	first, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	if first.RefreshToken == "" || first.RefreshExpiresIn <= 0 {
		t.Fatalf("login response lacks refresh token: %+v", first)
	}

	second, err := service.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.AccessToken == "" || second.RefreshToken == "" || second.RefreshToken == first.RefreshToken {
		t.Fatalf("refresh did not rotate session: %+v", second)
	}
	if _, err := service.ValidateAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("validate refreshed token: %v", err)
	}
	if _, err := service.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("reusing revoked refresh token error = %v", err)
	}

	third, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateAccessToken(context.Background(), third.AccessToken)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := service.Logout(context.Background(), *claims); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Refresh(context.Background(), third.RefreshToken); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("refresh after logout error = %v", err)
	}
}

func TestRefreshDisabledWithoutRepository(t *testing.T) {
	users := &fakeUsers{byEmail: make(map[string]domain.User)}
	sessions := &fakeSessions{byUser: make(map[string]string)}
	service := NewAuthService(users, sessions, "a-test-secret-that-is-at-least-32-bytes-long", "tka-test", 15*time.Minute, nil)
	if _, err := service.Register(context.Background(), domain.RegisterRequest{Email: "student@example.com", Password: "strong-password", SchoolLevel: "SMA"}); err != nil {
		t.Fatal(err)
	}
	login, err := service.Login(context.Background(), domain.LoginRequest{Email: "student@example.com", Password: "strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	if login.RefreshToken != "" {
		t.Fatalf("unexpected refresh token without repository: %+v", login)
	}
	if _, err := service.Refresh(context.Background(), "whatever"); !errors.Is(err, domain.ErrSessionInvalid) {
		t.Fatalf("refresh without repository error = %v", err)
	}
}

type fakeRefreshTokens struct {
	mu      sync.Mutex
	byHash  map[string]domain.RefreshToken
	userIDs map[string][]string
}

func (f *fakeRefreshTokens) Create(_ context.Context, token *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := string(token.TokenHash)
	if _, exists := f.byHash[key]; exists {
		return errors.New("duplicate refresh hash")
	}
	f.byHash[key] = *token
	f.userIDs[token.UserID] = append(f.userIDs[token.UserID], key)
	return nil
}

func (f *fakeRefreshTokens) FindByHash(_ context.Context, tokenHash []byte) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	token, exists := f.byHash[string(tokenHash)]
	if !exists {
		return nil, domain.ErrSessionInvalid
	}
	return &token, nil
}

func (f *fakeRefreshTokens) Revoke(_ context.Context, tokenHash []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := string(tokenHash)
	token, exists := f.byHash[key]
	if !exists {
		return nil
	}
	token.RevokedAt = newTimePtr()
	f.byHash[key] = token
	return nil
}

func (f *fakeRefreshTokens) RevokeAllForUser(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, key := range f.userIDs[userID] {
		token := f.byHash[key]
		token.RevokedAt = newTimePtr()
		f.byHash[key] = token
	}
	return nil
}

func newTimePtr() *time.Time {
	now := time.Now()
	return &now
}
