package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID string
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(email)=LOWER($1) AND NOT EXISTS (
		SELECT 1 FROM password_reset_tokens pr WHERE pr.user_id=users.id AND pr.created_at>NOW()-INTERVAL '60 seconds'
	)`, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at=NOW() WHERE user_id=$1 AND used_at IS NULL`, userID); err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, userID, tokenHash, expiresAt); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepository) ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, now time.Time) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID string
	err = tx.QueryRow(ctx, `SELECT user_id FROM password_reset_tokens WHERE token_hash=$1 AND used_at IS NULL AND expires_at>$2 FOR UPDATE`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrInvalidResetToken
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET password_hash=$2,updated_at=NOW() WHERE id=$1`, userID, passwordHash); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at=$2 WHERE user_id=$1 AND used_at IS NULL`, userID, now); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// FindReferralAffiliate mengembalikan ID user affiliate pemilik kode rujukan.
func (r *UserRepository) FindReferralAffiliate(ctx context.Context, code string) (string, error) {
	var affiliateID string
	err := r.db.QueryRow(ctx, `SELECT id FROM affiliates WHERE referral_code = UPPER($1)`, strings.TrimSpace(code)).Scan(&affiliateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrReferralNotFound
	}
	if err != nil {
		return "", err
	}
	return affiliateID, nil
}

// InsertReferral mencatat atribusi first-touch; pengguna yang sama hanya
// dapat dirujuk sekali (unique constraint referrals_referred_user_uq).
func (r *UserRepository) InsertReferral(ctx context.Context, affiliateID, referredUserID string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO referrals(affiliate_id, referred_user_id) VALUES($1,$2)
		ON CONFLICT (referred_user_id) DO NOTHING`, affiliateID, referredUserID)
	return err
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (email, password_hash, role, school_level, teacher_ktp)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5,''),''))
		RETURNING id, COALESCE(name,''), COALESCE(to_char(birth_date,'YYYY-MM-DD'),''), COALESCE(phone,''),
		          COALESCE(teacher_ktp,''), COALESCE(teacher_verification_status,'pending'),
		          created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.SchoolLevel,
		user.TeacherKTP,
	).Scan(&user.ID, &user.Name, &user.BirthDate, &user.Phone, &user.TeacherKTP, &user.TeacherVerificationStatus, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailAlreadyExists
	}
	return err
}

const userColumns = `id, email, password_hash, COALESCE(name,''), COALESCE(to_char(birth_date,'YYYY-MM-DD'),''), COALESCE(phone,''), role, school_level, COALESCE(totp_secret_base32,''), totp_enabled, created_at, updated_at, COALESCE(teacher_ktp,''), COALESCE(teacher_verification_status,'pending'), COALESCE(teacher_rejection_reason,''), COALESCE(teacher_appeal_image,''), teacher_verified_at, COALESCE(teacher_simpkb_status,'pending'), COALESCE(teacher_simpkb_image,''), teacher_simpkb_checked_at`

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.BirthDate,
		&user.Phone,
		&user.Role,
		&user.SchoolLevel,
		&user.TOTPSecretBase32,
		&user.TOTPEnabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.TeacherKTP,
		&user.TeacherVerificationStatus,
		&user.TeacherRejectionReason,
		&user.TeacherAppealImage,
		&user.TeacherVerifiedAt,
		&user.TeacherSIMPKBStatus,
		&user.TeacherSIMPKBImage,
		&user.TeacherSIMPKBCheckedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.BirthDate, &user.Phone, &user.Role, &user.SchoolLevel, &user.TOTPSecretBase32, &user.TOTPEnabled, &user.CreatedAt, &user.UpdatedAt, &user.TeacherKTP, &user.TeacherVerificationStatus, &user.TeacherRejectionReason, &user.TeacherAppealImage, &user.TeacherVerifiedAt, &user.TeacherSIMPKBStatus, &user.TeacherSIMPKBImage, &user.TeacherSIMPKBCheckedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// SetTOTPSecret menyimpan seed baru; dipanggil saat setup sebelum diaktifkan.
func (r *UserRepository) SetTOTPSecret(ctx context.Context, userID, secretBase32 string) error {
	return r.userExec(ctx, userID, `UPDATE users SET totp_secret_base32=$2, updated_at=NOW() WHERE id=$1`, secretBase32)
}

// ConfirmTOTPSecret mengaktifkan 2FA setelah kode verifikasi lolos.
func (r *UserRepository) ConfirmTOTPSecret(ctx context.Context, userID string) error {
	return r.userExec(ctx, userID, `UPDATE users SET totp_enabled=TRUE, updated_at=NOW() WHERE id=$1`)
}

// ClearTOTP menghapus seed dan menonaktifkan 2FA.
func (r *UserRepository) ClearTOTP(ctx context.Context, userID string) error {
	return r.userExec(ctx, userID, `UPDATE users SET totp_secret_base32=NULL, totp_enabled=FALSE, updated_at=NOW() WHERE id=$1`)
}

func (r *UserRepository) userExec(ctx context.Context, userID, query string, args ...any) error {
	_, err := r.db.Exec(ctx, query, append([]any{userID}, args...)...)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID string, update domain.ProfileUpdate) error {
	_, err := r.db.Exec(ctx, `UPDATE users
		SET name=$2,
		    birth_date=NULLIF(NULLIF($3,''),'')::date,
		    phone=$4,
		    school_level=$5,
		    password_hash=COALESCE($6, password_hash),
		    updated_at=NOW()
		WHERE id=$1`,
		userID,
		update.Name,
		update.BirthDate,
		update.Phone,
		update.SchoolLevel,
		update.PasswordHash,
	)
	return err
}

var _ domain.AuthRepository = (*UserRepository)(nil)
