package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	service domain.AdminService
	logger  *slog.Logger
}

func NewAdminHandler(service domain.AdminService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{service: service, logger: logger}
}
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetDashboard(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "admin dashboard", "error", err)
		writeError(w, 500, "internal server error")
		return
	}
	writeJSON(w, 200, result)
}
func (h *AdminHandler) SiteSettings(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetSiteSettings(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": result})
}
func (h *AdminHandler) UpdateSiteSettings(w http.ResponseWriter, r *http.Request) {
	var input domain.SiteSettings
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	result, err := h.service.UpdateSiteSettings(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": result})
}
func (h *AdminHandler) FinanceDashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetFinanceDashboard(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *AdminHandler) UpdateFinanceSettings(w http.ResponseWriter, r *http.Request) {
	var input domain.FinanceSettings
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	result, err := h.service.UpdateFinanceSettings(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": result})
}
func (h *AdminHandler) UpdateUserFinance(w http.ResponseWriter, r *http.Request) {
	var input domain.UserFinanceUpdateRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	result, err := h.service.UpdateUserFinance(r.Context(), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": result})
}
func (h *AdminHandler) PayTeacherCommissions(w http.ResponseWriter, r *http.Request) {
	var input domain.TeacherPayoutRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	result, err := h.service.PayTeacherCommissions(r.Context(), chi.URLParam(r, "id"), input.Reference)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payout": result})
}
func (h *AdminHandler) ReviewPayoutRequest(w http.ResponseWriter, r *http.Request) {
	var input domain.PayoutRequestReview
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	result, err := h.service.ReviewPayoutRequest(r.Context(), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_request": result})
}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var input domain.AdminUpdateUserRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	result, err := h.service.UpdateUser(r.Context(), claims.UserID, chi.URLParam(r, "id"), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, 422, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, 404, err.Error())
			return
		}
		h.logger.ErrorContext(r.Context(), "admin update user", "error", err)
		writeError(w, 500, "internal server error")
		return
	}
	writeJSON(w, 200, map[string]any{"user": result})
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminCreateUserRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateUser(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"user": item})
}
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	err := h.service.DeleteUser(r.Context(), claims.UserID, chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *AdminHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminPackageRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePackage(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"package": item})
}
func (h *AdminHandler) CreatePackageBundle(w http.ResponseWriter, r *http.Request) {
	var input domain.PackageBundle
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePackageBundle(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"package": item})
}
func (h *AdminHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminPackageRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdatePackage(r.Context(), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"package": item})
}
func (h *AdminHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeletePackage(r.Context(), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *AdminHandler) CreateExam(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminExamRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateExam(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"exam": item})
}
func (h *AdminHandler) UpdateExam(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminExamRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdateExam(r.Context(), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"exam": item})
}
func (h *AdminHandler) DeleteExam(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeleteExam(r.Context(), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *AdminHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminQuestionRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateQuestion(r.Context(), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"question": item})
}
func (h *AdminHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminQuestionRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdateQuestion(r.Context(), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"question": item})
}
func (h *AdminHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeleteQuestion(r.Context(), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *AdminHandler) ListMaster(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMaster(r.Context(), chi.URLParam(r, "category"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (h *AdminHandler) CreateMaster(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Nama string `json:"nama"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateMaster(r.Context(), chi.URLParam(r, "category"), input.Nama)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"item": item})
}
func (h *AdminHandler) UpdateMaster(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Nama string `json:"nama"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdateMaster(r.Context(), chi.URLParam(r, "category"), chi.URLParam(r, "id"), input.Nama)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"item": item})
}
func (h *AdminHandler) DeleteMaster(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeleteMaster(r.Context(), chi.URLParam(r, "category"), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}

func writeAdminError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrPayoutAccountRequired), errors.Is(err, domain.ErrPayoutMinimum):
		writeError(w, 422, err.Error())
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrPackageNotFound), errors.Is(err, domain.ErrExamNotFound), errors.Is(err, domain.ErrQuestionNotFound), errors.Is(err, domain.ErrMasterNotFound), errors.Is(err, domain.ErrPayoutRequestNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists), errors.Is(err, domain.ErrAdminConflict), errors.Is(err, domain.ErrDuplicateName), errors.Is(err, domain.ErrDuplicateKode), errors.Is(err, domain.ErrPayoutRequestActive), errors.Is(err, domain.ErrPayoutTransition):
		writeError(w, 409, err.Error())
	default:
		logger.ErrorContext(r.Context(), "admin mutation", "error", err)
		writeError(w, 500, "internal server error")
	}
	return true
}
