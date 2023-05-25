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
	Create(ctx context.Context, maintenance *Check) error
	GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*Check, error)
	GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]Check, error)
	GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint) (*[]AggMetric, error)
	GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error)
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*Check, error) {
	var check Check
	err := r.db.WithContext(
		ctx,
	).Where(
		Check{
			ID:        checkID,
			TeamID:    uint64(teamID),
			MonitorID: uint64(monitorID),
		},
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

func (r *RepositoryImpl) GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]Check, error) {
	var checks []Check
	err := r.db.WithContext(
		ctx,
	).Where(
		Check{
			TeamID:    uint64(teamID),
			MonitorID: uint64(monitorID),
		},
	).Order(
		"created_at desc",
	).Scopes(
		clickhouse.Paginated(offset, limit),
	).Find(
		&checks,
	).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &[]Check{}, nil
		}

		return nil, err
	}

	return &checks, nil
}

type AggMetric struct {
	Start      string
	DNS        float64
	TCP        float64
	TLS        float64
	Processing float64
	Transfer   float64
}

func (r *RepositoryImpl) GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint) (*[]AggMetric, error) {
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
		Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw").
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 MONTH) AND NOW()").
		Order("start ASC").
		Find(&metrics).Error

	return &metrics, err
}

func (r *RepositoryImpl) Create(ctx context.Context, check *Check) error {
	return r.db.WithContext(ctx).Create(check).Error
}

type MonitorOverviews struct {
	MonitorID uint
	Start     string
	P99       float32
	P95       float32
	Stats     []float64 `gorm:"type:float"`
}

func (r *RepositoryImpl) GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error) {
	var overviews []MonitorOverviews
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		monitor_id,
		tumbleStart(wndw) as start, 
		quantile(0.99)(timing_total)/1000000 as p99, 
		quantile(0.95)(timing_total)/1000000 as p95,
		groupArray(24)(timing_total/1000000) as stats`).
		Where("team_id = ?", teamID).
		Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw, monitor_id").
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 DAY) AND NOW()").
		Order("start ASC").
		Find(&overviews).Error

	return &overviews, err
}
