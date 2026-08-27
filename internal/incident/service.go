package incident

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
)

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error)
	GetByTeamIDAndStatusPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]entities.Incident, error)
	GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error)
	GetActiveByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Incident, error)
	Upsert(ctx context.Context, incidents *[]entities.Incident) error
	Create(ctx context.Context, incidents *[]entities.Incident) error
	Update(ctx context.Context, incident *entities.Incident) error
	Delete(ctx context.Context, incident *entities.Incident) error
	GetByTeamIDMonitorsIncidentStats(ctx context.Context, teamID uint, start, end string) (*[]entities.MonitorIncident, error)
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

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error) {
	return s.repository.GetByTeamIDPaginated(ctx, teamID, offset, limit)
}

func (s *ServiceImpl) GetByTeamIDAndStatusPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]entities.Incident, error) {
	return s.repository.GetByTeamIDAndStatusPaginated(ctx, teamID, resolved, offset, limit)
}

func (s *ServiceImpl) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error) {
	return s.repository.GetByMonitorIDWithAssertionPaginated(ctx, monitorID, offset, limit)
}

func (s *ServiceImpl) GetActiveByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Incident, error) {
	return s.repository.GetActiveByMonitorIDs(ctx, monitorIDs)
}

func (s *ServiceImpl) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	err := s.repository.Upsert(ctx, incidents)
	if err == nil && incidents != nil {
		for i := range *incidents {
			_ = s.eventService.Publish(events.IncidentCreatedEvent{
				Incident: &(*incidents)[i],
			})
		}
	}
	return err
}

func (s *ServiceImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	err := s.repository.Create(ctx, incidents)
	if err == nil && incidents != nil {
		for i := range *incidents {
			_ = s.eventService.Publish(events.IncidentCreatedEvent{
				Incident: &(*incidents)[i],
			})
		}
	}
	return err
}

func (s *ServiceImpl) Update(ctx context.Context, incident *entities.Incident) error {
	return s.repository.Update(ctx, incident)
}

func (s *ServiceImpl) Delete(ctx context.Context, incident *entities.Incident) error {
	return s.repository.Delete(ctx, incident)
}

func (s *ServiceImpl) GetByTeamIDMonitorsIncidentStats(ctx context.Context, teamID uint, start, end string) (*[]entities.MonitorIncident, error) {
	return s.repository.GetByTeamIDMonitorsIncidentStats(ctx, teamID, start, end)
}
