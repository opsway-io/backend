package probes

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

type Service interface {
	GetMonitorResultsID(ctx context.Context, monitorID uint64) (*entities.HttpResult, error)
	Create(ctx context.Context, monitor *entities.HttpResult) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) GetMonitorResultsID(ctx context.Context, monitorID uint64) (*entities.HttpResult, error) {
	return s.repository.Get(ctx, monitorID)
}

func (s *ServiceImpl) Create(ctx context.Context, m *entities.HttpResult) error {
	return s.repository.Create(ctx, m)
}
