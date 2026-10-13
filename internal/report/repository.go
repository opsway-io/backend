package report

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("monitor not found")

type Repository interface {
	GetReportsByTeamID(ctx context.Context, teamID uint) (*[]entities.Report, error)
	GetByID(ctx context.Context, id uint) (*entities.Report, error)
	Create(ctx context.Context, rep *entities.Report) error
	Update(ctx context.Context, rep *entities.Report) error
	Delete(ctx context.Context, teamID, reportID uint) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
	}
}

func (r *RepositoryImpl) GetReportsByTeamID(ctx context.Context, teamID uint) (*[]entities.Report, error) {
	var reports []entities.Report
	err := r.db.WithContext(
		ctx,
	).Where(entities.Report{
		TeamID: teamID,
	}).Find(&reports).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &reports, err
}

func (r *RepositoryImpl) Create(ctx context.Context, rep *entities.Report) error {
	return r.db.WithContext(ctx).Create(rep).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, reportID uint) error {
	res := r.db.WithContext(ctx).
		Where(entities.Report{
			ID:     reportID,
			TeamID: teamID,
		}).Delete(&entities.Report{})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Report, error) {
	var rep entities.Report
	err := r.db.WithContext(ctx).First(&rep, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rep, nil
}

func (r *RepositoryImpl) Update(ctx context.Context, rep *entities.Report) error {
	return r.db.WithContext(ctx).Save(rep).Error
}
