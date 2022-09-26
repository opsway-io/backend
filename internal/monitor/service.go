package monitor

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetMonitorByIDAndTeamID(ctx context.Context, teamID, id int) (*entities.Monitor, error)
	GetMonitorByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error)
	Create(ctx context.Context, monitor *entities.Monitor) error
	Update(ctx context.Context, monitor *entities.Monitor) error
	Delete(ctx context.Context, id int) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) GetMonitorByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error) {
	return s.repository.GetMonitorByTeamID(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) GetMonitorByIDAndTeamID(ctx context.Context, id, teamID int) (*entities.Monitor, error) {
	return s.repository.GetMonitorByIDAndTeamID(ctx, id, teamID)
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
