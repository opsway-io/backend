package maintenance

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	GetActive(ctx context.Context, now time.Time) (*[]entities.Maintenance, error)
	GetActiveByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error)
	GetUpcomingByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error)
	GetAllByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Maintenance, error)
	Create(ctx context.Context, maintenance *entities.Maintenance) error
	Update(ctx context.Context, maintenance *entities.Maintenance) error
	Delete(ctx context.Context, maintenance *entities.Maintenance) error
}

type ServiceImpl struct {
	repository   Repository
	eventService event.Service
}

func NewService(repository Repository, eventService event.Service) Service {
	return &ServiceImpl{
		repository:   repository,
		eventService: eventService,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Maintenance, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetActive(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	return s.repository.GetActive(ctx, now)
}

func (s *ServiceImpl) GetActiveByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	return s.repository.GetActiveByMonitorIDs(ctx, now, monitorIDs)
}

func (s *ServiceImpl) GetUpcomingByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	return s.repository.GetUpcomingByMonitorIDs(ctx, now, monitorIDs)
}

func (s *ServiceImpl) GetAllByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Maintenance, error) {
	return s.repository.GetAllByMonitorIDs(ctx, monitorIDs)
}

func (s *ServiceImpl) Create(ctx context.Context, maintenance *entities.Maintenance) error {
	err := s.repository.Create(ctx, maintenance)
	if err == nil {
		_ = s.eventService.Publish(events.MaintenanceEvent{
			Maintenance: maintenance,
			Action:      "created",
		})
	}
	return err
}

func (s *ServiceImpl) Update(ctx context.Context, maintenance *entities.Maintenance) error {
	err := s.repository.Update(ctx, maintenance)
	if err == nil {
		_ = s.eventService.Publish(events.MaintenanceEvent{
			Maintenance: maintenance,
			Action:      "updated",
		})
	}
	return err
}

func (s *ServiceImpl) Delete(ctx context.Context, maintenance *entities.Maintenance) error {
	err := s.repository.Delete(ctx, maintenance)
	if err == nil {
		_ = s.eventService.Publish(events.MaintenanceEvent{
			Maintenance: maintenance,
			Action:      "completed",
		})
	}
	return err
}
