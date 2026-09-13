package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.db.Exec(ctx, `INSERT INTO refresh_tokens(user_id, token_hash, expires_at) VALUES($1,$2,$3)`,
		token.UserID, token.TokenHash, token.ExpiresAt)
	return err
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash []byte) (*domain.RefreshToken, error) {
	row := r.db.QueryRow(ctx, `SELECT id, user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash=$1`, tokenHash)
	token := &domain.RefreshToken{}
	err := row.Scan(&token.ID, &token.UserID, &token.ExpiresAt, &token.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash []byte) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}
