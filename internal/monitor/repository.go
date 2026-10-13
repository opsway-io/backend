package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
	GetMonitorsAndIncidentsByTeamID(ctx context.Context, teamID uint) (*[]entities.Monitor, error)
	GetMonitorAssertionByID(ctx context.Context, monitorAssertionID uint) (*entities.MonitorAssertion, error)
	GetStepByID(ctx context.Context, stepID uint) (*entities.MonitorStep, error)
	GetActiveMonitors(ctx context.Context) ([]entities.Monitor, error)
	SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error
	Create(ctx context.Context, monitor *entities.Monitor) error
	CreateMany(ctx context.Context, monitors []*entities.Monitor) error
	Update(ctx context.Context, teamID, monitorID uint, monitor *entities.Monitor) error
	Delete(ctx context.Context, teamID, monitorID uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

// preloadMonitor loads settings and ordered steps with their ordered assertions.
func preloadMonitor(db *gorm.DB) *gorm.DB {
	return db.Preload(
		"Settings",
	).Preload(
		"Steps", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, id asc") },
	).Preload(
		"Steps.Assertions", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, id asc") },
	).Preload(
		"Assertions", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, id asc") },
	)
}

func (r *RepositoryImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	var monitor entities.Monitor
	err := preloadMonitor(r.db.WithContext(ctx)).Where(entities.Monitor{
		ID:     monitorID,
		TeamID: teamID,
	}).First(&monitor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &monitor, err
}

type MonitorWithTotalCount struct {
	entities.Monitor
	TotalCount int
}

func (r *RepositoryImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error) {
	var monitors []MonitorWithTotalCount
	err := preloadMonitor(r.db.WithContext(
		ctx,
	).Scopes(
		postgres.Paginated(offset, limit),
		postgres.IncludeTotalCount("total_count"),
		postgres.Search([]string{"name"}, query),
	)).Where(entities.Monitor{
		TeamID: teamID,
	}).Order(
		"created_at asc",
	).Find(&monitors).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &monitors, err
}

func (r *RepositoryImpl) GetActiveMonitors(ctx context.Context) ([]entities.Monitor, error) {
	var monitors []entities.Monitor
	err := preloadMonitor(r.db.WithContext(ctx)).Where(
		"state = ?", entities.MonitorStateActive,
	).Find(&monitors).Error

	return monitors, err
}

func (r *RepositoryImpl) GetMonitorsAndIncidentsByTeamID(ctx context.Context, teamID uint) (*[]entities.Monitor, error) {
	var monitors []entities.Monitor
	err := r.db.WithContext(
		ctx,
	).Preload("Incidents", "resolved = ?", false).
		Where(entities.Monitor{
			TeamID: teamID,
		}).Find(&monitors).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &monitors, err
}

func (r *RepositoryImpl) GetMonitorAssertionByID(ctx context.Context, monitorAssertionID uint) (*entities.MonitorAssertion, error) {
	var monitorAssertion entities.MonitorAssertion
	err := r.db.WithContext(
		ctx,
	).Where(entities.MonitorAssertion{
		ID: monitorAssertionID,
	}).First(&monitorAssertion).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &monitorAssertion, err
}

func (r *RepositoryImpl) GetStepByID(ctx context.Context, stepID uint) (*entities.MonitorStep, error) {
	var step entities.MonitorStep
	err := r.db.WithContext(ctx).Where(entities.MonitorStep{ID: stepID}).First(&step).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &step, err
}

func (r *RepositoryImpl) SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error {
	res := r.db.WithContext(ctx).Model(
		&entities.Monitor{},
	).Where(entities.Monitor{
		ID:     monitorID,
		TeamID: teamID,
	}).Update("state", state)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) Create(ctx context.Context, m *entities.Monitor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return createMonitor(tx, m)
	})
}

func (r *RepositoryImpl) CreateMany(ctx context.Context, monitors []*entities.Monitor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, m := range monitors {
			if err := createMonitor(tx, m); err != nil {
				return err
			}
		}

		return nil
	})
}

func createMonitor(tx *gorm.DB, m *entities.Monitor) error {
	syncLegacySettings(m)

	steps := m.Steps
	m.Steps = nil
	m.Assertions = nil

	if err := tx.Omit("Steps", "Assertions", "Incidents").Create(m).Error; err != nil {
		return err
	}

	for i := range steps {
		steps[i].ID = 0
		if err := createStep(tx, m.ID, i, &steps[i]); err != nil {
			return err
		}
	}

	m.Steps = steps

	return nil
}

func createStep(tx *gorm.DB, monitorID uint, position int, step *entities.MonitorStep) error {
	assertions := step.Assertions
	step.Assertions = nil
	step.MonitorID = monitorID
	step.Position = position

	if err := tx.Omit(clause.Associations).Create(step).Error; err != nil {
		return err
	}

	for j := range assertions {
		assertions[j].ID = 0
		assertions[j].MonitorID = monitorID
		assertions[j].StepID = &step.ID
		assertions[j].Position = j
	}

	if len(assertions) > 0 {
		if err := tx.Create(&assertions).Error; err != nil {
			return err
		}
	}

	step.Assertions = assertions

	return nil
}

// syncLegacySettings mirrors the first step into the legacy (single request)
// settings columns, which are NOT NULL and still read by older code paths.
func syncLegacySettings(m *entities.Monitor) {
	if len(m.Steps) == 0 {
		return
	}

	first := m.Steps[0]
	m.Settings.Method = first.Method
	m.Settings.URL = first.URL
	m.Settings.Headers = first.Headers
	m.Settings.Body = first.Body
}

// Update replaces the monitor's settings and steps. Steps and assertions are
// matched by position and updated in place, so their IDs (referenced by
// incidents) stay stable across edits.
func (r *RepositoryImpl) Update(ctx context.Context, teamID, monitorID uint, m *entities.Monitor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(
			&entities.Monitor{},
		).Where(entities.Monitor{
			ID:     monitorID,
			TeamID: teamID,
		}).Updates(map[string]interface{}{
			"name":  m.Name,
			"state": m.State,
		})
		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return ErrNotFound
		}

		syncLegacySettings(m)

		// Select("*") so zero values (e.g. tls disabled, empty auth) are written too
		if err := tx.Model(
			&entities.MonitorSettings{},
		).Where(
			"monitor_id = ?", monitorID,
		).Select(
			"method", "url", "frequency", "headers",
			"body_content", "body_type",
			"tls_enabled", "tls_verify_hostname", "tls_check_expiration", "tls_expiration_threshold_days",
			"auth_method", "auth_token_url", "auth_client_id", "auth_client_secret", "auth_username", "auth_password",
			"locations", "updated_at",
		).Updates(&m.Settings).Error; err != nil {
			return err
		}

		var existing []entities.MonitorStep
		if err := tx.Preload(
			"Assertions", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, id asc") },
		).Where(
			"monitor_id = ?", monitorID,
		).Order("position asc, id asc").Find(&existing).Error; err != nil {
			return err
		}

		for i := range m.Steps {
			step := &m.Steps[i]

			if i >= len(existing) {
				step.ID = 0
				if err := createStep(tx, monitorID, i, step); err != nil {
					return err
				}

				continue
			}

			old := existing[i]
			step.ID = old.ID
			step.MonitorID = monitorID
			step.Position = i

			if err := tx.Model(&entities.MonitorStep{ID: old.ID}).Select(
				"position", "name", "method", "url", "headers", "body_content", "body_type", "variables", "updated_at",
			).Updates(step).Error; err != nil {
				return err
			}

			if err := updateStepAssertions(tx, monitorID, old, step); err != nil {
				return err
			}
		}

		// Remove steps (and their assertions) that no longer exist
		if len(existing) > len(m.Steps) {
			removed := make([]uint, 0, len(existing)-len(m.Steps))
			for _, s := range existing[len(m.Steps):] {
				removed = append(removed, s.ID)
			}

			if err := tx.Where("step_id IN ?", removed).Delete(&entities.MonitorAssertion{}).Error; err != nil {
				return err
			}

			if err := tx.Where("id IN ?", removed).Delete(&entities.MonitorStep{}).Error; err != nil {
				return err
			}
		}

		// Assertions not attached to any step are leftovers of the legacy model
		return tx.Where("monitor_id = ? AND step_id IS NULL", monitorID).Delete(&entities.MonitorAssertion{}).Error
	})
}

func updateStepAssertions(tx *gorm.DB, monitorID uint, old entities.MonitorStep, step *entities.MonitorStep) error {
	for j := range step.Assertions {
		a := &step.Assertions[j]
		a.MonitorID = monitorID
		a.StepID = &old.ID
		a.Position = j

		if j < len(old.Assertions) {
			a.ID = old.Assertions[j].ID
			if err := tx.Model(&entities.MonitorAssertion{ID: a.ID}).Select(
				"position", "source", "property", "operator", "target", "updated_at",
			).Updates(a).Error; err != nil {
				return err
			}

			continue
		}

		a.ID = 0
		if err := tx.Create(a).Error; err != nil {
			return err
		}
	}

	if len(old.Assertions) > len(step.Assertions) {
		removed := make([]uint, 0, len(old.Assertions)-len(step.Assertions))
		for _, a := range old.Assertions[len(step.Assertions):] {
			removed = append(removed, a.ID)
		}

		if err := tx.Where("id IN ?", removed).Delete(&entities.MonitorAssertion{}).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, monitorID uint) error {
	res := r.db.WithContext(ctx).
		Where(entities.Monitor{
			ID:     monitorID,
			TeamID: teamID,
		}).Delete(&entities.Monitor{})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
