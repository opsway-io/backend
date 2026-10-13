package authentication

import (
	"context"

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
	if err != nil {
		return false, errors.Wrap(err, "failed to check refresh token")
	}

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
