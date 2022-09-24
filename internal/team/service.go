package team

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Team, error)
	GetUsersByID(ctx context.Context, id uint) (*[]entities.User, error)
	Create(ctx context.Context, team *entities.Team) error
	Update(ctx context.Context, team *entities.Team) error
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

func (s *ServiceImpl) Update(ctx context.Context, team *entities.Team) error {
	return s.repository.Update(ctx, team)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetUsersByID(ctx context.Context, id uint) (*[]entities.User, error) {
	return s.repository.GetUsersByID(ctx, id)
}
