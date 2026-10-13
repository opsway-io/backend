package team_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/team"
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

	c := team.NewCache(cli)

	t.Run("Get and Set Team", func(t *testing.T) {
		teamEntity := &entities.Team{
			ID:   1,
			Name: "Cache Team",
		}

		err := c.SetTeam(ctx, teamEntity, 1*time.Minute)
		assert.NoError(t, err)

		cachedTeam, err := c.GetTeam(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, teamEntity.Name, cachedTeam.Name)

		err = c.DeleteTeam(ctx, 1)
		assert.NoError(t, err)

		_, err = c.GetTeam(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("Get and Set TeamUserCount", func(t *testing.T) {
		err := c.SetTeamUserCount(ctx, 1, 42, 1*time.Minute)
		assert.NoError(t, err)

		count, err := c.GetTeamUserCount(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), count)

		err = c.DeleteTeamUserCount(ctx, 1)
		assert.NoError(t, err)

		_, err = c.GetTeamUserCount(ctx, 1)
		assert.Error(t, err)
	})
}
