// Package handler berisi handler HTTP untuk endpoint publik/admin.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.UpdateSiteSettings(r.Context(), actorID, actorEmail, input)
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
func (h *AdminHandler) FinanceExportCSV(w http.ResponseWriter, r *http.Request) {
	payload, err := h.service.ExportFinanceCSV(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="finance-ledger-`+time.Now().Format("20060102-150405")+`.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
func (h *AdminHandler) UpdateFinanceSettings(w http.ResponseWriter, r *http.Request) {
	var input domain.FinanceSettings
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.UpdateFinanceSettings(r.Context(), actorID, actorEmail, input)
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
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.UpdateUserFinance(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
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
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.PayTeacherCommissions(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input.Reference)
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
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.ReviewPayoutRequest(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout_request": result})
}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var input domain.AdminUpdateUserRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.UpdateUser(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, 422, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, 404, err.Error())
			return
		}
		if errors.Is(err, domain.ErrLastOwnerGuard) {
			writeError(w, 409, err.Error())
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
	actorID, actorEmail := actorFromContext(r)
	item, err := h.service.CreateUser(r.Context(), actorID, actorEmail, input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 201, map[string]any{"user": item})
}
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	actorID, actorEmail := actorFromContext(r)
	err := h.service.DeleteUser(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func pageQuery(r *http.Request) (int, int) {
	query := r.URL.Query()
	return parsePositiveInt(query.Get("page"), 1), parsePositiveInt(query.Get("per_page"), 25)
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	query := r.URL.Query()
	result, err := h.service.ListUsers(r.Context(), page, perPage, query.Get("role"), query.Get("level"), query.Get("q"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) Packages(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	query := r.URL.Query()
	result, err := h.service.ListAdminPackages(r.Context(), page, perPage, query.Get("status"), query.Get("jenjang"), query.Get("q"), query.Get("exam_type"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) Exams(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	result, err := h.service.ListAdminExams(r.Context(), page, perPage, r.URL.Query().Get("package_id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result.Items, "page": result.Page, "count": result.Count, "total": result.Total})
}

func (h *AdminHandler) Questions(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	result, err := h.service.ListAdminQuestions(r.Context(), page, perPage, r.URL.Query().Get("package_id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result.Items, "page": result.Page, "count": result.Count, "total": result.Total})
}

func (h *AdminHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 25)
	result, err := h.service.ListTransactions(r.Context(), page, perPage, query.Get("status"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":        result.Items,
		"page":         result.Page,
		"count":        result.Count,
		"total":        result.Total,
		"transactions": result.Items,
	})
}

func (h *AdminHandler) RefundTransaction(w http.ResponseWriter, r *http.Request) {
	var input domain.RefundTransactionRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := actorFromContext(r)
	result, err := h.service.RefundTransaction(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transaction": result})
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
func (h *AdminHandler) AuditLogList(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAuditLogs(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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
func (h *AdminHandler) CBTSettings(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCBTPublishSettings(r.Context())
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *AdminHandler) SetCBTPublish(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PublishPembahasan bool `json:"publish_pembahasan"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	actorID, actorEmail := actorFromContext(r)
	item, err := h.service.SetExamPublishPembahasan(r.Context(), actorID, actorEmail, chi.URLParam(r, "id"), input.PublishPembahasan)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}
func (h *AdminHandler) CBTParticipants(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCBTParticipants(r.Context(), chi.URLParam(r, "id"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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
func (h *AdminHandler) BulkDeleteQuestions(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDs       []string `json:"ids"`
		PackageID string   `json:"package_id"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid JSON request")
		return
	}
	deleted, err := h.service.BulkDeleteQuestions(r.Context(), input.IDs, input.PackageID)
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, 200, map[string]any{"deleted": deleted})
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

// TeacherVerifications menampilkan daftar guru yang menunggu/telah ditolak
// untuk diproses operator/admin (opsional filter ?status=pending|rejected).
func (h *AdminHandler) TeacherVerifications(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageQuery(r)
	result, err := h.service.ListTeacherVerifications(r.Context(), page, perPage, r.URL.Query().Get("status"))
	if writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ApproveTeacher menyetujui pendaftaran seorang guru.
func (h *AdminHandler) ApproveTeacher(w http.ResponseWriter, r *http.Request) {
	id, email := actorFromContext(r)
	if err := h.service.ApproveTeacher(r.Context(), id, email, chi.URLParam(r, "id")); writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": domain.TeacherVerificationApproved})
}

// RejectTeacher menolak pendaftaran guru dengan alasan yang akan ditampilkan
// pada halaman guru beserta tombol sanggah.
func (h *AdminHandler) RejectTeacher(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	id, email := actorFromContext(r)
	if err := h.service.RejectTeacher(r.Context(), id, email, chi.URLParam(r, "id"), input.Reason); writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": domain.TeacherVerificationRejected})
}

// CheckTeacherSIMPKB memeriksa NIK guru ke portal SIMPKB dan menyimpan
// screenshot hasil pencarian sebagai bukti di panel verifikasi.
func (h *AdminHandler) CheckTeacherSIMPKB(w http.ResponseWriter, r *http.Request) {
	if err := h.service.CheckTeacherSIMPKB(r.Context(), chi.URLParam(r, "id")); writeAdminError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": domain.SIMPKBStatusChecking, "message": "Pengecekan SIMPKB dijalankan di latar belakang."})
}

func writeAdminError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if ok {
		logger.DebugContext(r.Context(), "admin access trace", "route", r.URL.Path, "actor", claims.Email, "role", claims.Role, "error", err.Error())
	}
	switch {
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrPayoutAccountRequired), errors.Is(err, domain.ErrPayoutMinimum), errors.Is(err, domain.ErrInsufficientPayoutBalance), errors.Is(err, domain.ErrPayoutProofRequired), errors.Is(err, domain.ErrInvalidPayment), errors.Is(err, domain.ErrNotFreePackage), errors.Is(err, domain.ErrInvalidReferralCode), errors.Is(err, domain.ErrInvalidWebhookSignature), errors.Is(err, domain.ErrInvalidPaymentTransition):
		writeError(w, 422, err.Error())
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrPackageNotFound), errors.Is(err, domain.ErrExamNotFound), errors.Is(err, domain.ErrQuestionNotFound), errors.Is(err, domain.ErrMasterNotFound), errors.Is(err, domain.ErrPayoutRequestNotFound), errors.Is(err, domain.ErrTransactionNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists), errors.Is(err, domain.ErrAdminConflict), errors.Is(err, domain.ErrDuplicateName), errors.Is(err, domain.ErrDuplicateKode), errors.Is(err, domain.ErrPayoutRequestActive), errors.Is(err, domain.ErrPayoutTransition), errors.Is(err, domain.ErrLastOwnerGuard), errors.Is(err, domain.ErrTransactionNotRefundable), errors.Is(err, domain.ErrPackageHasAttempts), errors.Is(err, domain.ErrPackageInUse), errors.Is(err, domain.ErrExamHasAttempts):
		writeError(w, 409, err.Error())
	case errors.Is(err, domain.ErrExamForbidden):
		writeError(w, 403, err.Error())
	default:
		logger.ErrorContext(r.Context(), "admin mutation", "error", err)
		writeError(w, 500, "internal server error")
	}
	return true
}

func actorFromContext(r *http.Request) (id, email string) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		return "", ""
	}
	return claims.UserID, claims.Email
}
