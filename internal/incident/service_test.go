package incident_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	eventMocks "github.com/opsway-io/backend/internal/event/mocks"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/incident/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_Upsert(t *testing.T) {
	mockRepo := mocks.NewRepository(t)
	mockEventService := eventMocks.NewService(t)
	svc := incident.NewService(mockRepo, mockEventService)

	t.Run("upserts and publishes events", func(t *testing.T) {
		ctx := context.Background()
		incidents := []entities.Incident{{ID: 1, TeamID: 1}}

		mockRepo.On("Upsert", ctx, &incidents).Return(nil).Once()
		mockEventService.On("Publish", mock.AnythingOfType("events.IncidentCreatedEvent")).Return(nil).Once()

		err := svc.Upsert(ctx, &incidents)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	mockRepo := mocks.NewRepository(t)
	mockEventService := eventMocks.NewService(t)
	svc := incident.NewService(mockRepo, mockEventService)

	t.Run("creates and publishes events", func(t *testing.T) {
		ctx := context.Background()
		incidents := []entities.Incident{{ID: 1, TeamID: 1}}

		mockRepo.On("Create", ctx, &incidents).Return(nil).Once()
		mockEventService.On("Publish", mock.AnythingOfType("events.IncidentCreatedEvent")).Return(nil).Once()

		err := svc.Create(ctx, &incidents)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
	})
}

func TestService_GetByID(t *testing.T) {
	mockRepo := mocks.NewRepository(t)
	mockEventService := eventMocks.NewService(t)
	svc := incident.NewService(mockRepo, mockEventService)

	t.Run("returns incident by ID", func(t *testing.T) {
		ctx := context.Background()
		expected := &entities.Incident{ID: 1}

		mockRepo.On("GetByID", ctx, uint(1)).Return(expected, nil).Once()

		actual, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
		mockRepo.AssertExpectations(t)
	})
}
