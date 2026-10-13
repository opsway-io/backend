package maintenance

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var (
	ErrNotFound       = errors.New("maintenance not found")
	ErrInvalidMonitor = errors.New("monitor does not belong to team")
)

type Repository interface {
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
	GetActiveByTeamID(ctx context.Context, teamID uint, at time.Time) (*[]entities.Maintenance, error)
	Create(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error
	Update(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error
	Delete(ctx context.Context, teamID, id uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) preloaded(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Settings").
		Preload("Monitors", func(db *gorm.DB) *gorm.DB {
			return db.Select("monitors.id", "monitors.team_id", "monitors.name").Order("monitors.id ASC")
		})
}

func (r *RepositoryImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.Maintenance, error) {
	var maintenance entities.Maintenance
	if err := r.preloaded(ctx).
		Where("maintenance.team_id = ? AND maintenance.id = ?", teamID, id).
		First(&maintenance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &maintenance, nil
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	var maintenances []entities.Maintenance
	if err := r.preloaded(ctx).
		Joins("JOIN maintenance_settings ON maintenance_settings.maintenance_id = maintenance.id").
		Where("maintenance.team_id = ?", teamID).
		Order("maintenance_settings.start_at DESC").
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

func (r *RepositoryImpl) GetActiveByTeamID(ctx context.Context, teamID uint, at time.Time) (*[]entities.Maintenance, error) {
	var maintenances []entities.Maintenance
	if err := r.preloaded(ctx).
		Joins("JOIN maintenance_settings ON maintenance_settings.maintenance_id = maintenance.id").
		Where("maintenance.team_id = ?", teamID).
		Where("maintenance_settings.start_at <= ? AND maintenance_settings.end_at > ?", at, at).
		Order("maintenance_settings.start_at ASC").
		Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

// resolveMonitors loads the given monitor ids, ensuring they all belong to the team.
func resolveMonitors(tx *gorm.DB, teamID uint, monitorIDs []uint) ([]entities.Monitor, error) {
	monitors := []entities.Monitor{}
	if len(monitorIDs) == 0 {
		return monitors, nil
	}

	unique := map[uint]struct{}{}
	for _, id := range monitorIDs {
		unique[id] = struct{}{}
	}

	if err := tx.Where("team_id = ? AND id IN ?", teamID, monitorIDs).Find(&monitors).Error; err != nil {
		return nil, err
	}

	if len(monitors) != len(unique) {
		return nil, ErrInvalidMonitor
	}

	return monitors, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		monitors, err := resolveMonitors(tx, maintenance.TeamID, monitorIDs)
		if err != nil {
			return err
		}

		maintenance.Monitors = nil
		if err := tx.Omit("Monitors").Create(maintenance).Error; err != nil {
			return err
		}

		if len(monitors) > 0 {
			if err := tx.Model(maintenance).Association("Monitors").Replace(monitors); err != nil {
				return err
			}
		}

		maintenance.Monitors = monitors

		return nil
	})
}

func (r *RepositoryImpl) Update(ctx context.Context, maintenance *entities.Maintenance, monitorIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		monitors, err := resolveMonitors(tx, maintenance.TeamID, monitorIDs)
		if err != nil {
			return err
		}

		result := tx.Model(&entities.Maintenance{}).
			Where("id = ? AND team_id = ?", maintenance.ID, maintenance.TeamID).
			Updates(map[string]any{
				"title":       maintenance.Title,
				"description": maintenance.Description,
				"updated_at":  time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		if err := tx.Model(&entities.MaintenanceSettings{}).
			Where("maintenance_id = ?", maintenance.ID).
			Updates(map[string]any{
				"start_at":   maintenance.Settings.StartAt,
				"end_at":     maintenance.Settings.EndAt,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(maintenance).Association("Monitors").Replace(monitors); err != nil {
			return err
		}

		maintenance.Monitors = monitors

		return nil
	})
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maintenance entities.Maintenance
		if err := tx.Where("id = ? AND team_id = ?", id, teamID).First(&maintenance).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}

			return err
		}

		return tx.Select("Settings", "Comments", "Monitors").Delete(&maintenance).Error
	})
}
