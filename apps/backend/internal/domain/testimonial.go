package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTestimonialNotFound    = errors.New("testimoni tidak ditemukan")
	ErrTestimonialAlreadySubmitted = errors.New("anda sudah mengirim testimoni")
	ErrTestimonialEmptyQuote  = errors.New("testimoni tidak boleh kosong")
)

type Testimonial struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	UserName    string    `json:"user_name"`
	Quote       string    `json:"quote"`
	Status      string    `json:"status"` // pending, approved, rejected
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TestimonialRepository interface {
	Create(ctx context.Context, userID, quote string) (*Testimonial, error)
	GetByUserID(ctx context.Context, userID string) (*Testimonial, error)
	ListAll(ctx context.Context, status string) ([]Testimonial, error)
	ListApproved(ctx context.Context) ([]Testimonial, error)
	UpdateStatus(ctx context.Context, id, status string) (*Testimonial, error)
	Delete(ctx context.Context, id string) error
}

type TestimonialService interface {
	Submit(ctx context.Context, userID, quote string) (*Testimonial, error)
	GetMine(ctx context.Context, userID string) (*Testimonial, error)
	ListAll(ctx context.Context, status string) ([]Testimonial, error)
	ListApproved(ctx context.Context) ([]Testimonial, error)
	UpdateStatus(ctx context.Context, id, status string) (*Testimonial, error)
	Delete(ctx context.Context, id string) error
}
