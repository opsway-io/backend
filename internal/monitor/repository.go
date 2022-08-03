package monitor

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	Create(ctx context.Context, monitor *Monitor) error
	Update(ctx context.Context, monitor *Monitor) error
	Delete(ctx context.Context, id int) error
	GetByTeamID(ctx context.Context, teamID int, offset int, limit int) ([]Monitor, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id int) (*Monitor, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) Create(ctx context.Context, m *Monitor) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, m *Monitor) error {
	err := r.db.WithContext(ctx).Updates(m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int) error {
	err := r.db.WithContext(ctx).Delete(&Monitor{}, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID int, offset int, limit int) ([]Monitor, error) {
	var monitors []Monitor
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Where(Monitor{
		TeamID: teamID,
	}).Find(&monitors).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []Monitor{}, nil
		}

		return nil, err
	}

	return monitors, err
}

func (r *RepositoryImpl) GetByTeamIDAndID(ctx context.Context, teamID, id int) (*Monitor, error) {
	var monitor Monitor
	err := r.db.WithContext(ctx).Where(Monitor{
		ID:     id,
		TeamID: teamID,
	}).First(&monitor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return &monitor, err
}
