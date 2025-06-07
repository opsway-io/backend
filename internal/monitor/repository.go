package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
	GetMonitorsAndIncidentsByTeamID(ctx context.Context, teamID uint) (*[]entities.Monitor, error)
	SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error
	Create(ctx context.Context, monitor *entities.Monitor) error
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

func (r *RepositoryImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	var monitor entities.Monitor
	err := r.db.WithContext(
		ctx,
	).Preload(
		"Settings",
	).Preload(
		"Assertions",
	).Where(entities.Monitor{
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
	err := r.db.WithContext(
		ctx,
	).Scopes(
		postgres.Paginated(offset, limit),
		postgres.IncludeTotalCount("total_count"),
		postgres.Search([]string{"name"}, query),
	).Preload(
		"Settings",
	).Preload(
		"Assertions",
	).Where(entities.Monitor{
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

func (r *RepositoryImpl) SetState(ctx context.Context, teamID, monitorID uint, state entities.MonitorState) error {
	err := r.db.WithContext(ctx).Model(
		&entities.Monitor{},
	).Where(entities.Monitor{
		ID:     monitorID,
		TeamID: teamID,
	}).Update("state", state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (r *RepositoryImpl) Create(ctx context.Context, m *entities.Monitor) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, teamID, monitorID uint, m *entities.Monitor) error {
	tx := r.db.WithContext(ctx).Begin()

	// Update monitor name and state
	if err := tx.Model(
		&entities.Monitor{},
	).Where(entities.Monitor{
		ID:     monitorID,
		TeamID: teamID,
	}).Updates(entities.Monitor{
		Name:  m.Name,
		State: m.State,
	}).Error; err != nil {
		tx.Rollback()

		return err
	}

	// Update monitor settings
	if err := tx.Model(
		&entities.MonitorSettings{},
	).Where(entities.MonitorSettings{
		MonitorID: monitorID,
	}).Updates(m.Settings).Error; err != nil {
		tx.Rollback()

		return err
	}

	// Replace assertions
	if err := tx.Delete(&entities.MonitorAssertion{}, "monitor_id = ?", monitorID).Error; err != nil {
		tx.Rollback()

		return err
	}

	for i := range m.Assertions {
		m.Assertions[i].MonitorID = monitorID
	}
	if err := tx.Create(&m.Assertions).Error; err != nil {
		tx.Rollback()

		return err
	}

	return tx.Commit().Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, monitorID uint) error {
	err := r.db.WithContext(ctx).
		Where(entities.Monitor{
			ID:     monitorID,
			TeamID: teamID,
		}).Delete(&entities.Monitor{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}
