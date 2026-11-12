package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"tka/apps/backend/internal/domain"
)

const mediaPublicPrefix = "/api/v1/media/"

var (
	mediaImageMimes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	mediaAudioMimes = map[string]bool{
		"audio/aac":     true,
		"audio/flac":    true,
		"audio/mp4":     true,
		"audio/mpeg":    true,
		"audio/ogg":     true,
		"audio/wav":     true,
		"audio/webm":    true,
		"audio/x-m4a":   true,
		"audio/x-wav":   true,
	}
	mediaVideoMimes = map[string]bool{
		"video/mp4":       true,
		"video/ogg":       true,
		"video/quicktime": true,
		"video/webm":      true,
		"video/x-matroska": true,
	}
)

type MediaService struct {
	repository domain.MediaRepository
	dir        string
	maxImage   int64
	maxAudio   int64
	maxVideo   int64
}

func NewMediaService(repository domain.MediaRepository, dir string, maxImage, maxAudio, maxVideo int64) *MediaService {
	if strings.TrimSpace(dir) == "" {
		dir = "data/media"
	}
	return &MediaService{
		repository: repository,
		dir:        dir,
		maxImage:   maxImage,
		maxAudio:   maxAudio,
		maxVideo:   maxVideo,
	}
}

func (s *MediaService) Upload(ctx context.Context, input domain.MediaUpload) (*domain.MediaAsset, error) {
	if strings.TrimSpace(input.OwnerID) == "" {
		return nil, domain.ErrInvalidInput
	}
	kind := normalizeMediaKind(input.DeclaredKind)
	if kind == "" {
		kind = mediaKindFromMime(input.DeclaredMime)
	}
	limit, ok := s.limitFor(kind)
	if !ok {
		return nil, domain.ErrMediaUnsupported
	}
	if input.Size > 0 && input.Size > limit {
		return nil, domain.ErrMediaTooLarge
	}

	reader := bufio.NewReader(input.Reader)
	peek, _ := reader.Peek(512)
	if len(peek) == 0 {
		return nil, domain.ErrInvalidInput
	}
	mimeType, ok := resolveMediaMime(kind, http.DetectContentType(peek), input.DeclaredMime, input.OriginalName)
	if !ok {
		return nil, domain.ErrMediaUnsupported
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("create media directory: %w", err)
	}
	id := uuid.NewString()
	storageKey := filepath.ToSlash(filepath.Join(id[0:2], id[2:4], id))
	fullPath := s.fullPath(storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, fmt.Errorf("create media subdirectory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(fullPath), ".upload-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary media file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	written, copyErr := io.Copy(tmp, io.LimitReader(reader, limit+1))
	if copyErr != nil {
		cleanup()
		return nil, fmt.Errorf("write media file: %w", copyErr)
	}
	if written > limit {
		cleanup()
		return nil, domain.ErrMediaTooLarge
	}
	if written == 0 {
		cleanup()
		return nil, domain.ErrInvalidInput
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("close media file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("set media file mode: %w", err)
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("store media file: %w", err)
	}

	asset, err := s.repository.Create(ctx, domain.MediaAsset{
		OwnerID:      input.OwnerID,
		Kind:         kind,
		MimeType:     mimeType,
		SizeBytes:    written,
		OriginalName: sanitizeMediaName(input.OriginalName),
		StorageKey:   storageKey,
	})
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}
	asset.URL = mediaPublicURL(asset.ID)
	return asset, nil
}

func (s *MediaService) Open(ctx context.Context, id string) (*domain.MediaFile, error) {
	if !validUUID(id) {
		return nil, domain.ErrMediaNotFound
	}
	asset, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	path := s.fullPath(asset.StorageKey)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, domain.ErrMediaNotFound
	}
	asset.URL = mediaPublicURL(asset.ID)
	return &domain.MediaFile{Asset: *asset, Path: path}, nil
}

func (s *MediaService) Delete(ctx context.Context, id, actorID, actorRole string) error {
	if !validUUID(id) {
		return domain.ErrMediaNotFound
	}
	asset, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if asset.OwnerID != actorID && actorRole != domain.RoleAdmin && actorRole != domain.RoleOwner {
		return domain.ErrMediaForbidden
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	if err := os.Remove(s.fullPath(asset.StorageKey)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove media file: %w", err)
	}
	return nil
}

func (s *MediaService) fullPath(storageKey string) string {
	return filepath.Join(s.dir, filepath.FromSlash(storageKey))
}

func (s *MediaService) limitFor(kind string) (int64, bool) {
	switch kind {
	case domain.MediaKindImage:
		return s.maxImage, true
	case domain.MediaKindAudio:
		return s.maxAudio, true
	case domain.MediaKindVideo:
		return s.maxVideo, true
	default:
		return 0, false
	}
}

func mediaPublicURL(id string) string { return mediaPublicPrefix + id }

func normalizeMediaKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case domain.MediaKindImage, "gambar", "foto":
		return domain.MediaKindImage
	case domain.MediaKindAudio, "suara", "lagu":
		return domain.MediaKindAudio
	case domain.MediaKindVideo, "film":
		return domain.MediaKindVideo
	default:
		return ""
	}
}

func mediaKindFromMime(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	switch {
	case strings.HasPrefix(value, "image/"):
		return domain.MediaKindImage
	case strings.HasPrefix(value, "audio/"):
		return domain.MediaKindAudio
	case strings.HasPrefix(value, "video/"):
		return domain.MediaKindVideo
	default:
		return ""
	}
}

func resolveMediaMime(kind, detected, declared, filename string) (string, bool) {
	candidates := []string{detected, declared}
	if ext := filepath.Ext(filename); ext != "" {
		candidates = append(candidates, mime.TypeByExtension(ext))
	}
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(strings.Split(candidate, ";")[0]))
		if candidate == "application/ogg" {
			switch kind {
			case domain.MediaKindAudio:
				candidate = "audio/ogg"
			case domain.MediaKindVideo:
				candidate = "video/ogg"
			}
		}
		if mediaMimeAllowed(kind, candidate) {
			return candidate, true
		}
	}
	return "", false
}

func mediaMimeAllowed(kind, mimeType string) bool {
	switch kind {
	case domain.MediaKindImage:
		return mediaImageMimes[mimeType]
	case domain.MediaKindAudio:
		return mediaAudioMimes[mimeType]
	case domain.MediaKindVideo:
		return mediaVideoMimes[mimeType]
	default:
		return false
	}
}

func sanitizeMediaName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	if len(value) > 180 {
		value = value[:180]
	}
	return value
}
