package handler

import (
	"log/slog"
	"net/http"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type CBTAccessHandler struct {
	service domain.CBTAccessService
	logger  *slog.Logger
}

func NewCBTAccessHandler(service domain.CBTAccessService, logger *slog.Logger) *CBTAccessHandler {
	return &CBTAccessHandler{service: service, logger: logger}
}

func (h *CBTAccessHandler) actor(r *http.Request) (string, string) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		return "", ""
	}
	return claims.UserID, claims.Email
}

func (h *CBTAccessHandler) AdminSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetSettings(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *CBTAccessHandler) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input domain.CBTAccessSettings
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := h.actor(r)
	settings, err := h.service.UpdateSettings(r.Context(), actorID, actorEmail, input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *CBTAccessHandler) AdminPayments(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	result, err := h.service.ListPayments(r.Context(), r.URL.Query().Get("status"), page, perPage)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result.Items, "page": result.Page, "count": result.Count, "total": result.Total})
}

func (h *CBTAccessHandler) AdminApprovePayment(w http.ResponseWriter, r *http.Request) {
	var input domain.CBTPaymentReviewRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := h.actor(r)
	item, err := h.service.ApprovePayment(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": item})
}

func (h *CBTAccessHandler) AdminRejectPayment(w http.ResponseWriter, r *http.Request) {
	var input domain.CBTPaymentReviewRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := h.actor(r)
	item, err := h.service.RejectPayment(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": item})
}

func (h *CBTAccessHandler) AdminTeacherAccess(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	query := r.URL.Query()
	result, err := h.service.ListTeacherAccess(r.Context(), query.Get("status"), query.Get("q"), page, perPage)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result.Items, "page": result.Page, "count": result.Count, "total": result.Total})
}

func (h *CBTAccessHandler) AdminGrantAccess(w http.ResponseWriter, r *http.Request) {
	var input domain.CBTAccessGrantRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := h.actor(r)
	if err := h.service.GrantAccess(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input); writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"granted": true})
}

func (h *CBTAccessHandler) AdminRevokeAccess(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := h.actor(r)
	if err := h.service.RevokeAccess(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input.Reason); writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (h *CBTAccessHandler) TeacherStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.actor(r)
	status, err := h.service.GetTeacherStatus(r.Context(), userID)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access": status})
}

func (h *CBTAccessHandler) TeacherPayments(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.actor(r)
	items, err := h.service.ListMyPayments(r.Context(), userID)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CBTAccessHandler) TeacherSubmitPayment(w http.ResponseWriter, r *http.Request) {
	var input domain.CBTPaymentRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	userID, _ := h.actor(r)
	item, err := h.service.SubmitPayment(r.Context(), userID, input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment": item})
}
