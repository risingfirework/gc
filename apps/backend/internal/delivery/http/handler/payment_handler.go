package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

type PaymentHandler struct {
	service domain.PaymentService
	logger  *slog.Logger
}

func NewPaymentHandler(service domain.PaymentService, logger *slog.Logger) *PaymentHandler {
	return &PaymentHandler{service: service, logger: logger}
}

func (h *PaymentHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	items, err := h.service.ListPackages(r.Context(), page, perPage)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list packages", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "page": max(page, 1), "per_page": normalizedPerPage(perPage)})
}

func (h *PaymentHandler) TrackPackageView(w http.ResponseWriter, r *http.Request) {
	var input domain.PackageViewRequest
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	count, err := h.service.TrackPackageView(r.Context(), chi.URLParam(r, "id"), input.VisitorKey)
	if err != nil {
		if errors.Is(err, domain.ErrPackageNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"view_count": count})
}

func (h *PaymentHandler) ListMyPackages(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	items, err := h.service.ListMyPackages(r.Context(), claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNoPurchasedPackage):
			writeJSON(w, http.StatusOK, map[string]any{"data": []domain.OwnedPackage{}})
		case errors.Is(err, domain.ErrInvalidPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "list my packages", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	if items == nil {
		items = []domain.OwnedPackage{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *PaymentHandler) PricingPolicy(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	policy, err := h.service.GetPricingPolicy(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPayment) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		h.logger.ErrorContext(r.Context(), "get pricing policy", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policy": policy})
}

func (h *PaymentHandler) ClaimFreePackage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	packageID, err := url.PathUnescape(chi.URLParam(r, "id"))
	if err != nil || packageID == "" {
		writeError(w, http.StatusBadRequest, "invalid package id")
		return
	}
	item, err := h.service.ClaimFreePackage(r.Context(), claims.UserID, packageID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPackageNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrNotFreePackage):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrPackageAlreadyOwned):
			writeError(w, http.StatusConflict, "Paket ini sudah kamu miliki dan tidak dapat dipesan kembali.")
		case errors.Is(err, domain.ErrInvalidPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "claim free package", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"package": item})
}

func (h *PaymentHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	var input domain.CheckoutRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	response, err := h.service.Checkout(r.Context(), claims.UserID, r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPackageNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrPackageAlreadyOwned):
			writeError(w, http.StatusConflict, "Paket ini sudah kamu miliki dan tidak dapat dipesan kembali.")
		case errors.Is(err, domain.ErrInvalidPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "checkout package", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	processed, err := h.service.HandleWebhook(r.Context(), rawBody, r.Header.Get("X-Payment-Timestamp"), r.Header.Get("X-Payment-Signature"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidWebhookSignature):
			writeError(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, domain.ErrInvalidPayment), errors.Is(err, domain.ErrInvalidPaymentTransition):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrTransactionNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			h.logger.ErrorContext(r.Context(), "payment webhook", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"received": true, "processed": processed})
}

func normalizedPerPage(value int) int {
	if value < 1 {
		return 20
	}
	if value > 100 {
		return 100
	}
	return value
}
