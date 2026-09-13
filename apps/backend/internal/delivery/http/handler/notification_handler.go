package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

type NotificationHandler struct {
	service domain.NotificationService
	logger  *slog.Logger
}

func NewNotificationHandler(service domain.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{service: service, logger: logger}
}

func (h *NotificationHandler) userID(r *http.Request) string {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	return claims.UserID
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.service.List(r.Context(), h.userID(r), limit)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list notifications", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": items})
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.UnreadCount(r.Context(), h.userID(r))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "notification unread count", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unread_count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkRead(r.Context(), h.userID(r), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkAllRead(r.Context(), h.userID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
