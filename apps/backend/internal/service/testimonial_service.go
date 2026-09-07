package service

import (
	"context"
	"strings"

	"tka/apps/backend/internal/domain"
)

type TestimonialService struct {
	repo domain.TestimonialRepository
}

func NewTestimonialService(repo domain.TestimonialRepository) *TestimonialService {
	return &TestimonialService{repo: repo}
}

func (s *TestimonialService) Submit(ctx context.Context, userID, quote string) (*domain.Testimonial, error) {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return nil, domain.ErrTestimonialEmptyQuote
	}
	return s.repo.Create(ctx, userID, quote)
}

func (s *TestimonialService) GetMine(ctx context.Context, userID string) (*domain.Testimonial, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *TestimonialService) ListAll(ctx context.Context, status string) ([]domain.Testimonial, error) {
	if status != "" {
		status = strings.ToLower(strings.TrimSpace(status))
	}
	return s.repo.ListAll(ctx, status)
}

func (s *TestimonialService) ListApproved(ctx context.Context) ([]domain.Testimonial, error) {
	return s.repo.ListApproved(ctx)
}

func (s *TestimonialService) UpdateStatus(ctx context.Context, id, status string) (*domain.Testimonial, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "approved" && status != "rejected" && status != "pending" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *TestimonialService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
