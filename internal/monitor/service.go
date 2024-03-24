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
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
	SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error
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

func (s *ServiceImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
}

func (s *ServiceImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error) {
	return s.repository.GetMonitorsAndSettingsByTeamID(ctx, teamID, offset, limit, query)
}

func (s *ServiceImpl) SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error {
	err := s.repository.SetState(ctx, teamID, monitorID, state)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return err
	}

	if state == entities.MonitorStateInactive {
		return s.schedule.Remove(ctx, monitorID)
	}

	m, err := s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
	if err != nil {
		return err
	}

	return s.schedule.Add(ctx, m)
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
	m.ID = monitorID

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
