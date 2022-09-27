package probes

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("probe result not found")

type Repository interface {
	Get(ctx context.Context, id uint) (*entities.ProbeResult, error)
	Create(ctx context.Context, maintenance *entities.ProbeResult) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Get(ctx context.Context, id uint) (*entities.ProbeResult, error) {
	var probeResult entities.ProbeResult
	if err := r.db.WithContext(
		ctx,
	).Where(entities.ProbeResult{
		ID: id,
	}).First(&probeResult).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &probeResult, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, probeResult *entities.ProbeResult) error {
	return r.db.WithContext(ctx).Create(probeResult).Error
}
