package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type MediaRepository struct{ db *pgxpool.Pool }

func NewMediaRepository(db *pgxpool.Pool) *MediaRepository { return &MediaRepository{db: db} }

func (r *MediaRepository) Create(ctx context.Context, asset domain.MediaAsset) (*domain.MediaAsset, error) {
	const query = `
		INSERT INTO media_assets(owner_id,kind,mime_type,size_bytes,original_name,storage_key)
		VALUES($1,$2,$3,$4,$5,$6)
		RETURNING id,owner_id,kind,mime_type,size_bytes,original_name,storage_key,created_at`
	var result domain.MediaAsset
	err := r.db.QueryRow(ctx, query,
		asset.OwnerID, asset.Kind, asset.MimeType, asset.SizeBytes, asset.OriginalName, asset.StorageKey,
	).Scan(
		&result.ID, &result.OwnerID, &result.Kind, &result.MimeType,
		&result.SizeBytes, &result.OriginalName, &result.StorageKey, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create media asset: %w", err)
	}
	return &result, nil
}

func (r *MediaRepository) Get(ctx context.Context, id string) (*domain.MediaAsset, error) {
	const query = `
		SELECT id,owner_id,kind,mime_type,size_bytes,original_name,storage_key,created_at
		FROM media_assets WHERE id=$1`
	var result domain.MediaAsset
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID, &result.OwnerID, &result.Kind, &result.MimeType,
		&result.SizeBytes, &result.OriginalName, &result.StorageKey, &result.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrMediaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get media asset: %w", err)
	}
	return &result, nil
}

func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM media_assets WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete media asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMediaNotFound
	}
	return nil
}
