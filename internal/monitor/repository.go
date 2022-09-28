package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	GetMonitors(ctx context.Context) (*[]entities.Monitor, error)
	GetMonitorByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error)
	GetMonitorByIDAndTeamID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorAndSettingsByTeamIDAndID(ctx context.Context, teamID uint, monitorID uint) (*entities.Monitor, error)
	GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error)
	Create(ctx context.Context, monitor *entities.Monitor) error
	Update(ctx context.Context, monitor *entities.Monitor) error
	Delete(ctx context.Context, id uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) GetMonitors(ctx context.Context) (*[]entities.Monitor, error) {
	var monitors []entities.Monitor
	err := r.db.WithContext(ctx).Preload("Settings").Find(&monitors).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &monitors, err
}

func (r *RepositoryImpl) GetMonitorByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error) {
	var monitors []entities.Monitor
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Where(entities.Monitor{
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

func (r *RepositoryImpl) GetMonitorsAndSettingsByTeamID(ctx context.Context, teamID uint, offset int, limit int) (*[]entities.Monitor, error) {
	var monitors []entities.Monitor
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Preload("Settings").Where(entities.Monitor{
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

func (r *RepositoryImpl) Create(ctx context.Context, m *entities.Monitor) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, m *entities.Monitor) error {
	result := r.db.WithContext(ctx).Updates(m)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&entities.Monitor{}, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}
