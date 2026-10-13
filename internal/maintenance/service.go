package maintenance

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
)

type Service interface {
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	GetActiveByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	Create(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error
	Update(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error
	Delete(ctx context.Context, teamID, id uint) error

	// IsMonitorUnderMaintenance reports whether an active maintenance window
	// currently covers the monitor. Intended for monitor GET responses and alert suppression.
	IsMonitorUnderMaintenance(ctx context.Context, teamID, monitorID uint) (bool, error)
	// MonitorsUnderMaintenance returns a predicate telling whether a monitor of
	// the team is covered by an active maintenance window (one query for many monitors).
	MonitorsUnderMaintenance(ctx context.Context, teamID uint) (func(monitorID uint) bool, error)
}

type ServiceImpl struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
		now:        time.Now,
	}
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Maintenance, error) {
	return s.repository.GetByTeamIDAndID(ctx, teamID, id)
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	return s.repository.GetByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetActiveByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	return s.repository.GetActiveByTeamID(ctx, teamID, s.now())
}

func (s *ServiceImpl) Create(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error {
	return s.repository.Create(ctx, maintenance, monitorIDs)
}

func (s *ServiceImpl) Update(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error {
	return s.repository.Update(ctx, maintenance, monitorIDs)
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, id uint) error {
	return s.repository.Delete(ctx, teamID, id)
}

func (s *ServiceImpl) IsMonitorUnderMaintenance(ctx context.Context, teamID, monitorID uint) (bool, error) {
	covered, err := s.MonitorsUnderMaintenance(ctx, teamID)
	if err != nil {
		return false, err
	}

	return covered(monitorID), nil
}

func (s *ServiceImpl) MonitorsUnderMaintenance(ctx context.Context, teamID uint) (func(monitorID uint) bool, error) {
	active, err := s.repository.GetActiveByTeamID(ctx, teamID, s.now())
	if err != nil {
		return nil, err
	}

	return CoversMonitor(*active), nil
}

// CoversMonitor builds a predicate from a set of (active) maintenance windows.
func CoversMonitor(windows []entities.Maintenance) func(monitorID uint) bool {
	all := false
	ids := map[uint]struct{}{}

	for _, w := range windows {
		if len(w.Monitors) == 0 {
			all = true

			break
		}

		for _, m := range w.Monitors {
			ids[m.ID] = struct{}{}
		}
	}

	return func(monitorID uint) bool {
		if all {
			return true
		}

		_, ok := ids[monitorID]

		return ok
	}
}
