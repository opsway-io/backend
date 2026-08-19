package metrics

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"gorm.io/gorm"
)

type Handlers struct {
	CheckService check.Service
	DB           *gorm.DB
}

type GetPrometheusMetricsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetPrometheusMetrics(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetPrometheusMetricsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetPrometheusMetricsRequest")
		return echo.ErrBadRequest
	}

	// Verify teamId matches the one in context from ApiKey
	ctxTeamID := c.Get("team_id")
	if ctxTeamID != req.TeamID {
		return echo.ErrForbidden
	}

	ctx := c.Request().Context()
	
	// Fetch all monitors for the team
	type monitorRow struct {
		ID   uint
		Name string
	}
	var monitors []monitorRow
	if err := h.DB.WithContext(ctx).Table("monitors").Where("team_id = ?", req.TeamID).Select("id, name").Find(&monitors).Error; err != nil {
		c.Log.WithError(err).Error("failed to fetch monitors for metrics")
		return echo.ErrInternalServerError
	}

	// Build prometheus response string
	var sb strings.Builder
	sb.WriteString("# HELP opsway_monitor_uptime_ratio Uptime ratio of the monitor (0.0 to 1.0)\n")
	sb.WriteString("# TYPE opsway_monitor_uptime_ratio gauge\n")

	for _, m := range monitors {
		stats, err := h.CheckService.GetMonitorStatsByMonitorID(ctx, m.ID)
		if err != nil {
			continue
		}
		
		uptimeRatio := 0.0
		if stats != nil {
			uptimeRatio = float64(stats.UptimePercentage) / 100.0
		}
		
		sb.WriteString(fmt.Sprintf("opsway_monitor_uptime_ratio{monitor_id=\"%d\",monitor_name=\"%s\"} %f\n", m.ID, m.Name, uptimeRatio))
	}

	return c.String(http.StatusOK, sb.String())
}
