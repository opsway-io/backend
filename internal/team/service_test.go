package team_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestService_AcceptInviteByToken(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret"
	cfg := team.Config{
		InvitationSecret: secret,
		InvitationExpiry: time.Hour,
	}

	createToken := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		str, err := tok.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		return str
	}

	t.Run("successfully accepts invite with exact matching email", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		tokenStr := createToken(jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "team-invite",
			"sub":     "user@example.com",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleAdmin),
		})

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		invitation := &entities.TeamInvitation{
			TeamID: 1,
			Email:  "user@example.com",
			Token:  tokenStr,
		}

		repo.On("GetTeamInvitationByToken", ctx, tokenStr).Return(invitation, nil)
		repo.On("AddUser", ctx, uint(1), uint(42), entities.TeamRoleAdmin).Return(nil)
		cache.On("DeleteTeamUserCount", ctx, uint(1)).Return(nil)
		repo.On("DeleteTeamInvitation", ctx, uint(1), "user@example.com").Return(nil)

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.NoError(t, err)

		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("successfully accepts invite with case-insensitive email match", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		tokenStr := createToken(jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "team-invite",
			"sub":     "User@Example.COM ",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleMember),
		})

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		invitation := &entities.TeamInvitation{
			TeamID: 1,
			Email:  "User@Example.COM ",
			Token:  tokenStr,
		}

		repo.On("GetTeamInvitationByToken", ctx, tokenStr).Return(invitation, nil)
		repo.On("AddUser", ctx, uint(1), uint(42), entities.TeamRoleMember).Return(nil)
		cache.On("DeleteTeamUserCount", ctx, uint(1)).Return(nil)
		repo.On("DeleteTeamInvitation", ctx, uint(1), "User@Example.COM ").Return(nil)

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.NoError(t, err)

		repo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("returns ErrEmailMismatch when signed-in user email differs from invite email", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		tokenStr := createToken(jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "team-invite",
			"sub":     "other@example.com",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleMember),
		})

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		invitation := &entities.TeamInvitation{
			TeamID: 1,
			Email:  "other@example.com",
			Token:  tokenStr,
		}

		repo.On("GetTeamInvitationByToken", ctx, tokenStr).Return(invitation, nil)

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.ErrorIs(t, err, team.ErrEmailMismatch)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrInvitationNotFound when token not in DB", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		tokenStr := createToken(jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "team-invite",
			"sub":     "user@example.com",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleMember),
		})

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		repo.On("GetTeamInvitationByToken", ctx, tokenStr).Return(nil, assert.AnError)

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.ErrorIs(t, err, team.ErrInvitationNotFound)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrInvalidInvitationToken when signature is invalid", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		wrongKeyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "team-invite",
			"sub":     "user@example.com",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleMember),
		})
		tokenStr, _ := wrongKeyToken.SignedString([]byte("wrong-secret"))

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.ErrorIs(t, err, team.ErrInvalidInvitationToken)
	})

	t.Run("returns ErrInvalidInvitationToken when token type is not team-invite", func(t *testing.T) {
		repo := new(teamMocks.Repository)
		cache := new(teamMocks.Cache)
		svc := team.NewService(cfg, repo, nil, nil, cache)

		tokenStr := createToken(jwt.MapClaims{
			"exp":     time.Now().Add(time.Hour).Unix(),
			"type":    "wrong-type",
			"sub":     "user@example.com",
			"team_id": float64(1),
			"role":    string(entities.TeamRoleMember),
		})

		user := &entities.User{
			ID:    42,
			Email: "user@example.com",
		}

		invitation := &entities.TeamInvitation{
			TeamID: 1,
			Email:  "user@example.com",
			Token:  tokenStr,
		}

		repo.On("GetTeamInvitationByToken", ctx, tokenStr).Return(invitation, nil)

		err := svc.AcceptInviteByToken(ctx, tokenStr, user)
		assert.ErrorIs(t, err, team.ErrInvalidInvitationToken)

		repo.AssertExpectations(t)
	})
}

