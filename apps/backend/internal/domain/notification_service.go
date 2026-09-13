package domain

import (
	"context"
	"fmt"
)

type NotificationService interface {
	List(ctx context.Context, userID string, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type notificationService struct {
	repository NotificationRepository
}

func NewNotificationService(repository NotificationRepository) NotificationService {
	return &notificationService{repository: repository}
}

func (s *notificationService) List(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if userID == "" {
		return nil, ErrNotificationNotFound
	}
	items, err := s.repository.List(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (s *notificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	if userID == "" {
		return 0, ErrNotificationNotFound
	}
	return s.repository.UnreadCount(ctx, userID)
}

func (s *notificationService) MarkRead(ctx context.Context, userID, notificationID string) error {
	if userID == "" || notificationID == "" {
		return ErrNotificationNotFound
	}
	return s.repository.MarkRead(ctx, userID, notificationID)
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrNotificationNotFound
	}
	return s.repository.MarkAllRead(ctx, userID)
}
