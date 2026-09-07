package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type TestimonialHandler struct {
	service domain.TestimonialService
	logger  *slog.Logger
}

func NewTestimonialHandler(service domain.TestimonialService, logger *slog.Logger) *TestimonialHandler {
	return &TestimonialHandler{service: service, logger: logger}
}

func (h *TestimonialHandler) Submit(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input struct {
		Quote string `json:"quote"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	t, err := h.service.Submit(r.Context(), claims.UserID, input.Quote)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTestimonialAlreadySubmitted):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrTestimonialEmptyQuote):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "submit testimonial", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"testimonial": t})
}

func (h *TestimonialHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	t, err := h.service.GetMine(r.Context(), claims.UserID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get my testimonial", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if t == nil {
		writeJSON(w, http.StatusOK, map[string]any{"testimonial": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"testimonial": t})
}

func (h *TestimonialHandler) ListApproved(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListApproved(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list approved testimonials", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"testimonials": list})
}

func (h *TestimonialHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := h.service.ListAll(r.Context(), status)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "admin list testimonials", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"testimonials": list})
}

func (h *TestimonialHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	t, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTestimonialNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "status must be approved, rejected, or pending")
		default:
			h.logger.ErrorContext(r.Context(), "admin update testimonial", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"testimonial": t})
}

func (h *TestimonialHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrTestimonialNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "admin delete testimonial", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
