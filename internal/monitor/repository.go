package monitor

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	Create(ctx context.Context, monitor *entities.Monitor) error
	Update(ctx context.Context, monitor *entities.Monitor) error
	Delete(ctx context.Context, id int) error
	GetByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error)
	GetByIDAndTeamID(ctx context.Context, teamID, id int) (*entities.Monitor, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
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

func (r *RepositoryImpl) Delete(ctx context.Context, id int) error {
	err := r.db.WithContext(ctx).Delete(&entities.Monitor{}, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID int, offset int, limit int) (*[]entities.Monitor, error) {
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

func (r *RepositoryImpl) GetByIDAndTeamID(ctx context.Context, id, teamID int) (*entities.Monitor, error) {
	var monitor entities.Monitor
	err := r.db.WithContext(ctx).Where(entities.Monitor{
		ID:     id,
		TeamID: teamID,
	}).First(&monitor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &monitor, err
}
