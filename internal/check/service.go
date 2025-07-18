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
	GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*MonitorStats, error)
	GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error)

	GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error)
	GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]MonitorUptime, error)
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

func (s *ServiceImpl) GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*MonitorStats, error) {
	return s.repository.GetMonitorStatsByMonitorID(ctx, monitorID)
}

func (s *ServiceImpl) GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
	return s.repository.GetMonitorMetricsByMonitorID(ctx, monitorID)
}

func (s *ServiceImpl) GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error) {
	overviews, err := s.repository.GetMonitorOverviewsByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	stats, err := s.repository.GetMonitorOverviewStatsByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	statsMap := make(map[uint][]float64)
	for _, stat := range *stats {
		statsMap[stat.MonitorID] = stat.Stats
	}
	for i, overview := range *overviews {
		(*overviews)[i].Stats = statsMap[overview.MonitorID]
	}

	return overviews, err
}

func (s *ServiceImpl) GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error) {
	return s.repository.GetMonitorIDAndAssertions(ctx, monitorID, assertions)
}
func (s *ServiceImpl) GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]MonitorUptime, error) {
	return s.repository.GetByTeamIDMonitorsUptime(ctx, teamID, start, end)
}
