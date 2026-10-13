package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/user"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainersredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	redisContainer, err := testcontainersredis.Run(ctx,
		"redis/redis-stack:7.4.0-v3",
	)
	require.NoError(t, err)
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redisclient.ParseURL(connStr)
	require.NoError(t, err)

	cli := redisclient.NewClient(opts)

	c := user.NewCache(cli)

	t.Run("Get and Set User", func(t *testing.T) {
		u := &entities.User{
			ID:    1,
			Name:  "Test User",
			Email: "test@opsway.eu",
		}

		err := c.SetUser(ctx, u, 1*time.Minute)
		assert.NoError(t, err)

		cachedUser, err := c.GetUser(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, u.Email, cachedUser.Email)

		err = c.DeleteUser(ctx, 1)
		assert.NoError(t, err)

		_, err = c.GetUser(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("Get and Set UserByEmail", func(t *testing.T) {
		u := &entities.User{
			ID:    2,
			Name:  "Test User 2",
			Email: "test2@opsway.eu",
		}

		err := c.SetUserByEmail(ctx, u, 1*time.Minute)
		assert.NoError(t, err)

		cachedUser, err := c.GetUserByEmail(ctx, u.Email)
		assert.NoError(t, err)
		assert.Equal(t, u.ID, cachedUser.ID)

		err = c.DeleteUserByEmail(ctx, u.Email)
		assert.NoError(t, err)

		_, err = c.GetUserByEmail(ctx, u.Email)
		assert.Error(t, err)
	})
}
