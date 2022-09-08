package maintenance

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("maintenance not found")

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Maintenance, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error)
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
	).Where(entities.Maintenance{
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
	).Where(entities.Maintenance{
		TeamID: teamID,
	}).Find(&maintenances).Error; err != nil {
		return nil, err
	}

	return &maintenances, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, maintenance *entities.Maintenance) error {
	return r.db.WithContext(ctx).Create(maintenance).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, maintenance *entities.Maintenance) error {
	result := r.db.WithContext(ctx).Updates(maintenance)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
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
