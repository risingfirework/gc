package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

type CBTHandler struct {
	cbt    domain.CBTService
	logger *slog.Logger
}

func NewCBTHandler(cbt domain.CBTService, logger *slog.Logger) *CBTHandler {
	return &CBTHandler{cbt: cbt, logger: logger}
}

func (h *CBTHandler) LookupCBT(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	packages, err := h.cbt.LookupCBT(r.Context(), claims.UserID, r.URL.Query().Get("token"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": packages})
}

func (h *CBTHandler) StartExam(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	response, err := h.cbt.StartExam(r.Context(), claims.UserID, chi.URLParam(r, "id"), r.URL.Query().Get("cbt_token"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *CBTHandler) ListPackageExams(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	exams, err := h.cbt.ListPackageExams(r.Context(), claims.UserID, chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": exams})
}

func (h *CBTHandler) SyncAnswer(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	var input domain.SyncAnswerRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	response, err := h.cbt.SyncAnswer(r.Context(), claims.UserID, input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *CBTHandler) SubmitExam(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	if r.ContentLength != 0 {
		var input domain.SubmitExamRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}
	}
	response, err := h.cbt.SubmitExam(r.Context(), claims.UserID, chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *CBTHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrExamNotFound), errors.Is(err, domain.ErrUserExamNotFound), errors.Is(err, domain.ErrQuestionNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrExamForbidden), errors.Is(err, domain.ErrCBTTokenRequired), errors.Is(err, domain.ErrCBTTokenInvalid):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrInvalidAnswer):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrExamExpired), errors.Is(err, domain.ErrExamAlreadySubmitted), errors.Is(err, domain.ErrExamSubmissionInProgress), errors.Is(err, domain.ErrExamStillRunning):
		writeError(w, http.StatusConflict, err.Error())
	default:
		h.logger.ErrorContext(r.Context(), "CBT request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
