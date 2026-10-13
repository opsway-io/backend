package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/redis/go-redis/v9"
)

var ErrNoSuchPasswordResetToken = errors.New("no such password reset token")

type Cache interface {
	SetPasswordResetToken(ctx context.Context, userID uint, token string, ttl time.Duration) (err error)
	VerifyAndDeletePasswordResetToken(ctx context.Context, token string) (userID uint, err error)
	GetUser(ctx context.Context, userID uint) (*entities.User, error)
	SetUser(ctx context.Context, user *entities.User, ttl time.Duration) error
	DeleteUser(ctx context.Context, userID uint) error
	GetUserByEmail(ctx context.Context, email string) (*entities.User, error)
	SetUserByEmail(ctx context.Context, user *entities.User, ttl time.Duration) error
	DeleteUserByEmail(ctx context.Context, email string) error
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

func userCacheKey(userID uint) string {
	return fmt.Sprintf("user:%d", userID)
}

func userEmailCacheKey(email string) string {
	return "user:email:" + email
}

func (c *CacheImpl) GetUser(ctx context.Context, userID uint) (*entities.User, error) {
	key := userCacheKey(userID)
	data, err := c.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var user entities.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *CacheImpl) SetUser(ctx context.Context, user *entities.User, ttl time.Duration) error {
	key := userCacheKey(user.ID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.cli.Set(ctx, key, data, ttl).Err()
}

func (c *CacheImpl) DeleteUser(ctx context.Context, userID uint) error {
	return c.cli.Del(ctx, userCacheKey(userID)).Err()
}

func (c *CacheImpl) GetUserByEmail(ctx context.Context, email string) (*entities.User, error) {
	key := userEmailCacheKey(email)
	data, err := c.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var user entities.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *CacheImpl) SetUserByEmail(ctx context.Context, user *entities.User, ttl time.Duration) error {
	key := userEmailCacheKey(user.Email)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.cli.Set(ctx, key, data, ttl).Err()
}

func (c *CacheImpl) DeleteUserByEmail(ctx context.Context, email string) error {
	return c.cli.Del(ctx, userEmailCacheKey(email)).Err()
}
