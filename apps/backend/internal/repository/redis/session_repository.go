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
const defaultDevicePrefix = "tka:auth:device:"

type SessionRepository struct {
	client goredis.UniversalClient
	prefix string
	devicePrefix string
}

func NewSessionRepository(client goredis.UniversalClient) *SessionRepository {
	return &SessionRepository{client: client, prefix: defaultSessionPrefix, devicePrefix: defaultDevicePrefix}
}

func (r *SessionRepository) SetSession(ctx context.Context, userID, jti, deviceID string, ttl time.Duration) error {
	pipe := r.client.Pipeline()
	pipe.Set(ctx, r.key(userID), jti, ttl)
	pipe.Set(ctx, r.deviceKey(userID), deviceID, ttl)
	_, err := pipe.Exec(ctx)
	return err
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

func (r *SessionRepository) GetSessionDevice(ctx context.Context, userID string) (string, error) {
	deviceID, err := r.client.Get(ctx, r.deviceKey(userID)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return deviceID, nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, userID string) error {
	return r.client.Del(ctx, r.key(userID), r.deviceKey(userID)).Err()
}

func (r *SessionRepository) key(userID string) string {
	return r.prefix + userID
}
func (r *SessionRepository) deviceKey(userID string) string {
	return r.devicePrefix + userID
}

// MarkTOTPUsed menyimpan penanda kode TOTP yang sudah dipakai. Nilai balik
// false berarti kode pada langkah waktu tersebut sudah pernah dipakai.
func (r *SessionRepository) MarkTOTPUsed(ctx context.Context, userID string, step int64, ttl time.Duration) (bool, error) {
	key := "tka:auth:totp:" + userID + ":" + strconv.FormatInt(step, 10)
	return r.client.SetNX(ctx, key, "1", ttl).Result()
}

var _ domain.SessionRepository = (*SessionRepository)(nil)
