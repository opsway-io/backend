package monitor_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	monitorMocks "github.com/opsway-io/backend/internal/monitor/mocks"
	"github.com/stretchr/testify/assert"
)

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates a monitor and adds it to schedule", func(t *testing.T) {
		repo := new(monitorMocks.Repository)
		schedule := new(monitorMocks.Schedule)

		svc := monitor.NewServiceWithDeps(repo, schedule)

		m := &entities.Monitor{
			Name: "Test Monitor",
		}

		repo.On("Create", ctx, m).Return(nil)
		schedule.Mock.On("Add", ctx, m).Return(nil)

		err := svc.Create(ctx, m)

		assert.NoError(t, err)
		assert.Equal(t, entities.MonitorStateActive, m.State)
		repo.AssertExpectations(t)
		schedule.Mock.AssertExpectations(t)
	})
}
