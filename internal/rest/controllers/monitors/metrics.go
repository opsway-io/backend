package monitors

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

/*
Something like this for metrics response:

{
    "uptime_percentage": 99.9,
    "average_response_time_ms": 20,
    "last_check_timestamp": 1672428631,
    "tls": {
        "version": "1.2",
        "cipher": "ECDHE-RSA-AES128-GCM-SHA256",
        "certificate": {
            "issuer": "Lets encrypt",
            "subject": "example.com",
            "not_before": 1672428631,
            "not_after": 1672428631
        }
    },
    "timing": [
        {
            "dns_lookup": 20,
            "tcp_connection": 20,
            "tls_handshake": 20,
            "server_processing": 20,
            "content_transfer": 20,
            "total": 20
        }
		...
    ]
}
*/

type GetMonitorMetricsRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMonitorMetrics(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorMetricsRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	results, err := h.CheckService.GetMonitorMetricsByID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	// TODO CREATE RESONSE MAPPING

	return ctx.JSON(http.StatusOK, results)
}
