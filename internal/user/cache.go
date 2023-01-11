package user

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrNoSuchPasswordResetToken = errors.New("no such password reset token")

type Cache interface {
	SetPasswordResetToken(ctx context.Context, userID uint, token string, ttl time.Duration) (err error)
	VerifyAndDeletePasswordResetToken(ctx context.Context, token string) (userID uint, err error)
}

type CacheImpl struct {
	cli *redis.Client
}

func NewCache(cli *redis.Client) Cache {
	return &CacheImpl{
		cli: cli,
	}
}

func (c *CacheImpl) SetPasswordResetToken(ctx context.Context, userID uint, token string, ttl time.Duration) error {
	return c.cli.Set(ctx, passwordResetTokenKey(token), userID, ttl).Err()
}

func (c *CacheImpl) VerifyAndDeletePasswordResetToken(ctx context.Context, token string) (uint, error) {
	key := passwordResetTokenKey(token)

	// Get the user ID from the token
	userID, err := c.cli.Get(ctx, key).Uint64()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrNoSuchPasswordResetToken
		}

		return 0, err
	}

	// Delete the token
	if err := c.cli.Del(ctx, key).Err(); err != nil {
		return 0, err
	}

	return uint(userID), nil
}

func passwordResetTokenKey(token string) string {
	return "password_reset_token:" + token
}
