package incident

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("incident not found")

type Repository interface {
	GetByID(ctx context.Context, id uint) (*entities.Incident, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Incident, error)
	Create(ctx context.Context, incident *entities.Incident) error
	Update(ctx context.Context, incident *entities.Incident) error
	Delete(ctx context.Context, incident *entities.Incident) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	var incident entities.Incident
	if err := r.db.WithContext(
		ctx,
	).Where(entities.Incident{
		ID: id,
	}).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &incident, nil
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Incident, error) {
	var incidents []entities.Incident
	if err := r.db.WithContext(
		ctx,
	).Where(entities.Incident{
		TeamID: teamID,
	}).Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, incident *entities.Incident) error {
	return r.db.WithContext(ctx).Create(incident).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, incident *entities.Incident) error {
	result := r.db.WithContext(ctx).Updates(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, incident *entities.Incident) error {
	result := r.db.WithContext(ctx).Delete(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
