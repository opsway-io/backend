package incident

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
)

// CreatedHook is invoked after a new incident has been created (not when an existing one is touched).
type CreatedHook func(ctx context.Context, incident *entities.Incident)

// Change describes a user driven state change of an incident.
type Change string

const (
	ChangeResolved    Change = "resolved"
	ChangeReopened    Change = "reopened"
	ChangeMadeVisible Change = "made_visible"
	ChangeHidden      Change = "hidden"
)

// ChangedHook is invoked after an incident was resolved/reopened or its status page visibility changed.
type ChangedHook func(ctx context.Context, incident *entities.Incident, change Change)

type Option func(*ServiceImpl)

// WithChangedHook registers a hook called after incident state changes, e.g. to notify subscribers.
func WithChangedHook(hook ChangedHook) Option {
	return func(s *ServiceImpl) {
		s.changedHooks = append(s.changedHooks, hook)
	}
}

// WithCreatedHook registers a hook called for every newly created incident, e.g. alert rule evaluation.
func WithCreatedHook(hook CreatedHook) Option {
	return func(s *ServiceImpl) {
		s.createdHooks = append(s.createdHooks, hook)
	}
}

type Service interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Incident, error)
	GetByTeamIDPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]IncidentWithOccurrences, error)
	GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error)
	GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, int64, error)
	CountOccurrences(ctx context.Context, incidentID uint) (int64, error)

	// Upsert creates or touches the incident of each failed assertion, recording an occurrence for each
	// and running the created hooks for incidents that are new.
	Upsert(ctx context.Context, incidents *[]entities.Incident) error
	Create(ctx context.Context, incidents *[]entities.Incident) error

	SetResolved(ctx context.Context, teamID, id uint, resolved bool, userID uint) error
	Acknowledge(ctx context.Context, teamID, id uint, userID uint) error
	SetStatusPageVisibility(ctx context.Context, teamID, id uint, visible bool) error

	Delete(ctx context.Context, incident *entities.Incident) error
}

type ServiceImpl struct {
	repository   Repository
	createdHooks []CreatedHook
	changedHooks []ChangedHook
}

func NewService(repository Repository, opts ...Option) Service {
	s := &ServiceImpl{
		repository: repository,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Incident, error) {
	return s.repository.GetByTeamIDAndID(ctx, teamID, id)
}

func (s *ServiceImpl) GetByTeamIDPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]IncidentWithOccurrences, error) {
	return s.repository.GetByTeamIDPaginated(ctx, teamID, resolved, offset, limit)
}

func (s *ServiceImpl) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]IncidentAndAssertion, error) {
	return s.repository.GetByMonitorIDWithAssertionPaginated(ctx, monitorID, offset, limit)
}

func (s *ServiceImpl) GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (*[]entities.IncidentOccurrence, int64, error) {
	return s.repository.GetOccurrencesPaginated(ctx, incidentID, offset, limit)
}

func (s *ServiceImpl) CountOccurrences(ctx context.Context, incidentID uint) (int64, error) {
	return s.repository.CountOccurrences(ctx, incidentID)
}

func (s *ServiceImpl) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	for i := range *incidents {
		in := &(*incidents)[i]

		created, err := s.repository.CreateOrTouch(ctx, in)
		if err != nil {
			return err
		}

		if created {
			for _, hook := range s.createdHooks {
				hook(ctx, in)
			}
		}
	}

	return nil
}

func (s *ServiceImpl) Create(ctx context.Context, incidents *[]entities.Incident) error {
	return s.repository.Create(ctx, incidents)
}

func (s *ServiceImpl) SetResolved(ctx context.Context, teamID, id uint, resolved bool, userID uint) error {
	current, err := s.repository.GetByTeamIDAndID(ctx, teamID, id)
	if err != nil {
		return err
	}
	if current.Resolved == resolved {
		return nil
	}

	fields := map[string]any{
		"resolved":            resolved,
		"resolved_at":         nil,
		"resolved_by_user_id": nil,
	}

	if resolved {
		fields["resolved_at"] = time.Now()
		fields["resolved_by_user_id"] = userID
	}

	if err := s.repository.UpdateFields(ctx, teamID, id, fields); err != nil {
		return err
	}

	change := ChangeReopened
	if resolved {
		change = ChangeResolved
	}

	s.runChangedHooks(ctx, teamID, id, change)

	return nil
}

func (s *ServiceImpl) runChangedHooks(ctx context.Context, teamID, id uint, change Change) {
	if len(s.changedHooks) == 0 {
		return
	}

	in, err := s.repository.GetByTeamIDAndID(ctx, teamID, id)
	if err != nil {
		return
	}

	for _, hook := range s.changedHooks {
		hook(ctx, in, change)
	}
}

func (s *ServiceImpl) Acknowledge(ctx context.Context, teamID, id uint, userID uint) error {
	return s.repository.UpdateFields(ctx, teamID, id, map[string]any{
		"acknowledged":                true,
		"acknowledged_at":             time.Now(),
		"acknowledged_by_user_id":     userID,
		"acknowledged_by_integration": nil,
	})
}

func (s *ServiceImpl) SetStatusPageVisibility(ctx context.Context, teamID, id uint, visible bool) error {
	current, err := s.repository.GetByTeamIDAndID(ctx, teamID, id)
	if err != nil {
		return err
	}
	if current.IsStatusPageVisible == visible {
		return nil
	}

	if err := s.repository.UpdateFields(ctx, teamID, id, map[string]any{
		"is_status_page_visible": visible,
	}); err != nil {
		return err
	}

	change := ChangeHidden
	if visible {
		change = ChangeMadeVisible
	}

	s.runChangedHooks(ctx, teamID, id, change)

	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, incident *entities.Incident) error {
	return s.repository.Delete(ctx, incident)
}
