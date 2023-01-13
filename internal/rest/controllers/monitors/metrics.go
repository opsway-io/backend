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

	metricMap := map[string][]MonitorMetrics{}

	for _, c := range *metrics {
		metricMap["dns"] = append(metricMap["dns"], MonitorMetrics{Start: c.Start, Timing: time.Duration(c.DNS)})
		metricMap["tcp"] = append(metricMap["tcp"], MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TCP)})
		metricMap["tls"] = append(metricMap["tls"], MonitorMetrics{Start: c.Start, Timing: time.Duration(c.TLS)})
		metricMap["processing"] = append(metricMap["processing"], MonitorMetrics{Start: c.Start, Timing: time.Duration(c.DNS)})
		metricMap["transfer"] = append(metricMap["transfer"], MonitorMetrics{Start: c.Start, Timing: time.Duration(c.Transfer)})
	}

	metricResp := make([]GetMonitorMetricsResponseMetric, len(metricMap))

	i := 0
	for name, data := range metricMap {
		metricResp[i] = GetMonitorMetricsResponseMetric{Name: name, Data: data}
		i += 1
	}

	return ctx.JSON(http.StatusOK, GetMonitorMetricsRespone{Metrics: metricResp})
}
