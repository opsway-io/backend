package monitor_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestScheduleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Start Redis Container
	redisC, err := redisContainer.Run(ctx,
		"redis:7-alpine",
	)
	require.NoError(t, err)
	defer func() {
		if err := redisC.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	uri, err := redisC.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redis.ParseURL(uri)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	defer client.Close()

	// 2. Test Schedule
	schedule := monitor.NewSchedule(client)

	m := &entities.Monitor{
		ID:   123,
		Name: "Test Monitor",
		Settings: entities.MonitorSettings{
			Frequency: time.Minute,
			Locations: []string{"eu-central", "us-east"},
		},
	}

	// Add to schedule
	err = schedule.Add(ctx, m)
	assert.NoError(t, err)

	// Remove from schedule
	err = schedule.Remove(ctx, m)
	assert.NoError(t, err)

	// Try removing again to verify no error or handled gracefully
	err = schedule.Remove(ctx, m)
	assert.NoError(t, err)
}
