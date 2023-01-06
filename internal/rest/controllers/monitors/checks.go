package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorChecksRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorChecksResponse struct {
	TotalCount uint                            `json:"totalCount"`
	Checks     []GetMonitorChecksResponseCheck `json:"checks"`
}

type GetMonitorChecksResponseCheck struct {
	ID         uint64                         `json:"id"`
	StatusCode uint64                         `json:"statusCode"`
	Timing     GetMonitorChecksResponseTiming `json:"timing"`
	TLS        GetMonitorChecksResponseTLS    `json:"tls"`
	CreatedAt  string                         `json:"createdAt"`
}

type GetMonitorChecksResponseTiming struct {
	DNSLookup        time.Duration `json:"dnsLookup"`
	TCPConnection    time.Duration `json:"tcpConnection"`
	TLSHandshake     time.Duration `json:"tlsHandshake"`
	ServerProcessing time.Duration `json:"serverProcessing"`
	ContentTransfer  time.Duration `json:"contentTransfer"`
	Total            time.Duration `json:"total"`
}

type GetMonitorChecksResponseTLS struct {
	Version   string    `json:"version"`
	Cipher    string    `json:"cipher"`
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	NotBefore time.Time `json:"notBefore"`
	NotAfter  time.Time `json:"notAfter"`
}

func (h *Handlers) GetMonitorChecks(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorChecksRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	_, err = h.MonitorService.GetMonitorByIDAndTeamID(ctx.Request().Context(), req.MonitorID, req.TeamID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			ctx.Log.WithError(err).Debug("monitor not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	results, err := h.CheckService.GetMonitorChecksByID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	resp := h.newGetMonitorChecksResponse(results)

	return ctx.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetMonitorChecksResponse(checks *[]check.Check) GetMonitorChecksResponse {
	checkRes := make([]GetMonitorChecksResponseCheck, len(*checks))

	for i, c := range *checks {
		checkRes[i] = GetMonitorChecksResponseCheck{
			ID:         c.ID,
			StatusCode: c.StatusCode,
			Timing: GetMonitorChecksResponseTiming{
				DNSLookup:        c.Timing.DNSLookup,
				TCPConnection:    c.Timing.TCPConnection,
				TLSHandshake:     c.Timing.TLSHandshake,
				ServerProcessing: c.Timing.ServerProcessing,
				ContentTransfer:  c.Timing.ContentTransfer,
				Total:            c.Timing.Total,
			},
			TLS: GetMonitorChecksResponseTLS{
				Version:   c.TLS.Version,
				Cipher:    c.TLS.Cipher,
				Issuer:    c.TLS.Issuer,
				Subject:   c.TLS.Subject,
				NotBefore: c.TLS.NotBefore,
				NotAfter:  c.TLS.NotAfter,
			},
			CreatedAt: c.CreatedAt.String(),
		}
	}

	return GetMonitorChecksResponse{
		TotalCount: 0, // TODO
		Checks:     checkRes,
	}
}
