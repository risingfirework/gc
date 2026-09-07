package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tka/apps/backend/internal/domain"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
)

type AuthService struct {
	users        domain.AuthRepository
	sessions     domain.SessionRepository
	secret       []byte
	issuer       string
	tokenTTL     time.Duration
	now          func() time.Time
	google       domain.GoogleIDTokenVerifier
	resetBaseURL string
	environment  string
	resetMailer  domain.PasswordResetMailer
}

type accessTokenClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(users domain.AuthRepository, sessions domain.SessionRepository, jwtSecret, issuer string, tokenTTL time.Duration, google domain.GoogleIDTokenVerifier) *AuthService {
	return &AuthService{users: users, sessions: sessions, secret: []byte(jwtSecret), issuer: issuer, tokenTTL: tokenTTL, now: time.Now, google: google}
}

func (s *AuthService) ConfigurePasswordReset(baseURL, environment string, mailer domain.PasswordResetMailer) {
	s.resetBaseURL = strings.TrimRight(baseURL, "/")
	s.environment = strings.ToLower(strings.TrimSpace(environment))
	s.resetMailer = mailer
}

func (s *AuthService) ForgotPassword(ctx context.Context, input domain.ForgotPasswordRequest) (*domain.ForgotPasswordResponse, error) {
	const genericMessage = "Jika email terdaftar, tautan reset kata sandi telah dikirim."
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid email", domain.ErrInvalidInput)
	}
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return nil, fmt.Errorf("generate reset token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	hash := sha256.Sum256([]byte(token))
	created, err := s.users.CreatePasswordResetToken(ctx, email, hash[:], s.now().UTC().Add(15*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("create password reset: %w", err)
	}
	response := &domain.ForgotPasswordResponse{Message: genericMessage}
	if !created {
		return response, nil
	}
	resetURL := s.resetBaseURL + "/reset-password?token=" + url.QueryEscape(token)
	if s.resetMailer != nil {
		_ = s.resetMailer.SendPasswordReset(ctx, email, resetURL)
	}
	if s.environment == "development" || s.environment == "local" {
		response.ResetURL = resetURL
	}
	return response, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, input domain.ResetPasswordRequest) error {
	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" || len(input.Token) > 128 {
		return domain.ErrInvalidResetToken
	}
	decoded, err := base64.RawURLEncoding.DecodeString(input.Token)
	if err != nil || len(decoded) != 32 {
		return domain.ErrInvalidResetToken
	}
	if err := validatePassword(input.NewPassword); err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	tokenHash := sha256.Sum256([]byte(input.Token))
	userID, err := s.users.ResetPassword(ctx, tokenHash[:], string(passwordHash), s.now().UTC())
	if err != nil {
		return err
	}
	if err := s.sessions.DeleteSession(ctx, userID); err != nil {
		return fmt.Errorf("revoke reset sessions: %w", err)
	}
	return nil
}

func (s *AuthService) Register(ctx context.Context, input domain.RegisterRequest) (*domain.RegisterResponse, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid email", domain.ErrInvalidInput)
	}
	if err := validatePassword(input.Password); err != nil {
		return nil, err
	}
	level, err := normalizeSchoolLevel(input.SchoolLevel)
	if err != nil {
		return nil, err
	}
	role, err := normalizeRole(input.Role)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := domain.User{Email: email, PasswordHash: string(hash), Role: role, SchoolLevel: level}
	if err := s.users.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &domain.RegisterResponse{User: publicUser(user)}, nil
}

func (s *AuthService) Login(ctx context.Context, input domain.LoginRequest) (*domain.LoginResponse, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil || input.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return nil, domain.ErrInvalidCredentials
	}
	return s.issueSession(ctx, *user)
}

// LoginWithGoogle verifies a Google ID token and signs in the matching user, provisioning
// a new student account on first sign-in (school_level is required in that case).
func (s *AuthService) LoginWithGoogle(ctx context.Context, input domain.GoogleAuthRequest) (*domain.LoginResponse, error) {
	if s.google == nil {
		return nil, domain.ErrGoogleAuthUnavailable
	}
	identity, err := s.google.Verify(ctx, input.IDToken)
	if err != nil || !identity.EmailVerified {
		return nil, domain.ErrInvalidCredentials
	}
	email, err := normalizeEmail(identity.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("find user: %w", err)
		}
		level, levelErr := normalizeSchoolLevel(input.SchoolLevel)
		if levelErr != nil {
			return nil, fmt.Errorf("%w: school_level required", domain.ErrInvalidInput)
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(uuid.NewString()+uuid.NewString()), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("hash google user password: %w", hashErr)
		}
		newUser := domain.User{Email: email, PasswordHash: string(hash), Role: domain.RoleStudent, SchoolLevel: level}
		if err := s.users.Create(ctx, &newUser); err != nil {
			return nil, fmt.Errorf("create google user: %w", err)
		}
		user = &newUser
	}
	return s.issueSession(ctx, *user)
}

func (s *AuthService) issueSession(ctx context.Context, user domain.User) (*domain.LoginResponse, error) {
	now := s.now().UTC()
	expiresAt := now.Add(s.tokenTTL)
	jti := uuid.NewString()
	claims := accessTokenClaims{
		Email: user.Email, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: s.issuer, Subject: user.ID, ID: jti,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	// A successful login atomically replaces this user's previous JTI.
	if err := s.sessions.SetSession(ctx, user.ID, jti, s.tokenTTL); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}
	return &domain.LoginResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: int64(s.tokenTTL.Seconds()), User: publicUser(user)}, nil
}

func (s *AuthService) ValidateAccessToken(ctx context.Context, rawToken string) (*domain.AuthClaims, error) {
	claims := &accessTokenClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims,
		func(token *jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer), jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || claims.Subject == "" || claims.ID == "" || claims.ExpiresAt == nil {
		return nil, domain.ErrSessionInvalid
	}
	activeJTI, err := s.sessions.GetSession(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil, domain.ErrSessionInvalid
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(activeJTI), []byte(claims.ID)) != 1 {
		return nil, domain.ErrSessionInvalid
	}
	return &domain.AuthClaims{UserID: claims.Subject, Email: claims.Email, Role: claims.Role, JTI: claims.ID, ExpiresAt: claims.ExpiresAt.Time}, nil
}

func (s *AuthService) Logout(ctx context.Context, claims domain.AuthClaims) error {
	activeJTI, err := s.sessions.GetSession(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.ErrSessionInvalid
		}
		return fmt.Errorf("get session: %w", err)
	}
	// A replaced token cannot delete the newer device's session.
	if subtle.ConstantTimeCompare([]byte(activeJTI), []byte(claims.JTI)) != 1 {
		return domain.ErrSessionInvalid
	}
	if err := s.sessions.DeleteSession(ctx, claims.UserID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *AuthService) CurrentUser(ctx context.Context, claims domain.AuthClaims) (*domain.UserResponse, error) {
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("find current user: %w", err)
	}
	response := publicUser(*user)
	return &response, nil
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 320 {
		return "", domain.ErrInvalidInput
	}
	return email, nil
}

func validatePassword(password string) error {
	length := len([]byte(password))
	if length < minPasswordLength || length > maxPasswordLength {
		return fmt.Errorf("%w: password must contain 8-72 bytes", domain.ErrInvalidInput)
	}
	return nil
}

func normalizeSchoolLevel(value string) (string, error) {
	level := strings.ToUpper(strings.TrimSpace(value))
	switch level {
	case "SD", "SMP", "SMA":
		return level, nil
	default:
		return "", fmt.Errorf("%w: school_level must be SD, SMP, or SMA", domain.ErrInvalidInput)
	}
}

func normalizeRole(value string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(value))
	switch role {
	case "", domain.RoleStudent:
		return domain.RoleStudent, nil
	case domain.RoleTeacher:
		return domain.RoleTeacher, nil
	default:
		return "", fmt.Errorf("%w: role must be student or teacher", domain.ErrInvalidInput)
	}
}

func publicUser(user domain.User) domain.UserResponse {
	return domain.UserResponse{
		ID: user.ID, Email: user.Email, Name: user.Name, BirthDate: user.BirthDate, Phone: user.Phone,
		Role: user.Role, SchoolLevel: user.SchoolLevel, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

func (s *AuthService) UpdateProfile(ctx context.Context, claims domain.AuthClaims, input domain.UpdateProfileRequest) (*domain.UserResponse, error) {
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	name := strings.TrimSpace(input.Name)
	phone := strings.TrimSpace(input.Phone)
	schoolLevel := user.SchoolLevel
	if strings.TrimSpace(input.SchoolLevel) != "" {
		schoolLevel, err = normalizeSchoolLevel(input.SchoolLevel)
		if err != nil {
			return nil, err
		}
	}
	if len([]byte(name)) > 120 {
		return nil, fmt.Errorf("%w: name too long", domain.ErrInvalidInput)
	}
	if len(phone) > 20 {
		return nil, fmt.Errorf("%w: phone too long", domain.ErrInvalidInput)
	}
	if phone != "" {
		for _, ch := range phone {
			if !((ch >= '0' && ch <= '9') || ch == '+' || ch == '-' || ch == ' ') {
				return nil, fmt.Errorf("%w: phone invalid", domain.ErrInvalidInput)
			}
		}
	}
	var birthDate *string
	if strings.TrimSpace(input.BirthDate) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(input.BirthDate)); err != nil {
			return nil, fmt.Errorf("%w: birth_date must be YYYY-MM-DD", domain.ErrInvalidInput)
		}
		birth := strings.TrimSpace(input.BirthDate)
		birthDate = &birth
	}
	var passwordHash *string
	if input.NewPassword != "" {
		if input.CurrentPassword == "" {
			return nil, fmt.Errorf("%w: current_password required", domain.ErrInvalidInput)
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)) != nil {
			return nil, fmt.Errorf("%w: password lama salah", domain.ErrInvalidCredentials)
		}
		if err := validatePassword(input.NewPassword); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		h := string(hash)
		passwordHash = &h
	}
	update := domain.ProfileUpdate{
		Name: name, BirthDate: birthDate, Phone: phone, SchoolLevel: schoolLevel, PasswordHash: passwordHash,
	}
	if err := s.users.UpdateProfile(ctx, claims.UserID, update); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	updated, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("re-fetch user: %w", err)
	}
	response := publicUser(*updated)
	return &response, nil
}

var _ domain.AuthService = (*AuthService)(nil)
