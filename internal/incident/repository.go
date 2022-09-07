package incident

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("incident not found")

type Repository interface {
	GetByID(ctx context.Context, id uint) (*Incident, error)
	GetByTeamID(ctx context.Context, teamID uint) (*[]Incident, error)
	Create(ctx context.Context, incident *Incident) error
	Update(ctx context.Context, incident *Incident) error
	Delete(ctx context.Context, incident *Incident) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*Incident, error) {
	var incident Incident
	if err := r.db.WithContext(
		ctx,
	).Where(Incident{
		ID: id,
	}).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &incident, nil
}

func (r *RepositoryImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]Incident, error) {
	var incidents []Incident
	if err := r.db.WithContext(
		ctx,
	).Where(Incident{
		TeamID: teamID,
	}).Find(&incidents).Error; err != nil {
		return nil, err
	}

	return &incidents, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, incident *Incident) error {
	return r.db.WithContext(ctx).Create(incident).Error
}

func (r *RepositoryImpl) Update(ctx context.Context, incident *Incident) error {
	result := r.db.WithContext(ctx).Updates(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, incident *Incident) error {
	result := r.db.WithContext(ctx).Delete(incident)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
