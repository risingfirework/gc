package domain

import "time"

const (
	RoleStudent = "student"
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	BirthDate    string    `json:"birth_date"` // YYYY-MM-DD atau kosong
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	SchoolLevel  string    `json:"school_level"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	SchoolLevel string `json:"school_level"`
	Role        string `json:"role"` // optional; "student" (default) or "teacher"
}

type RegisterResponse struct {
	User UserResponse `json:"user"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id,omitempty"` // Accepted for client compatibility; JTI is the session authority.
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Message  string `json:"message"`
	ResetURL string `json:"reset_url,omitempty"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	BirthDate   string    `json:"birth_date"`
	Phone       string    `json:"phone"`
	Role        string    `json:"role"`
	SchoolLevel string    `json:"school_level"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateProfileRequest berisi data profil yang boleh diubah sendiri.
// current_password wajib diisi saat new_password disertakan.
type UpdateProfileRequest struct {
	Name            string `json:"name"`
	BirthDate       string `json:"birth_date"`
	Phone           string `json:"phone"`
	SchoolLevel     string `json:"school_level"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type AuthClaims struct {
	UserID    string
	Email     string
	Role      string
	JTI       string
	ExpiresAt time.Time
}
