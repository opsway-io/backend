package authentication

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	bloomFilterSize = 1000000
	bloomFilterFPR  = 0.0001
)

type Repository interface {
	UseRefreshToken(ctx context.Context, token string) (ok bool, err error)
}

type RepositoryImpl struct {
	redis *redis.Client
}

func NewRepository(redis *redis.Client) Repository {
	redis.Do(
		context.Background(),
		"BF.RESERVE",
		"refresh_tokens",
		bloomFilterFPR,
		bloomFilterSize,
	)

	return &RepositoryImpl{
		redis: redis,
	}
}

func (r *RepositoryImpl) UseRefreshToken(ctx context.Context, token string) (ok bool, err error) {
	res, err := r.redis.Do(ctx, "BF.ADD", "refresh_tokens", token).Result()
	if err == nil {
		// RESP3 replies with a boolean, RESP2 with an integer
		switch notSeenBefore := res.(type) {
		case bool:
			return notSeenBefore, nil
		case int64:
			return notSeenBefore == 1, nil
		default:
			return false, errors.Errorf("unexpected BF.ADD reply type %T", res)
		}
	}

	// If BF.ADD is not supported by Redis (missing RedisBloom module), fallback to SetNX with 90 days TTL
	if strings.Contains(err.Error(), "unknown command") {
		return r.redis.SetNX(ctx, "refresh_tokens:"+token, "1", 90*24*time.Hour).Result()
	}

	return false, errors.Wrap(err, "failed to check refresh token")
}
