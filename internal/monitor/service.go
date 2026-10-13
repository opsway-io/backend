package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/boomerang"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// DefaultLocation is the probe location used when none are configured.
const DefaultLocation = "global"

type Service interface {
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
	GetMonitorsAndIncidentsByTeamID(ctx context.Context, teamID uint) (*[]entities.Monitor, error)
	GetMonitorAssertionByID(ctx context.Context, monitorAssertionID uint) (*entities.MonitorAssertion, error)
	GetStepByID(ctx context.Context, stepID uint) (*entities.MonitorStep, error)
	SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error
	Create(ctx context.Context, monitor *entities.Monitor) error
	CreateMany(ctx context.Context, monitors []*entities.Monitor) error
	Update(ctx context.Context, teamID, monitorID uint, monitor *entities.Monitor) error
	Delete(ctx context.Context, teamID, monitorID uint) error

	// Locations returns the configured probe location ids.
	Locations() []string
	// RescheduleActive re-adds all active monitors to the schedule so the
	// queued task payloads match the current data model.
	RescheduleActive(ctx context.Context) (int, error)
}

type ServiceImpl struct {
	repository Repository
	schedule   Schedule
	locations  []string
}

type Option func(*ServiceImpl)

// WithLocations sets the probe locations monitors may reference.
func WithLocations(locations []string) Option {
	return func(s *ServiceImpl) {
		s.locations = NormalizeLocations(locations)
	}
}

func NewService(db *gorm.DB, redisClient *redis.Client, opts ...Option) Service {
	s := &ServiceImpl{
		repository: NewRepository(db),
		schedule:   NewSchedule(redisClient),
		locations:  NormalizeLocations(nil),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// NormalizeLocations drops empty/duplicate ids and falls back to the default location.
func NormalizeLocations(locations []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(locations))

	for _, l := range locations {
		if l == "" || seen[l] {
			continue
		}

		seen[l] = true
		out = append(out, l)
	}

	if len(out) == 0 {
		return []string{DefaultLocation}
	}

	return out
}

func (s *ServiceImpl) Locations() []string {
	return s.locations
}

func (s *ServiceImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	return s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
}

func (s *ServiceImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error) {
	return s.repository.GetMonitorsAndSettingsByTeamID(ctx, teamID, offset, limit, query)
}

func (s *ServiceImpl) GetMonitorsAndIncidentsByTeamID(ctx context.Context, teamID uint) (*[]entities.Monitor, error) {
	return s.repository.GetMonitorsAndIncidentsByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetMonitorAssertionByID(ctx context.Context, monitorAssertionID uint) (*entities.MonitorAssertion, error) {
	return s.repository.GetMonitorAssertionByID(ctx, monitorAssertionID)
}

func (s *ServiceImpl) GetStepByID(ctx context.Context, stepID uint) (*entities.MonitorStep, error) {
	return s.repository.GetStepByID(ctx, stepID)
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
		if err := s.schedule.Remove(ctx, monitorID); err != nil && !errors.Is(err, boomerang.ErrTaskDoesNotExist) {
			return err
		}

		return nil
	}

	m, err := s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
	if err != nil {
		return err
	}

	return s.schedule.Upsert(ctx, m)
}

func (s *ServiceImpl) Create(ctx context.Context, m *entities.Monitor) error {
	m.State = entities.MonitorStateActive

	if err := s.repository.Create(ctx, m); err != nil {
		return err
	}

	return s.scheduleFresh(ctx, m.TeamID, m.ID)
}

func (s *ServiceImpl) CreateMany(ctx context.Context, monitors []*entities.Monitor) error {
	for _, m := range monitors {
		m.State = entities.MonitorStateActive
	}

	if err := s.repository.CreateMany(ctx, monitors); err != nil {
		return err
	}

	for _, m := range monitors {
		if err := s.scheduleFresh(ctx, m.TeamID, m.ID); err != nil {
			return err
		}
	}

	return nil
}

// scheduleFresh reloads the monitor (so steps and assertions carry their IDs)
// and (re)schedules it.
func (s *ServiceImpl) scheduleFresh(ctx context.Context, teamID, monitorID uint) error {
	m, err := s.repository.GetMonitorAndSettingsByTeamIDAndID(ctx, teamID, monitorID)
	if err != nil {
		return err
	}

	return s.schedule.Upsert(ctx, m)
}

func (s *ServiceImpl) Update(ctx context.Context, teamID, monitorID uint, m *entities.Monitor) error {
	m.ID = monitorID

	if err := s.repository.Update(ctx, teamID, monitorID, m); err != nil {
		return err
	}

	if m.State == entities.MonitorStateInactive {
		if err := s.schedule.Remove(ctx, monitorID); err != nil && !errors.Is(err, boomerang.ErrTaskDoesNotExist) {
			return err
		}

		return nil
	}

	return s.scheduleFresh(ctx, teamID, monitorID)
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

func (s *ServiceImpl) RescheduleActive(ctx context.Context) (int, error) {
	monitors, err := s.repository.GetActiveMonitors(ctx)
	if err != nil {
		return 0, err
	}

	for i := range monitors {
		if err := s.schedule.Upsert(ctx, &monitors[i]); err != nil {
			return i, err
		}
	}

	return len(monitors), nil
}
