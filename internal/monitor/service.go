package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/boomerang"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service interface {
	GetMonitorByIDAndTeamID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
	Create(ctx context.Context, monitor *entities.Monitor) error
	Update(ctx context.Context, teamID, monitorID uint, monitor *entities.Monitor) error
	Delete(ctx context.Context, teamID, monitorID uint) error
}

type ServiceImpl struct {
	repository Repository
	schedule   Schedule
}

func NewService(db *gorm.DB, redisClient *redis.Client) Service {
	return &ServiceImpl{
		repository: NewRepository(db),
		schedule:   NewSchedule(redisClient),
	}
}

func (s *ServiceImpl) GetMonitorByIDAndTeamID(ctx context.Context, monitorID uint, teamID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorByIDAndTeamID(ctx, monitorID, teamID)
}

func (s *ServiceImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
}

func (s *ServiceImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error) {
	return s.repository.GetMonitorsAndSettingsByTeamID(ctx, teamID, offset, limit, query)
}

func (s *ServiceImpl) Create(ctx context.Context, m *entities.Monitor) error {
	m.State = entities.MonitorStateActive

	err := s.repository.Create(ctx, m)
	if err != nil {
		return err
	}

	return s.schedule.Add(ctx, m)
}

func (s *ServiceImpl) Update(ctx context.Context, teamID, monitorID uint, m *entities.Monitor) error {
	err := s.schedule.Remove(ctx, m.ID)
	if err != nil {
		if !errors.Is(err, boomerang.ErrTaskDoesNotExist) {
			return err
		}
	}

	if err = s.repository.Update(
		ctx,
		teamID,
		monitorID,
		m,
	); err != nil {
		return err
	}

	if m.State == entities.MonitorStateInactive {
		return nil
	}

	return s.schedule.Add(ctx, m)
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, monitorID uint) error {
	err := s.repository.Delete(ctx, teamID, monitorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return err
	}

	err = s.schedule.Remove(ctx, monitorID)
	if err != nil {
		if errors.Is(err, boomerang.ErrTaskDoesNotExist) {
			return nil
		}

		return err
	}

	return nil
}
