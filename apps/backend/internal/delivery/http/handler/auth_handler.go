package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

const maxRequestBody = 1 << 20
const maxQuestionRequestBody = 16 << 20

type AuthHandler struct {
	auth   domain.AuthService
	logger *slog.Logger
}

func NewAuthHandler(auth domain.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.RegisterRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	response, err := h.auth.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrInvalidReferralCode):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			h.logger.ErrorContext(r.Context(), "register user", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	response, err := h.auth.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		h.logger.ErrorContext(r.Context(), "login user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input domain.RefreshRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	response, err := h.auth.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrSessionInvalid) {
			writeError(w, http.StatusUnauthorized, "Sesi berakhir. Silakan masuk kembali.")
			return
		}
		h.logger.ErrorContext(r.Context(), "refresh session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input domain.ForgotPasswordRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	response, err := h.auth.ForgotPassword(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "alamat email tidak valid")
			return
		}
		h.logger.ErrorContext(r.Context(), "forgot password", "error", err)
		writeError(w, http.StatusInternalServerError, "permintaan reset gagal diproses")
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input domain.ResetPasswordRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := h.auth.ResetPassword(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidResetToken):
			writeError(w, http.StatusUnprocessableEntity, domain.ErrInvalidResetToken.Error())
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusUnprocessableEntity, "kata sandi harus terdiri dari 8–72 karakter")
		default:
			h.logger.ErrorContext(r.Context(), "reset password", "error", err)
			writeError(w, http.StatusInternalServerError, "reset kata sandi gagal diproses")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Kata sandi berhasil diperbarui. Silakan masuk kembali."})
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var input domain.GoogleAuthRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	response, err := h.auth.LoginWithGoogle(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGoogleAuthUnavailable):
			writeError(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid google credential")
		default:
			h.logger.ErrorContext(r.Context(), "google login", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	var input domain.Verify2FARequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	response, err := h.auth.Verify2FA(r.Context(), input.MFAToken, input.Code)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSessionInvalid):
			writeError(w, http.StatusUnauthorized, "Sesi dua langkah sudah kedaluwarsa. Silakan masuk kembali.")
		case errors.Is(err, domain.Err2FANotEnabled):
			writeError(w, http.StatusConflict, domain.Err2FANotEnabled.Error())
		case errors.Is(err, domain.ErrInvalid2FACode):
			writeError(w, http.StatusUnprocessableEntity, domain.ErrInvalid2FACode.Error())
		default:
			h.logger.ErrorContext(r.Context(), "verify 2fa", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) TwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	response, err := h.auth.Setup2FA(r.Context(), claims)
	if err != nil {
		switch {
		case errors.Is(err, domain.Err2FAUnsupportedRole):
			writeError(w, http.StatusForbidden, domain.Err2FAUnsupportedRole.Error())
		default:
			h.logger.ErrorContext(r.Context(), "setup 2fa", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) TwoFactorEnable(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input domain.TwoFactorCodeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := h.auth.Enable2FA(r.Context(), claims, input.Code); err != nil {
		switch {
		case errors.Is(err, domain.Err2FAUnsupportedRole):
			writeError(w, http.StatusForbidden, domain.Err2FAUnsupportedRole.Error())
		case errors.Is(err, domain.Err2FAAlreadyEnabled):
			writeError(w, http.StatusConflict, domain.Err2FAAlreadyEnabled.Error())
		case errors.Is(err, domain.Err2FANotEnabled):
			writeError(w, http.StatusUnprocessableEntity, "Lakukan pemindaian QR terlebih dahulu.")
		case errors.Is(err, domain.ErrInvalid2FACode):
			writeError(w, http.StatusUnprocessableEntity, domain.ErrInvalid2FACode.Error())
		default:
			h.logger.ErrorContext(r.Context(), "enable 2fa", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"totp_enabled": true})
}

func (h *AuthHandler) TwoFactorDisable(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input domain.TwoFactorCodeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := h.auth.Disable2FA(r.Context(), claims, input.Code); err != nil {
		switch {
		case errors.Is(err, domain.Err2FAUnsupportedRole):
			writeError(w, http.StatusForbidden, domain.Err2FAUnsupportedRole.Error())
		case errors.Is(err, domain.Err2FANotEnabled):
			writeError(w, http.StatusConflict, domain.Err2FANotEnabled.Error())
		case errors.Is(err, domain.ErrInvalid2FACode):
			writeError(w, http.StatusUnprocessableEntity, domain.ErrInvalid2FACode.Error())
		default:
			h.logger.ErrorContext(r.Context(), "disable 2fa", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"totp_enabled": false})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	if err := h.auth.Logout(r.Context(), claims); err != nil {
		if errors.Is(err, domain.ErrSessionInvalid) {
			writeError(w, http.StatusUnauthorized, "session invalid")
			return
		}
		h.logger.ErrorContext(r.Context(), "logout user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "session invalid")
		return
	}
	user, err := h.auth.CurrentUser(r.Context(), claims)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get current user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input domain.UpdateProfileRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	user, err := h.auth.UpdateProfile(r.Context(), claims, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusBadRequest, "password lama salah")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	limit := int64(maxRequestBody)
	if strings.Contains(r.URL.Path, "/questions") || strings.HasSuffix(r.URL.Path, "/packages/bundle") || strings.HasSuffix(r.URL.Path, "/appeal") {
		limit = maxQuestionRequestBody
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"code":    http.StatusText(status),
		"message": message,
		"error":   message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
