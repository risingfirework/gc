package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

type AnalyticsHandler struct {
	service domain.ExamAnalyticsService
	logger  *slog.Logger
}

func NewAnalyticsHandler(service domain.ExamAnalyticsService, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{service: service, logger: logger}
}

func (h *AnalyticsHandler) Result(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	response, err := h.service.GetResult(r.Context(), claims.UserID, chi.URLParam(r, "id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserExamNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrExamStillRunning):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrPembahasanNotPublished):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "get exam result", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AnalyticsHandler) GlobalRanking(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 50)
	mode := strings.TrimSpace(query.Get("mode"))
	entries, err := h.service.GetGlobalRanking(r.Context(), claims.UserID, strings.TrimSpace(query.Get("jenjang")), mode, page, perPage)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "jenjang must be SD, SMP, or SMA")
		default:
			h.logger.ErrorContext(r.Context(), "get global ranking", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": entries.Items, "page": entries.Page, "count": entries.Count, "total": entries.Total})
}
