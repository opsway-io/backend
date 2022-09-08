package maintenance

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	Create(ctx context.Context, maintenance *entities.Maintenance) error
	Update(ctx context.Context, maintenance *entities.Maintenance) error
	Delete(ctx context.Context, maintenance *entities.Maintenance) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Maintenance, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) Create(ctx context.Context, maintenance *entities.Maintenance) error {
	return s.repository.Create(ctx, maintenance)
}

func (s *ServiceImpl) Update(ctx context.Context, maintenance *entities.Maintenance) error {
	return s.repository.Update(ctx, maintenance)
}

func (s *ServiceImpl) Delete(ctx context.Context, maintenance *entities.Maintenance) error {
	return s.repository.Delete(ctx, maintenance)
}
