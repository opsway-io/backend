package check

import (
	"context"

	"gorm.io/gorm"
)

type Service interface {
	GetMonitorChecksByID(ctx context.Context, monitorID uint) (*[]Check, error)
	GetMonitorMetricsByID(ctx context.Context, monitorID uint) (*[]AggMetric, error)
	Create(ctx context.Context, monitor *Check) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) GetMonitorChecksByID(ctx context.Context, monitorID uint) (*[]Check, error) {
	return s.repository.Get(ctx, monitorID)
}

func (s *ServiceImpl) GetMonitorMetricsByID(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
	return s.repository.GetAggMetrics(ctx, monitorID)
}

func (s *ServiceImpl) Create(ctx context.Context, m *Check) error {
	return s.repository.Create(ctx, m)
}
