package monitor

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetMonitors(ctx context.Context) (*[]entities.Monitor, error)
	GetMonitorByIDAndTeamID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error)
	GetMonitorAndSettingsByID(ctx context.Context, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error)
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

func (s *ServiceImpl) GetMonitors(ctx context.Context) (*[]entities.Monitor, error) {
	return s.repository.GetMonitors(ctx)
}

func (s *ServiceImpl) GetMonitorByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error) {
	return s.repository.GetMonitorByTeamID(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) GetMonitorByIDAndTeamID(ctx context.Context, monitorID uint, teamID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorByIDAndTeamID(ctx, monitorID, teamID)
}

func (s *ServiceImpl) GetMonitorAndSettingsByID(ctx context.Context, monitorID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorAndSettingsByID(ctx, monitorID)
}

func (s *ServiceImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error) {
	return s.repository.GetMonitorsAndSettingsByTeamID(ctx, teamID, offset, limit)
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
