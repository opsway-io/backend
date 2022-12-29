package check

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("probe result not found")

type Repository interface {
	Get(ctx context.Context, monitorID uint) (*[]Check, error)
	GetAggMetrics(ctx context.Context, monitorID uint) (*[]AggMetric, error)
	Create(ctx context.Context, maintenance *Check) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Get(ctx context.Context, monitorID uint) (*[]Check, error) {
	var checks []Check
	err := r.db.WithContext(
		ctx,
	).Where("monitor_id = ?", monitorID).Find(&checks).Error

	return &checks, err
}

type AggMetric struct {
	Start  string
	Timing string
}

func (r *RepositoryImpl) GetAggMetrics(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
	var metrics []AggMetric
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select("tumbleStart(wndw) as start, avg(JSONExtractFloat(timing, 'total')) as timing").
		Where("monitor_id = ?", monitorID).
		Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw").
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 24 HOUR) AND NOW()").
		Find(&metrics).Error

	return &metrics, err
}

func (r *RepositoryImpl) Create(ctx context.Context, check *Check) error {
	return r.db.WithContext(ctx).Create(check).Error
}
