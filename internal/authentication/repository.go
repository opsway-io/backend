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
	used, err := r.redis.Do(ctx, "BF.ADD", "refresh_tokens", token).Bool()
	if err != nil {
		return false, errors.Wrap(err, "failed to check refresh token")
	}

	if used {
		return false, nil
	}

	return true, nil
}
