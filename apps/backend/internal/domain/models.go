package domain

import "time"

const (
	RoleStudent   = "student"
	RoleAdmin     = "admin"
	RoleTeacher   = "teacher"
	RoleOwner     = "owner"
	RoleFinance   = "finance"
	RoleAffiliate = "affiliate"
)

// Status verifikasi guru oleh operator/admin.
const (
	TeacherVerificationPending  = "pending"
	TeacherVerificationApproved = "approved"
	TeacherVerificationRejected = "rejected"
)

// Hasil pengecekan NIK otomatis ke portal SIMPKB.
const (
	SIMPKBStatusPending  = "pending"
	SIMPKBStatusChecking = "checking"
	SIMPKBStatusFound    = "found"
	SIMPKBStatusNotFound = "not_found"
	SIMPKBStatusError    = "error"
)

type User struct {
	ID                        string `json:"id"`
	Email                     string `json:"email"`
	PasswordHash              string `json:"-"`
	Name                      string `json:"name"`
	BirthDate                 string `json:"birth_date"` // YYYY-MM-DD atau kosong
	Phone                     string `json:"phone"`
	Role                      string `json:"role"`
	SchoolLevel               string `json:"school_level"`
	TOTPSecretBase32          string `json:"-"`
	TOTPEnabled               bool   `json:"-"`
	TeacherKTP                string `json:"-"`
	TeacherVerificationStatus string
	TeacherRejectionReason    string
	TeacherAppealImage        string
	TeacherSIMPKBStatus       string
	TeacherSIMPKBImage        string
	TeacherSIMPKBCheckedAt    *time.Time
	TeacherVerifiedAt         *time.Time
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	SchoolLevel  string `json:"school_level"`
	Role         string `json:"role"`          // optional; "student" (default) or "teacher"
	TeacherKTP   string `json:"teacher_ktp"`   // wajib untuk role teacher; nomor NIK pada KTP
	ReferralCode string `json:"referral_code"` // optional; kode affiliate saat mendaftar sebagai siswa
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
	AccessToken      string       `json:"access_token,omitempty"`
	TokenType        string       `json:"token_type,omitempty"`
	ExpiresIn        int64        `json:"expires_in,omitempty"`
	RefreshToken     string       `json:"refresh_token,omitempty"` // token penyegar sekali-pakai berumur panjang
	RefreshExpiresIn int64        `json:"refresh_expires_in,omitempty"`
	User             UserResponse `json:"user"`
	Requires2FA      bool         `json:"requires_2fa,omitempty"` // true bila akun memerlukan kode 2FA sebelum sesi diterbitkan
	MFAToken         string       `json:"mfa_token,omitempty"`    // token sekali-pakai pendek untuk melengkapi login 2FA
}

// RefreshRequest adalah token penyegar sekali pakai untuk mendapatkan sesi baru.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
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
	ID                        string    `json:"id"`
	Email                     string    `json:"email"`
	Name                      string    `json:"name"`
	BirthDate                 string    `json:"birth_date"`
	Phone                     string    `json:"phone"`
	Role                      string    `json:"role"`
	SchoolLevel               string    `json:"school_level"`
	ReferralCode              string    `json:"referral_code,omitempty"` // hanya terisi untuk role affiliate
	TOTPEnabled               bool      `json:"totp_enabled,omitempty"`  // hanya relevan untuk role owner/finance
	TeacherKTP                string    `json:"teacher_ktp,omitempty"`
	TeacherVerificationStatus string    `json:"teacher_verification_status,omitempty"`
	TeacherRejectionReason    string    `json:"teacher_rejection_reason,omitempty"`
	TeacherAppealImage        string    `json:"teacher_appeal_image,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// TwoFactorCodeRequest adalah kode TOTP 6 digit dari aplikasi autentikator.
type TwoFactorCodeRequest struct {
	Code string `json:"code"`
}

// Verify2FARequest melengkapi login dua langkah: mfa_token dari langkah
// pertama + kode TOTP sekali ini.
type Verify2FARequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

type TwoFactorSetupResponse struct {
	Secret     string `json:"secret"`      // seed base32 untuk dimasukkan manual
	OTPAuthURL string `json:"otpauth_url"` // URI standar otpauth
	QRDataURL  string `json:"qr_data_url"` // gambar QR data-URI untuk dipindai
	Enabled    bool   `json:"enabled"`
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
