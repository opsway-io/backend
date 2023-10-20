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
	GetMonitorByIDAndTeamID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset *int, limit *int, query *string) (*[]MonitorWithTotalCount, error)
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

func (r *RepositoryImpl) GetMonitorByIDAndTeamID(ctx context.Context, monitorID uint, teamID uint) (*entities.Monitor, error) {
	var monitor entities.Monitor
	err := r.db.WithContext(ctx).Where(entities.Monitor{
		ID:     monitorID,
		TeamID: teamID,
	}).First(&monitor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &monitor, err
}

func (r *RepositoryImpl) GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error) {
	var monitor entities.Monitor
	err := r.db.WithContext(ctx).Preload("Settings").Where(entities.Monitor{
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

func (r *RepositoryImpl) Create(ctx context.Context, m *entities.Monitor) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, teamID, monitorID uint, m *entities.Monitor) error {
	tx := r.db.WithContext(ctx).Begin()

	result := tx.
		Where(
			entities.Monitor{
				ID:     monitorID,
				TeamID: teamID,
			},
		).
		Select(
			"name",
			"state",
		).Updates(
		&entities.Monitor{
			Name:  m.Name,
			State: m.State,
		},
	)

	if result.Error != nil {
		tx.Rollback()

		return result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()

		return ErrNotFound
	}

	// update settings
	result = tx.
		Where(
			entities.MonitorSettings{
				MonitorID: int(monitorID),
			},
		).
		Select(
			"method",
			"url",
			"headers",
			"body",
			"body_type",
			"frequency",
		).Updates(&entities.MonitorSettings{
		Method:    m.Settings.Method,
		URL:       m.Settings.URL,
		Headers:   m.Settings.Headers,
		Body:      m.Settings.Body,
		BodyType:  m.Settings.BodyType,
		Frequency: m.Settings.Frequency,
	})
	if result.Error != nil {
		tx.Rollback()

		return result.Error
	}

	// Commit transaction
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
