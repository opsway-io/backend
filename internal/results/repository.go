package results

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("maintenance not found")

type Repository interface {
	Get(ctx context.Context, id uint) (*ProbeResult, error)
	Create(ctx context.Context, maintenance *ProbeResult) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Get(ctx context.Context, id uint) (*ProbeResult, error) {
	var probeResult ProbeResult
	if err := r.db.WithContext(
		ctx,
	).Where(ProbeResult{
		ID: id,
	}).First(&probeResult).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &probeResult, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, probeResult *ProbeResult) error {
	return r.db.WithContext(ctx).Create(probeResult).Error
}
