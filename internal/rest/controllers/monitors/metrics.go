package monitors

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorMetricsRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorMetricsRespone struct {
	Metrics []GetMonitorMetricsResponseMetric `json:"metrics"`
}

type GetMonitorMetricsResponseMetric struct {
	Start  string        `json:"start"`
	Timing time.Duration `json:"timing"`
}

func (h *Handlers) GetMonitorMetrics(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorMetricsRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	metrics, err := h.CheckService.GetMonitorMetricsByID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	metricResp := make([]GetMonitorMetricsResponseMetric, len(*metrics))

	for i, c := range *metrics {
		metricResp[i] = h.newGetMonitorMetricResponse(c)
	}

	return ctx.JSON(http.StatusOK, GetMonitorMetricsRespone{Metrics: metricResp})
}

func (h *Handlers) newGetMonitorMetricResponse(metric check.AggMetric) GetMonitorMetricsResponseMetric {
	return GetMonitorMetricsResponseMetric{
		Start:  metric.Start,
		Timing: time.Duration(metric.Timing),
	}
}
