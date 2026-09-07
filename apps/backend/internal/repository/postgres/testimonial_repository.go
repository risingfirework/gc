package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type TestimonialRepository struct{ db *pgxpool.Pool }

func NewTestimonialRepository(db *pgxpool.Pool) *TestimonialRepository {
	return &TestimonialRepository{db: db}
}

const testimonialColumns = `id, user_id, COALESCE(email,''), COALESCE(name,''), quote, status, created_at, updated_at`

func (r *TestimonialRepository) Create(ctx context.Context, userID, quote string) (*domain.Testimonial, error) {
	var t domain.Testimonial
	err := r.db.QueryRow(ctx, `
		INSERT INTO testimonials (user_id, quote)
		VALUES ($1, $2)
		RETURNING id, user_id, '', '', quote, status, created_at, updated_at`,
		userID, quote,
	).Scan(&t.ID, &t.UserID, &t.UserEmail, &t.UserName, &t.Quote, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("%w: anda sudah mengirim testimoni", domain.ErrTestimonialAlreadySubmitted)
		}
		return nil, fmt.Errorf("insert testimonial: %w", err)
	}
	// fetch user email/name
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(email,''), COALESCE(name,'') FROM users WHERE id=$1`, userID).Scan(&t.UserEmail, &t.UserName)
	return &t, nil
}

func (r *TestimonialRepository) GetByUserID(ctx context.Context, userID string) (*domain.Testimonial, error) {
	var t domain.Testimonial
	err := r.db.QueryRow(ctx, `
		SELECT t.id, t.user_id, u.email, COALESCE(u.name,''), t.quote, t.status, t.created_at, t.updated_at
		FROM testimonials t JOIN users u ON u.id = t.user_id
		WHERE t.user_id = $1
		ORDER BY t.created_at DESC LIMIT 1`, userID,
	).Scan(&t.ID, &t.UserID, &t.UserEmail, &t.UserName, &t.Quote, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get testimonial by user: %w", err)
	}
	return &t, nil
}

func (r *TestimonialRepository) ListAll(ctx context.Context, status string) ([]domain.Testimonial, error) {
	query := `SELECT t.id, t.user_id, u.email, COALESCE(u.name,''), t.quote, t.status, t.created_at, t.updated_at
		FROM testimonials t JOIN users u ON u.id = t.user_id`
	args := []any{}
	if status != "" {
		query += ` WHERE t.status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY t.created_at DESC`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list testimonials: %w", err)
	}
	defer rows.Close()
	var result []domain.Testimonial
	for rows.Next() {
		var t domain.Testimonial
		if err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.UserName, &t.Quote, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *TestimonialRepository) ListApproved(ctx context.Context) ([]domain.Testimonial, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.user_id, u.email, COALESCE(u.name,''), t.quote, t.status, t.created_at, t.updated_at
		FROM testimonials t JOIN users u ON u.id = t.user_id
		WHERE t.status = 'approved'
		ORDER BY t.created_at DESC LIMIT 20`)
	if err != nil {
		return nil, fmt.Errorf("list approved testimonials: %w", err)
	}
	defer rows.Close()
	var result []domain.Testimonial
	for rows.Next() {
		var t domain.Testimonial
		if err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.UserName, &t.Quote, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *TestimonialRepository) UpdateStatus(ctx context.Context, id, status string) (*domain.Testimonial, error) {
	var t domain.Testimonial
	err := r.db.QueryRow(ctx, `
		UPDATE testimonials SET status=$2, updated_at=NOW()
		WHERE id=$1
		RETURNING id, user_id, '', '', quote, status, created_at, updated_at`,
		id, status,
	).Scan(&t.ID, &t.UserID, &t.UserEmail, &t.UserName, &t.Quote, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTestimonialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update testimonial status: %w", err)
	}
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(email,''), COALESCE(name,'') FROM users WHERE id=$1`, t.UserID).Scan(&t.UserEmail, &t.UserName)
	return &t, nil
}

func (r *TestimonialRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM testimonials WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete testimonial: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTestimonialNotFound
	}
	return nil
}
