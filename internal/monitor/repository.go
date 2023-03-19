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
	result := r.db.WithContext(ctx).
		Where(
			entities.Monitor{
				ID:     monitorID,
				TeamID: teamID,
			},
		).
		Select(
			"name",
			"state",
			"tags",
			"settings.method",
			"settings.url",
			"settings.headers",
			"settings.body",
			"settings.body_type",
			"settings.frequency",
		).Updates(&entities.Monitor{
		Name:  m.Name,
		State: m.State,
		Tags:  m.Tags,
		Settings: entities.MonitorSettings{
			Method:    m.Settings.Method,
			URL:       m.Settings.URL,
			Headers:   m.Settings.Headers,
			Body:      m.Settings.Body,
			BodyType:  m.Settings.BodyType,
			Frequency: m.Settings.Frequency,
		},
	})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return result.Error
	}

	return nil
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
