package check

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("probe result not found")

type Repository interface {
	Create(ctx context.Context, maintenance *Check) error
	GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*Check, error)
	GetLatestByMonitorID(ctx context.Context, monitorID uint) (*Check, error)
	GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]Check, error)
	GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint, start *string, end *string) (*[]AggMetric, error)
	GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviews, error)
	GetMonitorOverviewStatsByTeamID(ctx context.Context, teamID uint) (*[]MonitorOverviewStats, error)
	GetMonitorUptimesByMonitorIDs(ctx context.Context, monitorIDs []uint) (map[uint]MonitorUptimeResult, error)
	GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error)
	GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]MonitorUptime, error)
	GetByTeamIDMonitorsPerformance(ctx context.Context, teamID uint, start, end string) (*[]MonitorPerformance, error)
	GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*MonitorStats, error)
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

func (r *RepositoryImpl) GetLatestByMonitorID(ctx context.Context, monitorID uint) (*Check, error) {
	var check Check
	err := r.db.WithContext(
		ctx,
	).Where(
		Check{
			MonitorID: uint64(monitorID),
		},
	).Order(
		"created_at desc",
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
	Total      float64
}

func (r *RepositoryImpl) GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint, start *string, end *string) (*[]AggMetric, error) {
	var metrics []AggMetric

	query := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		tumbleStart(wndw) as start, 
		avg(timing_dns_lookup)/1000000 as dns, 
		avg(timing_tcp_connection)/1000000 as tcp,
		avg(timing_tls_handshake)/1000000 as tls,
		avg(timing_server_processing)/1000000 as processing,
		avg(timing_content_transfer)/1000000 as transfer,
		avg(timing_total)/1000000 as total`).
		Where("monitor_id = ?", monitorID)

	if start != nil && end != nil {
		// Calculate the difference in hours
		startTime, err1 := time.Parse(time.RFC3339, *start)
		endTime, err2 := time.Parse(time.RFC3339, *end)
		if err1 == nil && err2 == nil {
			duration := endTime.Sub(startTime)
			if duration.Hours() <= 24 {
				// Less than a day, group by 5 minutes
				query = query.Group("tumble(toDateTime(created_at), INTERVAL 5 MINUTE) as wndw")
			} else if duration.Hours() <= 72 {
				// Less than 3 days, group by 15 minutes
				query = query.Group("tumble(toDateTime(created_at), INTERVAL 15 MINUTE) as wndw")
			} else {
				// Otherwise group by 1 hour
				query = query.Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw")
			}
			query = query.Where("created_at BETWEEN ? AND ?", startTime, endTime)
		} else {
			query = query.Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw")
			query = query.Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 MONTH) AND NOW()")
		}
	} else {
		query = query.Group("tumble(toDateTime(created_at), INTERVAL 1 HOUR) as wndw")
		query = query.Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 MONTH) AND NOW()")
	}

	err := query.Order("start ASC").
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
		if(count() = 0, 0, (countIf(status_code > 0 AND status_code < 400) / count()) * 100) as uptime_percentage,
		if(count() = 0, 0, avg(timing_total/1000000)) as average_response_time`).
		Where("monitor_id = ?", monitorID).
		Where("created_at BETWEEN DATE_SUB(NOW(), INTERVAL 1 DAY) AND NOW()").
		Scan(&stats).Error

	return &stats, err
}

type MonitorOverviews struct {
	MonitorID           uint
	Latest              string
	UptimePercentage    float32
	AverageResponseTime float32
	P99                 float32
	P95                 float32
	Stats               []float64 `gorm:"type:float"`
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
		(countIf(status_code > 0 AND status_code < 400) / count()) * 100 as uptime_percentage,
		avg(timing_total/1000000) as average_response_time,
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
			WHERE (team_id = ?) AND ((created_at >= (NOW() - toIntervalDay(1))) AND (created_at <= NOW()))
			GROUP BY
				monitor_id,
				tumble(toDateTime(created_at), toIntervalHour('1')) AS wndw
			ORDER BY start ASC
		)
		GROUP BY monitor_id`, teamID).
		Find(&overviews).Error

	return &overviews, err
}

type MonitorUptimeResult struct {
	MonitorID        uint
	UptimePercentage float32
	DailyUptimes     []float32 `gorm:"type:float"`
}

func (r *RepositoryImpl) GetMonitorUptimesByMonitorIDs(ctx context.Context, monitorIDs []uint) (map[uint]MonitorUptimeResult, error) {
	var results []MonitorUptimeResult

	if len(monitorIDs) == 0 {
		return make(map[uint]MonitorUptimeResult), nil
	}

	// Calculate overall 90-day uptime and daily uptimes using ClickHouse array aggregation
	var queryResults []struct {
		MonitorID        uint
		UptimePercentage float32
		DailyUptimes     []float32   `gorm:"type:float"`
		DailyDates       []time.Time `gorm:"type:datetime"`
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			m.monitor_id as monitor_id,
			if(sum(m.total_count) = 0, 0, (sum(m.up_count) / sum(m.total_count)) * 100) as uptime_percentage,
			groupArray(m.daily_uptime) as daily_uptimes,
			groupArray(m.day) as daily_dates
		FROM (
			SELECT 
				monitor_id,
				toDate(created_at) as day,
				count() as total_count,
				countIf(status_code > 0 AND status_code < 400) as up_count,
				if(count() = 0, 0, (countIf(status_code > 0 AND status_code < 400) / count()) * 100) as daily_uptime
			FROM checks
			WHERE monitor_id IN ? AND created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY)
			GROUP BY monitor_id, day
			ORDER BY monitor_id, day ASC
		) m
		GROUP BY m.monitor_id
	`, monitorIDs).Scan(&queryResults).Error

	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	// Truncate to day
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	for _, qr := range queryResults {
		// Initialize exactly 90 days with -1
		paddedUptimes := make([]float32, 90)
		for i := 0; i < 90; i++ {
			paddedUptimes[i] = -1 // No data
		}

		// Map existing data to the correct index
		for i, date := range qr.DailyDates {
			// Calculate days ago (0 = today, 89 = 89 days ago)
			daysAgo := int(today.Sub(date.UTC()).Hours() / 24)
			if daysAgo >= 0 && daysAgo < 90 {
				// We want index 0 to be oldest (89 days ago) and index 89 to be today (0 days ago)
				idx := 89 - daysAgo
				paddedUptimes[idx] = qr.DailyUptimes[i]
			}
		}

		results = append(results, MonitorUptimeResult{
			MonitorID:        qr.MonitorID,
			UptimePercentage: qr.UptimePercentage,
			DailyUptimes:     paddedUptimes,
		})
	}

	uptimes := make(map[uint]MonitorUptimeResult)
	for _, r := range results {
		uptimes[r.MonitorID] = r
	}
	return uptimes, nil
}

func (r *RepositoryImpl) GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]Check, error) {
	var checks []Check
	err := r.db.WithContext(
		ctx,
	).Where(
		Check{
			MonitorID: uint64(monitorID),
		},
	).Where("assertion = ?", assertions[0]).Order(
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
		toString(toMonth(created_at)) as date`).
		Where("team_id = ?", teamID).
		Where("created_at BETWEEN parseDateTimeBestEffort(?) AND parseDateTimeBestEffort(?)", start, end).
		Group("monitor_id, url, date").
		Order("date ASC").
		Find(&uptime).Error

	if err != nil {
		return nil, err
	}

	return &uptime, nil
}

type MonitorPerformance struct {
	MonitorID           uint
	AverageResponseTime float32
	P99                 float32
	P95                 float32
}

func (r *RepositoryImpl) GetByTeamIDMonitorsPerformance(ctx context.Context, teamID uint, start, end string) (*[]MonitorPerformance, error) {
	var performance []MonitorPerformance
	err := r.db.WithContext(
		ctx,
	).Table("checks").Select(`
		monitor_id, 
		avg(timing_total/1000000) as average_response_time, 
		quantile(0.99)(timing_total)/1000000 as p99, 
		quantile(0.95)(timing_total)/1000000 as p95`).
		Where("team_id = ?", teamID).
		Where("created_at BETWEEN parseDateTimeBestEffort(?) AND parseDateTimeBestEffort(?)", start, end).
		Group("monitor_id").
		Find(&performance).Error

	if err != nil {
		return nil, err
	}

	return &performance, nil
}
