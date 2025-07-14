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
	GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*MonitorStats, error)
	GetMonitorOverviewStatsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviewStats, error)
	GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error)
	GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]MonitorUptime, error)
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

type MonitorStats struct {
	UptimePercentage    float32
	AverageResponseTime float32
}

func (r *RepositoryImpl) GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*MonitorStats, error) {
	var stats MonitorStats
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		(count(status_code <= 400) / count(status_code)) * 100 as uptime_percentage,
		avg(timing_total/1000000) as average_response_time`).
		Where("monitor_id = ?", monitorID).
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 DAY) AND NOW()").
		Scan(&stats).Error

	return &stats, err
}

type MonitorOverviews struct {
	MonitorID uint
	Latest    string
	P99       float32
	P95       float32
	Stats     []float64 `gorm:"type:float"`
}

type MonitorOverviewStats struct {
	MonitorID uint
	Stats     []float64 `gorm:"type:float"`
}

func (r *RepositoryImpl) GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error) {
	var overviews []MonitorOverviews
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		monitor_id,
		max(created_at) as latest, 
		quantile(0.99)(timing_total)/1000000 as p99, 
		quantile(0.95)(timing_total)/1000000 as p95`).
		Where("team_id = ?", teamID).
		Group("monitor_id").
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 DAY) AND NOW()").
		Order("latest ASC").
		Find(&overviews).Error

	return &overviews, err
}
func (r *RepositoryImpl) GetMonitorOverviewStatsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviewStats, error) {
	var overviews []MonitorOverviewStats
	err := r.db.WithContext(
		ctx,
	).Raw(`
		SELECT
		monitor_id,
		groupArray(timing) as stats
		FROM
		(
			SELECT
				monitor_id,
				avg(timing_total) / 1000000 AS timing,
				tumbleStart(wndw) AS start
			FROM checks
			WHERE (team_id = 1) AND ((created_at >= (NOW() - toIntervalDay(1))) AND (created_at <= NOW()))
			GROUP BY
				monitor_id,
				tumble(toDateTime(created_at), toIntervalHour('1')) AS wndw
			ORDER BY start ASC
		)
		GROUP BY monitor_id`).
		Find(&overviews).Error

	return &overviews, err
}

func (r *RepositoryImpl) GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error) {
	var checks []Check
	err := r.db.WithContext(
		ctx,
	).Where(
		Check{
			MonitorID: uint64(monitorID),
		},
	).Where(assertions[0]).Order(
		"created_at desc",
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

type MonitorUptime struct {
	MonitorID        uint
	Url              string
	UptimePercentage float32
	Date             string
}

func (r *RepositoryImpl) GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]MonitorUptime, error) {
	var uptime []MonitorUptime
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		monitor_id, 
		url,
		count(status_code <= 400) / count(status_code) * 100 as uptime_percentage, 
		avg(timing_total/1000000) as average_response_time, 
		toMonth(created_at) as date`).
		Where("team_id = ?", teamID).
		// Where("created_at BETWEEN ? AND ?", start, end).
		Group("monitor_id, url, date").
		Order("date ASC").
		Find(&uptime).Error

	if err != nil {
		return nil, err
	}

	return &uptime, nil
}
