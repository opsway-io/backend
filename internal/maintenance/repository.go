package maintenance

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("maintenance not found")

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	GetActive(ctx context.Context, now time.Time) (*[]entities.Maintenance, error)
	GetActiveByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error)
	GetUpcomingByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error)
	GetAllByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Maintenance, error)
	GetUnnotified(ctx context.Context, now time.Time) (*[]entities.Maintenance, error)
	Create(ctx context.Context, maintenance *entities.Maintenance) error
	Update(ctx context.Context, maintenance *entities.Maintenance) error
	Delete(ctx context.Context, maintenance *entities.Maintenance) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Maintenance, error) {
	var maintenance entities.Maintenance
	if err := r.db.WithContext(
		ctx,
	).Preload("Settings").Preload("Monitors").Where(entities.Maintenance{
		ID: id,
	}).First(&maintenance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &maintenance, nil
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	var maintenances []entities.Maintenance
	if err := r.db.WithContext(
		ctx,
	).Preload("Settings").Preload("Monitors").Where(entities.Maintenance{
		TeamID: teamID,
	}).Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

func (r *RepositoryImpl) GetActive(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	var maintenances []entities.Maintenance
	if err := r.db.WithContext(
		ctx,
	).Joins("JOIN maintenance_settings ms ON ms.maintenance_id = maintenance.id").
		Preload("Settings").
		Preload("Monitors").
		Where("ms.start_at <= ? AND ms.end_at > ?", now, now).
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

func (r *RepositoryImpl) GetUnnotified(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	var maintenances []entities.Maintenance
	if err := r.db.WithContext(
		ctx,
	).Joins("JOIN maintenance_settings ms ON ms.maintenance_id = maintenance.id").
		Preload("Settings").
		Preload("Monitors").
		Where("ms.notified = ?", false).
		Where("ms.end_at > ?", now).
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

func (r *RepositoryImpl) GetActiveByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	if len(monitorIDs) == 0 {
		return []entities.Maintenance{}, nil
	}

	var maintenances []entities.Maintenance
	if err := r.db.WithContext(ctx).
		Joins("JOIN maintenance_settings ms ON ms.maintenance_id = maintenance.id").
		Joins("JOIN maintenance_monitors mm ON mm.maintenance_id = maintenance.id").
		Preload("Settings").
		Preload("Monitors").
		Where("ms.start_at <= ? AND ms.end_at > ?", now, now).
		Where("mm.monitor_id IN ?", monitorIDs).
		Group("maintenance.id").
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return maintenances, nil
}

func (r *RepositoryImpl) GetUpcomingByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	if len(monitorIDs) == 0 {
		return []entities.Maintenance{}, nil
	}

	var maintenances []entities.Maintenance
	if err := r.db.WithContext(ctx).
		Joins("JOIN maintenance_settings ms ON ms.maintenance_id = maintenance.id").
		Joins("JOIN maintenance_monitors mm ON mm.maintenance_id = maintenance.id").
		Preload("Settings").
		Preload("Monitors").
		Where("ms.end_at > ?", now).
		Where("mm.monitor_id IN ?", monitorIDs).
		Group("maintenance.id").
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return maintenances, nil
}

func (r *RepositoryImpl) GetAllByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Maintenance, error) {
	if len(monitorIDs) == 0 {
		return []entities.Maintenance{}, nil
	}

	var maintenances []entities.Maintenance
	if err := r.db.WithContext(ctx).
		Joins("JOIN maintenance_settings ms ON ms.maintenance_id = maintenance.id").
		Joins("LEFT JOIN maintenance_monitors mm ON mm.maintenance_id = maintenance.id").
		Preload("Settings").
		Preload("Monitors").
		Where("mm.monitor_id IN ? OR mm.monitor_id IS NULL", monitorIDs).
		Group("maintenance.id").
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return maintenances, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, maintenance *entities.Maintenance) error {
	return r.db.WithContext(ctx).Create(maintenance).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, maintenance *entities.Maintenance) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if maintenance.Monitors != nil {
			if err := tx.Model(maintenance).Association("Monitors").Replace(maintenance.Monitors); err != nil {
				return err
			}
		}

		result := tx.Updates(maintenance)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		if err := tx.Save(&maintenance.Settings).Error; err != nil {
			return err
		}

		return nil
	})

	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, maintenance *entities.Maintenance) error {
	result := r.db.WithContext(ctx).Delete(maintenance)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
