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
)

type AuthRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	UpdateProfile(ctx context.Context, userID string, update ProfileUpdate) error
	CreatePasswordResetToken(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (bool, error)
	ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) (string, error)
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
	ValidateAccessToken(ctx context.Context, token string) (*AuthClaims, error)
	ForgotPassword(ctx context.Context, input ForgotPasswordRequest) (*ForgotPasswordResponse, error)
	ResetPassword(ctx context.Context, input ResetPasswordRequest) error
}
