package postgres

import (
	"context"
	"errors"
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

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (email, password_hash, role, school_level)
		VALUES ($1, $2, $3, $4)
		RETURNING id, COALESCE(name,''), COALESCE(to_char(birth_date,'YYYY-MM-DD'),''), COALESCE(phone,''), created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.SchoolLevel,
	).Scan(&user.ID, &user.Name, &user.BirthDate, &user.Phone, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailAlreadyExists
	}
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, COALESCE(name,''), COALESCE(to_char(birth_date,'YYYY-MM-DD'),''), COALESCE(phone,''), role, school_level, created_at, updated_at
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
		&user.CreatedAt,
		&user.UpdatedAt,
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
	const query = `SELECT id, email, password_hash, COALESCE(name,''), COALESCE(to_char(birth_date,'YYYY-MM-DD'),''), COALESCE(phone,''), role, school_level, created_at, updated_at FROM users WHERE id = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.BirthDate, &user.Phone, &user.Role, &user.SchoolLevel, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
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
