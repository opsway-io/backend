package maintenance_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	eventMocks "github.com/opsway-io/backend/internal/event/mocks"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/maintenance/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetActive(t *testing.T) {
	mockRepo := mocks.NewRepository(t)
	mockEventService := eventMocks.NewService(t)
	svc := maintenance.NewService(mockRepo, mockEventService)

	t.Run("returns active maintenance windows", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		expected := &[]entities.Maintenance{{ID: 1}}

		mockRepo.On("GetActive", ctx, now).Return(expected, nil).Once()

		actual, err := svc.GetActive(ctx, now)

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	mockRepo := mocks.NewRepository(t)
	mockEventService := eventMocks.NewService(t)
	svc := maintenance.NewService(mockRepo, mockEventService)

	t.Run("creates maintenance window", func(t *testing.T) {
		ctx := context.Background()
		m := &entities.Maintenance{ID: 1}

		mockRepo.On("Create", ctx, m).Return(nil).Once()
		mockEventService.On("Publish", mock.AnythingOfType("events.MaintenanceEvent")).Return(nil).Once()

		err := svc.Create(ctx, m)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
	})
}
