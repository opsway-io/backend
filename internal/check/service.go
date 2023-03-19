package check

import (
	"context"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx context.Context, c *Check) error

	GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]Check, error)
	GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*Check, error)

	GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint) (*[]AggMetric, error)
}

type ServiceImpl struct {
	repository Repository
}

func NewService(db *gorm.DB) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
	}
}

func (s *ServiceImpl) Create(ctx context.Context, c *Check) error {
	return s.repository.Create(ctx, c)
}

func (s *ServiceImpl) GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]Check, error) {
	return s.repository.GetByTeamIDAndMonitorIDPaginated(ctx, teamID, monitorID, offset, limit)
}

func (s *ServiceImpl) GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*Check, error) {
	return s.repository.GetByTeamIDAndMonitorIDAndCheckID(ctx, teamID, monitorID, checkID)
}

func (s *ServiceImpl) GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
	return s.repository.GetMonitorMetricsByMonitorID(ctx, monitorID)
}
