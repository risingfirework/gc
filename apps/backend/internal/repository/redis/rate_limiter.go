package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"tka/apps/backend/internal/domain"
)

const defaultRateLimitPrefix = "tka:ratelimit:"

// RateLimiter menegakkan batas pemakaian fixed-window di Redis menggunakan
// INCR + EXPIRE atomik. Kunci identik dalam satu jendela berbagi counter.
type RateLimiter struct {
	client goredis.UniversalClient
	prefix string
}

func NewRateLimiter(client goredis.UniversalClient) *RateLimiter {
	return &RateLimiter{client: client, prefix: defaultRateLimitPrefix}
}

// Allow mengembalikan true bila pemakaian masih di bawah limit untuk jendela
// window. Jika Redis tidak tersedia, request dilewatkan (fail-open) agar
// layanan autentikasi tidak terblokir total karena infrastruktur sekunder.
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if key == "" {
		return true, nil
	}
	redisKey := r.prefix + key
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, nil
	}
	count := incr.Val()
	return count <= int64(limit), nil
}

var _ domain.RateLimiter = (*RateLimiter)(nil)
