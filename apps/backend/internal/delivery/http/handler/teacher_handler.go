package handler

import (
	"log/slog"
	"net/http"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type TeacherHandler struct {
	service domain.TeacherService
	logger  *slog.Logger
}

func NewTeacherHandler(service domain.TeacherService, logger *slog.Logger) *TeacherHandler {
	return &TeacherHandler{service: service, logger: logger}
}

func (h *TeacherHandler) publisherID(r *http.Request) string {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	return claims.UserID
}

func (h *TeacherHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetDashboard(r.Context(), h.publisherID(r))
	if err != nil {
		h.logger.ErrorContext(r.Context(), "teacher dashboard", "error", err)
		writeError(w, 500, "internal server error")
		return
	}
	writeJSON(w, 200, result)
}
func (h *TeacherHandler) UpdatePayoutAccount(w http.ResponseWriter, r *http.Request) {
	var input domain.TeacherPayoutAccount
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	item, err := h.service.UpdatePayoutAccount(r.Context(), h.publisherID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_account": item})
}
func (h *TeacherHandler) CreatePayoutRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Amount float64 `json:"amount"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePayoutRequest(r.Context(), h.publisherID(r), input.Amount)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payout_request": item})
}
func (h *TeacherHandler) CancelPayoutRequest(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.CancelPayoutRequest(r.Context(), h.publisherID(r), chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_request": item})
}

// Appeal menerima bukti sanggah guru (gambar/surat keterangan mengajar) saat
// pendaftarannya ditolak, lalu mengembalikan status ke tahap verifikasi.
func (h *TeacherHandler) Appeal(w http.ResponseWriter, r *http.Request) {
	var input domain.TeacherAppealRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := h.service.Appeal(r.Context(), h.publisherID(r), input); err != nil {
		if writeAdminError(h.logger, w, r, err) {
			return
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": domain.TeacherVerificationPending})
}
func (h *TeacherHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminPackageRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePackage(r.Context(), h.publisherID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"package": item})
}
func (h *TeacherHandler) CreatePackageBundle(w http.ResponseWriter, r *http.Request) {
	var input domain.PackageBundle
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreatePackageBundle(r.Context(), h.publisherID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"package": item})
}
func (h *TeacherHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminPackageRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdatePackage(r.Context(), h.publisherID(r), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"package": item})
}
func (h *TeacherHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeletePackage(r.Context(), h.publisherID(r), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *TeacherHandler) CreateExam(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminExamRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateExam(r.Context(), h.publisherID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"exam": item})
}
func (h *TeacherHandler) UpdateExam(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminExamRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdateExam(r.Context(), h.publisherID(r), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"exam": item})
}
func (h *TeacherHandler) DeleteExam(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeleteExam(r.Context(), h.publisherID(r), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *TeacherHandler) CBTSettings(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCBTPublishSettings(r.Context(), h.publisherID(r))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *TeacherHandler) SetCBTPublish(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PublishPembahasan bool `json:"publish_pembahasan"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	item, err := h.service.SetExamPublishPembahasan(r.Context(), h.publisherID(r), chi.URLParam(r, "id"), input.PublishPembahasan)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *TeacherHandler) CBTParticipants(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCBTParticipants(r.Context(), h.publisherID(r), chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *TeacherHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminQuestionRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.CreateQuestion(r.Context(), h.publisherID(r), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"question": item})
}
func (h *TeacherHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminQuestionRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	item, err := h.service.UpdateQuestion(r.Context(), h.publisherID(r), chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"question": item})
}
func (h *TeacherHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if writeAdminError(h.logger, w, r, h.service.DeleteQuestion(r.Context(), h.publisherID(r), chi.URLParam(r, "id"))) {
		return
	}
	w.WriteHeader(204)
}
func (h *TeacherHandler) BulkDeleteQuestions(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDs       []string `json:"ids"`
		PackageID string   `json:"package_id"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	deleted, err := h.service.BulkDeleteQuestions(r.Context(), h.publisherID(r), input.IDs, input.PackageID)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"deleted": deleted})
}
