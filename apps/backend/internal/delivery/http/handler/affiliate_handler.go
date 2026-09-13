package handler

import (
	"log/slog"
	"net/http"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type AffiliateHandler struct {
	service domain.AffiliateService
	logger  *slog.Logger
}

func NewAffiliateHandler(service domain.AffiliateService, logger *slog.Logger) *AffiliateHandler {
	return &AffiliateHandler{service: service, logger: logger}
}

func (h *AffiliateHandler) affiliateID(r *http.Request) string {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	return claims.UserID
}

func (h *AffiliateHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetDashboard(r.Context(), h.affiliateID(r))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "affiliate dashboard", "error", err)
		writeError(w, 500, "internal server error")
		return
	}
	writeJSON(w, 200, result)
}

func (h *AffiliateHandler) UpdatePayoutAccount(w http.ResponseWriter, r *http.Request) {
	var input domain.TeacherPayoutAccount
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	item, err := h.service.UpdatePayoutAccount(r.Context(), h.affiliateID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_account": item})
}

func (h *AffiliateHandler) CreatePayoutRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Amount float64 `json:"amount"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePayoutRequest(r.Context(), h.affiliateID(r), input.Amount)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payout_request": item})
}

func (h *AffiliateHandler) CancelPayoutRequest(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.CancelPayoutRequest(r.Context(), h.affiliateID(r), chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_request": item})
}
