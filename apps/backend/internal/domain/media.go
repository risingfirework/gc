package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrMediaNotFound    = errors.New("media not found")
	ErrMediaForbidden   = errors.New("media access forbidden")
	ErrMediaTooLarge    = errors.New("media file too large")
	ErrMediaUnsupported = errors.New("unsupported media type")
)

const (
	MediaKindImage = "image"
	MediaKindAudio = "audio"
	MediaKindVideo = "video"
)

// MediaAsset adalah metadata file media yang diunggah ke server. URL bersifat
// publik dan tidak dapat ditebak karena memakai UUID.
type MediaAsset struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"owner_id,omitempty"`
	Kind         string    `json:"kind"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	OriginalName string    `json:"original_name,omitempty"`
	URL          string    `json:"url"`
	StorageKey   string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type MediaUpload struct {
	OwnerID      string
	DeclaredKind string
	DeclaredMime string
	OriginalName string
	Size         int64
	Reader       io.Reader
}

type MediaFile struct {
	Asset MediaAsset
	Path  string
}

type MediaRepository interface {
	Create(ctx context.Context, asset MediaAsset) (*MediaAsset, error)
	Get(ctx context.Context, id string) (*MediaAsset, error)
	Delete(ctx context.Context, id string) error
}

type MediaService interface {
	Upload(ctx context.Context, input MediaUpload) (*MediaAsset, error)
	Open(ctx context.Context, id string) (*MediaFile, error)
	Delete(ctx context.Context, id, actorID, actorRole string) error
}
