package monitors

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
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
	Name string           `json:"name"`
	Data []MonitorMetrics `json:"timing"`
}
type MonitorMetrics struct {
	Start  string        `json:"start"`
	Timing time.Duration `json:"timing"`
}

func (h *Handlers) GetMonitorMetrics(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorMetricsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	metrics, err := h.CheckService.GetMonitorMetricsByMonitorID(
		ctx,
		req.MonitorID,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	metrics_list := []string{"DNS", "TCP", "TLS", "Processing", "Transfer"}
	metricResp := make([]GetMonitorMetricsResponseMetric, len(metrics_list))

	for i, metric := range metrics_list {
		metricResp[i] = GetMonitorMetricsResponseMetric{Name: metric, Data: []MonitorMetrics{}}
	}

	for _, c := range *metrics {
		metricResp[0].Data = append(metricResp[0].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.DNS)})
		metricResp[1].Data = append(metricResp[1].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TCP)})
		metricResp[2].Data = append(metricResp[2].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TLS)})
		metricResp[3].Data = append(metricResp[3].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Processing)})
		metricResp[4].Data = append(metricResp[4].Data, MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Transfer)})
	}

	return c.JSON(http.StatusOK, GetMonitorMetricsRespone{Metrics: metricResp})
}
