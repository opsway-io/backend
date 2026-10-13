package report

import (
	"context"
	"errors"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("report not found")

type Repository interface {
	GetReportsByTeamID(ctx context.Context, teamID uint) (*[]entities.Report, error)
	GetByTeamIDAndID(ctx context.Context, teamID, reportID uint) (*entities.Report, error)
	Create(ctx context.Context, rep *entities.Report) error
	Delete(ctx context.Context, teamID, reportID uint) error

	GetUptime(ctx context.Context, teamID uint, start, end time.Time) (*[]check.MonitorUptime, error)
	GetPerformance(ctx context.Context, teamID uint, start, end time.Time) (*[]check.MonitorPerformance, error)
	GetIncidentCounts(ctx context.Context, teamID uint, start, end time.Time) (*[]entities.ReportMonitorIncident, error)
}

type RepositoryImpl struct {
	db *gorm.DB
	ch *gorm.DB
}

// NewRepository creates a report repository. db is the Postgres client, ch the ClickHouse client
// (may be nil, in which case check based report sections are empty).
func NewRepository(db *gorm.DB, ch *gorm.DB) Repository {
	return &RepositoryImpl{
		db: db,
		ch: ch,
	}
}

func (r *RepositoryImpl) GetReportsByTeamID(ctx context.Context, teamID uint) (*[]entities.Report, error) {
	var reports []entities.Report
	if err := r.db.WithContext(ctx).
		Omit("report").
		Where(entities.Report{TeamID: teamID}).
		Order("created_at desc").
		Find(&reports).Error; err != nil {
		return nil, err
	}

	return &reports, nil
}

func (r *RepositoryImpl) GetByTeamIDAndID(ctx context.Context, teamID, reportID uint) (*entities.Report, error) {
	var rep entities.Report
	if err := r.db.WithContext(ctx).
		Where(entities.Report{ID: reportID, TeamID: teamID}).
		First(&rep).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &rep, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, rep *entities.Report) error {
	return r.db.WithContext(ctx).Create(rep).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, teamID, reportID uint) error {
	res := r.db.WithContext(ctx).
		Where(entities.Report{ID: reportID, TeamID: teamID}).
		Delete(&entities.Report{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RepositoryImpl) GetUptime(ctx context.Context, teamID uint, start, end time.Time) (*[]check.MonitorUptime, error) {
	uptime := []check.MonitorUptime{}
	if r.ch == nil {
		return &uptime, nil
	}

	err := r.ch.WithContext(ctx).Raw(`
		SELECT
			monitor_id,
			any(url) AS url,
			countIf(status_code < 400) / count() * 100 AS uptime_percentage,
			toString(toDate(created_at)) AS date
		FROM checks
		WHERE team_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY monitor_id, date
		ORDER BY date ASC, monitor_id ASC`,
		teamID, start, end,
	).Scan(&uptime).Error
	if err != nil {
		return nil, err
	}

	return &uptime, nil
}

func (r *RepositoryImpl) GetPerformance(ctx context.Context, teamID uint, start, end time.Time) (*[]check.MonitorPerformance, error) {
	perf := []check.MonitorPerformance{}
	if r.ch == nil {
		return &perf, nil
	}

	err := r.ch.WithContext(ctx).Raw(`
		SELECT
			monitor_id,
			avg(timing_total) / 1000000 AS average_response_time,
			quantile(0.99)(timing_total) / 1000000 AS p99,
			quantile(0.95)(timing_total) / 1000000 AS p95
		FROM checks
		WHERE team_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY monitor_id
		ORDER BY monitor_id ASC`,
		teamID, start, end,
	).Scan(&perf).Error
	if err != nil {
		return nil, err
	}

	return &perf, nil
}

func (r *RepositoryImpl) GetIncidentCounts(ctx context.Context, teamID uint, start, end time.Time) (*[]entities.ReportMonitorIncident, error) {
	counts := []entities.ReportMonitorIncident{}
	err := r.db.WithContext(ctx).
		Model(&entities.Incident{}).
		Select("monitor_id, count(*) AS count").
		Where("team_id = ? AND created_at BETWEEN ? AND ?", teamID, start, end).
		Group("monitor_id").
		Order("monitor_id ASC").
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}

	return &counts, nil
}
