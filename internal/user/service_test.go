package user_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	emailMocks "github.com/opsway-io/backend/internal/notification/email/mocks"
	eventMocks "github.com/opsway-io/backend/internal/event/mocks"
	storageMocks "github.com/opsway-io/backend/internal/storage/mocks"
	"github.com/opsway-io/backend/internal/user"
	userMocks "github.com/opsway-io/backend/internal/user/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates a user and sends welcome email", func(t *testing.T) {
		repo := new(userMocks.Repository)
		cache := new(userMocks.Cache)
		storage := new(storageMocks.Service)
		emailSender := new(emailMocks.Sender)
		eventService := new(eventMocks.Service)

		svc := user.NewService(repo, cache, storage, emailSender, eventService, user.Config{})

		u := &entities.User{
			Name:  "Test User",
			Email: "test@opsway.io",
		}

		repo.On("Create", ctx, u).Return(nil)
		emailSender.On("Send", ctx, u.Name, u.Email, mock.AnythingOfType("*templates.NewUserWelcomeTemplate")).Return(nil)
		eventService.On("Publish", mock.Anything).Return(nil)

		err := svc.Create(ctx, u)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		emailSender.AssertExpectations(t)
		eventService.AssertExpectations(t)
	})
}

func TestService_GetUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully gets user by ID", func(t *testing.T) {
		repo := new(userMocks.Repository)
		cache := new(userMocks.Cache)

		svc := user.NewService(repo, cache, nil, nil, nil, user.Config{})

		u := &entities.User{
			ID:    1,
			Name:  "Test User",
			Email: "test@opsway.io",
		}
		
		cache.On("GetUser", ctx, uint(1)).Return(nil, nil)
		cache.On("SetUser", ctx, mock.Anything, mock.Anything).Return(nil)

		repo.On("GetUserByID", ctx, uint(1)).Return(u, nil)

		res, err := svc.GetUserByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, u.ID, res.ID)
		repo.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully updates user and invalidates cache", func(t *testing.T) {
		repo := new(userMocks.Repository)
		cache := new(userMocks.Cache)

		svc := user.NewService(repo, cache, nil, nil, nil, user.Config{})

		u := &entities.User{
			ID:    1,
			Name:  "Test User",
			Email: "test@opsway.io",
		}

		repo.On("Update", ctx, u).Return(nil)
		cache.On("DeleteUser", ctx, u.ID).Return(nil)
		cache.On("DeleteUserByEmail", ctx, u.Email).Return(nil)

		err := svc.Update(ctx, u)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}
