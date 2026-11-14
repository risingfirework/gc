package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"tka/apps/backend/internal/domain"
)

const defaultSessionPrefix = "tka:auth:session:"

type SessionRepository struct {
	client goredis.UniversalClient
	prefix string
}

func NewSessionRepository(client goredis.UniversalClient) *SessionRepository {
	return &SessionRepository{client: client, prefix: defaultSessionPrefix}
}

func (r *SessionRepository) SetSession(ctx context.Context, userID, jti string, ttl time.Duration) error {
	return r.client.Set(ctx, r.key(userID), jti, ttl).Err()
}

func (r *SessionRepository) GetSession(ctx context.Context, userID string) (string, error) {
	jti, err := r.client.Get(ctx, r.key(userID)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrSessionNotFound
	}
	if err != nil {
		return "", err
	}
	return jti, nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, userID string) error {
	return r.client.Del(ctx, r.key(userID)).Err()
}

func (r *SessionRepository) key(userID string) string {
	return r.prefix + userID
}

// MarkTOTPUsed menyimpan penanda kode TOTP yang sudah dipakai. Nilai balik
// false berarti kode pada langkah waktu tersebut sudah pernah dipakai.
func (r *SessionRepository) MarkTOTPUsed(ctx context.Context, userID string, step int64, ttl time.Duration) (bool, error) {
	key := "tka:auth:totp:" + userID + ":" + strconv.FormatInt(step, 10)
	return r.client.SetNX(ctx, key, "1", ttl).Result()
}

var _ domain.SessionRepository = (*SessionRepository)(nil)
