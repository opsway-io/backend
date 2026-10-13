package statuspages_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/controllers/statuspages"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockMaintenanceService struct{}

func (m *mockMaintenanceService) GetByID(ctx context.Context, id uint) (*entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetActive(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetActiveByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetUpcomingByMonitorIDs(ctx context.Context, now time.Time, monitorIDs []uint) ([]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetAllByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Maintenance, error) {
	now := time.Now()
	desc := "Database upgrade"
	return []entities.Maintenance{
		{
			ID:          1,
			Title:       "Scheduled DB Upgrade",
			Description: &desc,
			Settings: entities.MaintenanceSettings{
				StartAt: now,
				EndAt:   now.Add(2 * time.Hour),
			},
		},
	}, nil
}
func (m *mockMaintenanceService) GetUnnotified(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetUnreminded(ctx context.Context, now time.Time, reminderWindow time.Duration) (*[]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) GetUnconcluded(ctx context.Context, now time.Time) (*[]entities.Maintenance, error) {
	return nil, nil
}
func (m *mockMaintenanceService) Create(ctx context.Context, maintenance *entities.Maintenance) error {
	return nil
}
func (m *mockMaintenanceService) Update(ctx context.Context, maintenance *entities.Maintenance) error {
	return nil
}
func (m *mockMaintenanceService) Delete(ctx context.Context, maintenance *entities.Maintenance) error {
	return nil
}

type mockCheckService struct{}

func (m *mockCheckService) Create(ctx context.Context, c *check.Check) error {
	return nil
}
func (m *mockCheckService) GetByTeamIDAndMonitorIDPaginated(ctx context.Context, teamID, monitorID uint, offset, limit *int) (*[]check.Check, error) {
	return nil, nil
}
func (m *mockCheckService) GetByTeamIDAndMonitorIDAndCheckID(ctx context.Context, teamID uint, monitorID uint, checkID uuid.UUID) (*check.Check, error) {
	return nil, nil
}
func (m *mockCheckService) GetLatestByMonitorID(ctx context.Context, monitorID uint) (*check.Check, error) {
	return nil, nil
}
func (m *mockCheckService) GetMonitorMetricsByMonitorID(ctx context.Context, monitorID uint, start *string, end *string) (*[]check.AggMetric, error) {
	return nil, nil
}
func (m *mockCheckService) GetMonitorStatsByMonitorID(ctx context.Context, monitorID uint) (*check.MonitorStats, error) {
	return nil, nil
}
func (m *mockCheckService) GetMonitorOverviewsByTeamID(ctx context.Context, teamID uint) (*[]check.MonitorOverviews, error) {
	return nil, nil
}
func (m *mockCheckService) GetMonitorIDAndAssertions(ctx context.Context, monitorID uint, assertions []string) (*[]check.Check, error) {
	return nil, nil
}
func (m *mockCheckService) GetByTeamIDMonitorsUptime(ctx context.Context, teamID uint, start, end string) (*[]check.MonitorUptime, error) {
	return nil, nil
}
func (m *mockCheckService) GetByTeamIDMonitorsPerformance(ctx context.Context, teamID uint, start, end string) (*[]check.MonitorPerformance, error) {
	return nil, nil
}
func (m *mockCheckService) GetMonitorUptimesByMonitorIDs(ctx context.Context, monitorIDs []uint) (map[uint]check.MonitorUptimeResult, error) {
	return map[uint]check.MonitorUptimeResult{
		10: {
			UptimePercentage: 99.95,
			DailyUptimes:     []float32{100, 100, 99.8, 100},
		},
	}, nil
}

func TestCalendarAndMetrics(t *testing.T) {
	e := echo.New()

	sp := &entities.StatusPage{
		ID:     1,
		Name:   "Acme Status",
		Domain: "acme",
		Monitors: []entities.Monitor{
			{
				ID:    10,
				Name:  "API",
				State: entities.MonitorStateActive,
			},
		},
		Groups: []entities.StatusPageGroup{
			{
				ID:        1,
				Name:      "API Service",
				Container: "US-East",
				Monitors: []entities.Monitor{
					{
						ID:    10,
						Name:  "API",
						State: entities.MonitorStateActive,
					},
				},
			},
		},
	}

	spSvc := &mockStatusPageService{sp: sp}
	mntSvc := &mockMaintenanceService{}
	chkSvc := &mockCheckService{}
	logger := logrus.NewEntry(logrus.StandardLogger())

	e.Validator = helpers.NewValidator()
	statuspages.RegisterPublic(e.Group("/v1"), logger, "https://status.opsway.dev", spSvc, nil, mntSvc, chkSvc, nil)

	t.Run("GetMaintenanceCalendar_ICS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme/calendar.ics", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/calendar")
		assert.Contains(t, rec.Body.String(), "BEGIN:VCALENDAR")
		assert.Contains(t, rec.Body.String(), "Scheduled DB Upgrade")
		assert.Contains(t, rec.Body.String(), "END:VCALENDAR")
	})

	t.Run("GetPublicMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme/metrics", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, rec.Body.String(), `"uptimePercentage":99.95`)
		assert.Contains(t, rec.Body.String(), `"name":"API"`)
	})

	t.Run("GetPublicStatusPage_ContainerAndStatusRollup", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"container":"US-East"`)
		assert.Contains(t, rec.Body.String(), `"status":"OPERATIONAL"`)
	})
}
