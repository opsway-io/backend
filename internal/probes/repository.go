package probes

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("probe result not found")

type Repository interface {
	Get(ctx context.Context, monitorID uint64) (*[]entities.HttpResult, error)
	Create(ctx context.Context, maintenance *entities.HttpResult) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Get(ctx context.Context, monitorID uint64) (*[]entities.HttpResult, error) {
	var probeResults []entities.HttpResult
	err := r.db.WithContext(
		ctx,
	).Where("monitor_id = ?", monitorID).Find(&probeResults).Error

	return &probeResults, err
}

func (r *RepositoryImpl) Create(ctx context.Context, probeResult *entities.HttpResult) error {
	return r.db.WithContext(ctx).Create(probeResult).Error
}
