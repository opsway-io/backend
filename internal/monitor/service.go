package monitor

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx context.Context, monitor *entities.Monitor) error
	Update(ctx context.Context, monitor *entities.Monitor) error
	Delete(ctx context.Context, id int) error
	GetByIDAndTeamID(ctx context.Context, teamID, id int) (*entities.Monitor, error)
	GetByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error)
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) Create(ctx context.Context, m *entities.Monitor) error {
	return s.repository.Create(ctx, m)
}

func (s *ServiceImpl) Update(ctx context.Context, m *entities.Monitor) error {
	return s.repository.Update(ctx, m)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error) {
	return s.repository.GetByTeamID(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) GetByIDAndTeamID(ctx context.Context, id, teamID int) (*entities.Monitor, error) {
	return s.repository.GetByIDAndTeamID(ctx, id, teamID)
}
