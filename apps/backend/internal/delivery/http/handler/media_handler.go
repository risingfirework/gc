package handler

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"tka/apps/backend/internal/delivery/http/middleware"
	"tka/apps/backend/internal/domain"
)

type MediaHandler struct {
	service        domain.MediaService
	logger         *slog.Logger
	maxUploadBytes int64
}

func NewMediaHandler(service domain.MediaService, logger *slog.Logger, maxUploadBytes int64) *MediaHandler {
	if maxUploadBytes <= 0 {
		maxUploadBytes = 100 << 20
	}
	return &MediaHandler{service: service, logger: logger, maxUploadBytes: maxUploadBytes}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	// Batas ini adalah batas atas seluruh jenis media; batas per jenis
	// ditegakkan lagi oleh service setelah jenis file diketahui.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "ukuran media melebihi batas")
			return
		}
		writeError(w, http.StatusBadRequest, "form upload tidak valid")
		return
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file media wajib diunggah")
		return
	}
	defer func() { _ = file.Close() }()

	asset, err := h.service.Upload(r.Context(), domain.MediaUpload{
		OwnerID:      claims.UserID,
		DeclaredKind: r.FormValue("kind"),
		DeclaredMime: header.Header.Get("Content-Type"),
		OriginalName: header.Filename,
		Size:         header.Size,
		Reader:       file,
	})
	if writeMediaError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"asset": asset})
}

func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	media, err := h.service.Open(r.Context(), chi.URLParam(r, "id"))
	if writeMediaError(h.logger, w, r, err) {
		return
	}
	file, err := os.Open(media.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "media tidak ditemukan")
			return
		}
		h.logger.Error("open media file", "error", err)
		writeError(w, http.StatusInternalServerError, "media tidak dapat dibuka")
		return
	}
	defer func() { _ = file.Close() }()

	w.Header().Set("Content-Type", media.Asset.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if disposition := mime.FormatMediaType("inline", map[string]string{"filename": media.Asset.OriginalName}); disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	http.ServeContent(w, r, media.Asset.OriginalName, media.Asset.CreatedAt, file)
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "id"), claims.UserID, claims.Role); writeMediaError(h.logger, w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func writeMediaError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrMediaNotFound):
		writeError(w, http.StatusNotFound, "media tidak ditemukan")
	case errors.Is(err, domain.ErrMediaForbidden):
		writeError(w, http.StatusForbidden, "tidak berhak menghapus media ini")
	case errors.Is(err, domain.ErrMediaTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran media melebihi batas")
	case errors.Is(err, domain.ErrMediaQuota):
		writeError(w, http.StatusTooManyRequests, "kuota unggah harian habis. Coba lagi besok.")
	case errors.Is(err, domain.ErrMediaUnsupported):
		writeError(w, http.StatusUnsupportedMediaType, "jenis media tidak didukung")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "data media tidak valid")
	default:
		logger.Error("media request failed", "error", err, "path", r.URL.Path)
		writeError(w, http.StatusInternalServerError, "media gagal diproses")
	}
	return true
}
