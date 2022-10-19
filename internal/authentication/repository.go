package authentication

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	CheckAndDeleteRefreshToken(ctx context.Context, refreshTokenID string, refreshToken string) (ok bool, err error)
	CreateRefreshToken(ctx context.Context, refreshTokenID string, refreshToken string, exp time.Duration) (err error)
}

type RepositoryImpl struct {
	redis *redis.Client
}

func NewRepository(redis *redis.Client) Repository {
	return &RepositoryImpl{
		redis: redis,
	}
}

func (r *RepositoryImpl) CheckAndDeleteRefreshToken(ctx context.Context, refreshTokenID string, refreshToken string) (ok bool, err error) {
	key := r.getRefreshTokenKey(refreshTokenID)

	val, err := r.redis.WithContext(ctx).Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}

		return false, errors.Wrap(err, "failed to get refresh token")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(val), []byte(refreshToken)); err != nil {
		return false, nil
	}

	deleted, err := r.redis.WithContext(ctx).Del(key).Result()
	if err != nil {
		return false, err
	}

	return deleted > 0, nil
}

func (r *RepositoryImpl) CreateRefreshToken(ctx context.Context, refreshTokenID string, refreshToken string, exp time.Duration) (err error) {
	key := r.getRefreshTokenKey(refreshTokenID)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to hash token")
	}

	hashStr := string(hashedPassword)

	return r.redis.WithContext(ctx).Set(key, hashStr, exp).Err()
}

func (r *RepositoryImpl) getRefreshTokenKey(refreshToken string) string {
	return fmt.Sprintf("refresh_token:%s", refreshToken)
}
