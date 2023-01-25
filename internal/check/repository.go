package check

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("probe result not found")

type Repository interface {
	Get(ctx context.Context, monitorID uint) (*[]Check, error)
	GetByIDAndMonitorID(ctx context.Context, monitorID uint, checkID uuid.UUID, offset *int, limit *int) (*Check, error)
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
	).Where(
		"monitor_id = ?",
		monitorID,
	).Order(
		"created_at desc",
	).Find(&checks).Error

	return &checks, err
}

func (r *RepositoryImpl) GetByIDAndMonitorID(ctx context.Context, monitorID uint, checkID uuid.UUID, offset *int, limit *int) (*Check, error) {
	var check Check
	err := r.db.WithContext(
		ctx,
	).Where(
		"monitor_id = ? AND id = ?",
		monitorID,
		checkID,
	).Scopes(
		clickhouse.Paginated(offset, limit),
	).First(
		&check,
	).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &check, nil
}

type AggMetric struct {
	Start      string
	DNS        float64
	TCP        float64
	TLS        float64
	Processing float64
	Transfer   float64
}

func (r *RepositoryImpl) GetAggMetrics(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
	var metrics []AggMetric
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		tumbleStart(wndw) as start, 
		avg(timing_dns_lookup)/1000000 as dns, 
		avg(timing_tcp_connection)/1000000 as tcp,
		avg(timing_tls_handshake)/1000000 as tls,
		avg(timing_server_processing)/1000000 as processing,
		avg(timing_content_transfer)/1000000 as transfer`).
		Where("monitor_id = ?", monitorID).
		Group("tumble(toDateTime(created_at), INTERVAL 1 MINUTE) as wndw").
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 24 HOUR) AND NOW()").
		Order("start ASC").
		Find(&metrics).Error

	return &metrics, err
}

func (r *RepositoryImpl) Create(ctx context.Context, check *Check) error {
	return r.db.WithContext(ctx).Create(check).Error
}
