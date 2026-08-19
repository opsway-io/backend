package team_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	emailMocks "github.com/opsway-io/backend/internal/notification/email/mocks"
	storageMocks "github.com/opsway-io/backend/internal/storage/mocks"
	"github.com/opsway-io/backend/internal/team"
	teamMocks "github.com/opsway-io/backend/internal/team/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully gets team by ID", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		storage := new(storageMocks.Service)
		emailSender := new(emailMocks.Sender)
		cache := new(teamMocks.Cache)

		cache.On("GetTeam", ctx, uint(1)).Return(nil, nil)
		cache.On("SetTeam", ctx, mock.Anything, mock.Anything).Return(nil)

		svc := team.NewService(team.Config{}, repo, storage, emailSender, cache)

		teamEntity := &entities.Team{
			ID:   1,
			Name: "Test Team",
		}

		repo.On("GetByID", ctx, uint(1)).Return(teamEntity, nil)

		res, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, teamEntity.ID, res.ID)
		repo.AssertExpectations(t)
	})
}

func TestService_CreateWithOwnerUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates a team with owner", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)

		svc := team.NewService(team.Config{}, repo, nil, nil, cache)

		teamEntity := &entities.Team{
			Name: "Test Team",
		}

		ownerID := uint(1)

		repo.On("CreateWithOwnerUserID", ctx, teamEntity, ownerID).Return(nil)

		err := svc.CreateWithOwnerUserID(ctx, teamEntity, ownerID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestService_UpdateDisplayName(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully updates display name and invalidates cache", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)

		svc := team.NewService(team.Config{}, repo, nil, nil, cache)

		teamID := uint(1)
		newName := "New Team Name"

		repo.On("UpdateDisplayName", ctx, teamID, newName).Return(nil)
		cache.On("DeleteTeam", ctx, teamID).Return(nil)

		err := svc.UpdateDisplayName(ctx, teamID, newName)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}
