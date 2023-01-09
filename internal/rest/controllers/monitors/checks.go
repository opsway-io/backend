package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
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
	ID         uuid.UUID                      `json:"id"`
	StatusCode uint64                         `json:"statusCode"`
	Method     string                         `json:"method"`
	URL        string                         `json:"url"`
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

			return echo.ErrForbidden
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
		checkRes[i] = h.newGetMonitorCheckResponse(c)
	}

	return GetMonitorChecksResponse{
		TotalCount: 0, // TODO
		Checks:     checkRes,
	}
}

func (h *Handlers) newGetMonitorCheckResponse(check check.Check) GetMonitorChecksResponseCheck {
	return GetMonitorChecksResponseCheck{
		ID:         check.ID,
		StatusCode: check.StatusCode,
		Method:     check.Method,
		URL:        check.URL,
		Timing: GetMonitorChecksResponseTiming{
			DNSLookup:        check.Timing.DNSLookup,
			TCPConnection:    check.Timing.TCPConnection,
			TLSHandshake:     check.Timing.TLSHandshake,
			ServerProcessing: check.Timing.ServerProcessing,
			ContentTransfer:  check.Timing.ContentTransfer,
			Total:            check.Timing.Total,
		},
		TLS: GetMonitorChecksResponseTLS{
			Version:   check.TLS.Version,
			Cipher:    check.TLS.Cipher,
			Issuer:    check.TLS.Issuer,
			Subject:   check.TLS.Subject,
			NotBefore: check.TLS.NotBefore,
			NotAfter:  check.TLS.NotAfter,
		},
		CreatedAt: check.CreatedAt.Format(time.UnixDate),
	}
}

type GetMonitorCheckRequest struct {
	TeamID    uint      `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint      `param:"monitorId" validate:"required,numeric,gte=0"`
	CheckID   uuid.UUID `param:"checkId" validate:"required"`
}

type GetMonitorCheckResponse struct {
	GetMonitorChecksResponseCheck
}

func (h *Handlers) GetMonitorCheck(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorCheckRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	_, err = h.MonitorService.GetMonitorByIDAndTeamID(ctx.Request().Context(), req.MonitorID, req.TeamID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			ctx.Log.WithError(err).Debug("monitor not found")

			return echo.ErrForbidden
		}

		ctx.Log.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	result, err := h.CheckService.GetMonitorCheckByIDAndMonitorID(ctx.Request().Context(), req.MonitorID, req.CheckID)
	if err != nil {
		if errors.Is(err, check.ErrNotFound) {
			ctx.Log.WithError(err).Debug("check not found")

			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to get monitor check")

		return echo.ErrInternalServerError
	}

	resp := h.newGetMonitorCheckResponse(*result)

	return ctx.JSON(http.StatusOK, resp)
}
