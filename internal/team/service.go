package team

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByID(ctx context.Context, teamId uint) (*entities.Team, error)
	GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error)
	Create(ctx context.Context, team *entities.Team) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error
	Delete(ctx context.Context, id uint) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) Create(ctx context.Context, team *entities.Team) error {
	return s.repository.Create(ctx, team)
}

func (s *ServiceImpl) UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error {
	return s.repository.UpdateDisplayName(ctx, teamID, displayName)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetUsersByID(ctx context.Context, id uint, offset *int, limit *int, query *string) (*[]TeamUser, error) {
	return s.repository.GetUsersByID(ctx, id, offset, limit, query)
}

func (s *ServiceImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	return s.repository.GetUserRole(ctx, teamID, userID)
}
