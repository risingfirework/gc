package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

type NotificationRepository struct{ db *pgxpool.Pool }

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, userID, title, body, link string) error {
	const query = `INSERT INTO notifications(user_id,title,body,link) VALUES($1,$2,$3,$4)`
	_, err := r.db.Exec(ctx, query, userID, title, body, link)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) List(ctx context.Context, userID string, limit int) ([]domain.Notification, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	const query = `SELECT id,user_id,title,body,link,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Notification, 0, limit)
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return items, nil
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	const query = `SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`
	var count int
	if err := r.db.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	const query = `UPDATE notifications SET read_at=NOW() WHERE id=$1 AND user_id=$2`
	tag, err := r.db.Exec(ctx, query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	const query = `UPDATE notifications SET read_at=NOW() WHERE user_id=$1 AND read_at IS NULL`
	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

var _ domain.NotificationRepository = (*NotificationRepository)(nil)
