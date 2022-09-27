package probes

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetMonitorByTeamID(ctx context.Context, monitorID int) (*entities.ProbeResult, error)
	Create(ctx context.Context, monitor *entities.ProbeResult) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) GetMonitorByTeamID(ctx context.Context, monitorID int) (*entities.ProbeResult, error) {
	return s.repository.Get(ctx, monitorID)
}

func (s *ServiceImpl) Create(ctx context.Context, m *entities.ProbeResult) error {
	return s.repository.Create(ctx, m)
}
