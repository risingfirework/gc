package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type PaymentSettingsRepository struct{ db *pgxpool.Pool }

func NewPaymentSettingsRepository(db *pgxpool.Pool) *PaymentSettingsRepository {
	return &PaymentSettingsRepository{db: db}
}

func (r *PaymentSettingsRepository) GetPaymentSettings(ctx context.Context) (*domain.EncryptedPaymentSettings, error) {
	var item domain.EncryptedPaymentSettings
	err := r.db.QueryRow(ctx, `SELECT secret_key_encrypted,webhook_token_encrypted,base_url,success_redirect_url,failure_redirect_url,updated_at
		FROM payment_provider_settings WHERE singleton=TRUE`).Scan(
		&item.SecretKeyCiphertext, &item.WebhookTokenCiphertext, &item.BaseURL,
		&item.SuccessRedirectURL, &item.FailureRedirectURL, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPaymentSettingsNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("payment settings: %w", err)
	}
	return &item, nil
}

func (r *PaymentSettingsRepository) UpsertPaymentSettings(ctx context.Context, settings *domain.EncryptedPaymentSettings) error {
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, `UPDATE payment_provider_settings
		SET provider='xendit',secret_key_encrypted=$1,webhook_token_encrypted=$2,
			base_url=$3,success_redirect_url=$4,failure_redirect_url=$5,updated_at=NOW()
		WHERE singleton=TRUE RETURNING updated_at`,
		settings.SecretKeyCiphertext, settings.WebhookTokenCiphertext, settings.BaseURL,
		settings.SuccessRedirectURL, settings.FailureRedirectURL).Scan(&updatedAt)
	if err != nil {
		return fmt.Errorf("upsert payment settings: %w", err)
	}
	settings.UpdatedAt = updatedAt
	return nil
}
