package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrInvalidInput          = errors.New("invalid input")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionInvalid        = errors.New("device limit reached or session invalid")
	ErrGoogleAuthUnavailable = errors.New("google sign-in is not configured")
	ErrInvalidResetToken     = errors.New("tautan reset tidak valid atau sudah kedaluwarsa")
	ErrReferralNotFound      = errors.New("kode rujukan tidak ditemukan")
	ErrInvalidReferralCode   = errors.New("kode rujukan tidak valid")
	Err2FANotEnabled         = errors.New("2fa tidak aktif")
	Err2FAAlreadyEnabled     = errors.New("2fa sudah aktif")
	ErrInvalid2FACode        = errors.New("kode autentikator tidak valid")
	Err2FAUnsupportedRole    = errors.New("2fa hanya untuk akun owner dan finance")
)

type AuthRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	UpdateProfile(ctx context.Context, userID string, update ProfileUpdate) error
	CreatePasswordResetToken(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (bool, error)
	ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) (string, error)
	FindReferralAffiliate(ctx context.Context, code string) (string, error)
	InsertReferral(ctx context.Context, affiliateID, referredUserID string) error
	SetTOTPSecret(ctx context.Context, userID, secretBase32 string) error
	ConfirmTOTPSecret(ctx context.Context, userID string) error
	ClearTOTP(ctx context.Context, userID string) error
}

// ProfileUpdate menyimpan perubahan field profil yang aman diupdate langsung.
type ProfileUpdate struct {
	Name         string
	BirthDate    *string // pointer agar NULL bisa dibedakan dari string kosong
	Phone        string
	SchoolLevel  string
	PasswordHash *string
}

type SessionRepository interface {
	SetSession(ctx context.Context, userID, jti string, ttl time.Duration) error
	GetSession(ctx context.Context, userID string) (string, error)
	DeleteSession(ctx context.Context, userID string) error
}

// RefreshToken adalah baris token penyegar yang disimpan di PostgreSQL
// (bukan Redis) agar tetap berlaku walau cache di-flush. Hanya hash-nya yang
// disimpan, sehingga kebocoran tabel tidak mengekspos token yang bisa dipakai.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	FindByHash(ctx context.Context, tokenHash []byte) (*RefreshToken, error)
	Revoke(ctx context.Context, tokenHash []byte) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

// GoogleIdentity is the verified subset of claims extracted from a Google ID token.
type GoogleIdentity struct {
	Email         string
	EmailVerified bool
}

type GoogleIDTokenVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleIdentity, error)
}

type PasswordResetMailer interface {
	SendPasswordReset(ctx context.Context, recipient, resetURL string) error
}

type GoogleAuthRequest struct {
	IDToken     string `json:"id_token"`
	SchoolLevel string `json:"school_level"`
}

type AuthService interface {
	Register(ctx context.Context, input RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, input LoginRequest) (*LoginResponse, error)
	LoginWithGoogle(ctx context.Context, input GoogleAuthRequest) (*LoginResponse, error)
	Logout(ctx context.Context, claims AuthClaims) error
	CurrentUser(ctx context.Context, claims AuthClaims) (*UserResponse, error)
	UpdateProfile(ctx context.Context, claims AuthClaims, input UpdateProfileRequest) (*UserResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*LoginResponse, error)
	ValidateAccessToken(ctx context.Context, token string) (*AuthClaims, error)
	ForgotPassword(ctx context.Context, input ForgotPasswordRequest) (*ForgotPasswordResponse, error)
	ResetPassword(ctx context.Context, input ResetPasswordRequest) error
	Verify2FA(ctx context.Context, mfaToken, code string) (*LoginResponse, error)
	Setup2FA(ctx context.Context, claims AuthClaims) (*TwoFactorSetupResponse, error)
	Enable2FA(ctx context.Context, claims AuthClaims, code string) error
	Disable2FA(ctx context.Context, claims AuthClaims, code string) error
}

// RateLimiter menghitung pemakaian per kunci (mis. IP) agar batas per
// jendela waktu dapat ditegakkan di Redis.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}
