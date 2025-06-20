package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorChecksRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
	Offset    *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit     *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetMonitorChecksResponse struct {
	Checks []GetMonitorChecksResponseCheck `json:"checks"`
}

type GetMonitorChecksResponseCheck struct {
	ID         uuid.UUID                      `json:"id"`
	StatusCode uint64                         `json:"statusCode"`
	Method     string                         `json:"method"`
	URL        string                         `json:"url"`
	Timing     GetMonitorChecksResponseTiming `json:"timing"`
	TLS        *GetMonitorChecksResponseTLS   `json:"tls,omitempty"`
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

func (h *Handlers) GetMonitorChecks(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorChecksRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	results, err := h.CheckService.GetByTeamIDAndMonitorIDPaginated(
		ctx,
		req.TeamID,
		req.MonitorID,
		req.Offset,
		req.Limit,
	)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	resp := h.newGetMonitorChecksResponse(results)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetMonitorChecksResponse(checks *[]check.Check) GetMonitorChecksResponse {
	checkRes := make([]GetMonitorChecksResponseCheck, len(*checks))

	for i, c := range *checks {
		checkRes[i] = h.newGetMonitorCheckResponse(c)
	}

	return GetMonitorChecksResponse{
		Checks: checkRes,
	}
}

func (h *Handlers) newGetMonitorCheckResponse(check check.Check) GetMonitorChecksResponseCheck {
	c := GetMonitorChecksResponseCheck{
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
		CreatedAt: check.CreatedAt.Format(time.UnixDate),
	}

	if check.TLS != nil {
		c.TLS = &GetMonitorChecksResponseTLS{
			Version:   check.TLS.Version,
			Cipher:    check.TLS.Cipher,
			Issuer:    check.TLS.Issuer,
			Subject:   check.TLS.Subject,
			NotBefore: check.TLS.NotBefore,
			NotAfter:  check.TLS.NotAfter,
		}
	}

	return c
}

type GetMonitorCheckRequest struct {
	TeamID    uint      `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint      `param:"monitorId" validate:"required,numeric,gte=0"`
	CheckID   uuid.UUID `param:"checkId" validate:"required"`
	Offset    *int      `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit     *int      `query:"limit" validate:"omitempty,numeric,gte=0"`
}

type GetMonitorCheckResponse struct {
	GetMonitorChecksResponseCheck
}

func (h *Handlers) GetMonitorCheck(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorCheckRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	result, err := h.CheckService.GetByTeamIDAndMonitorIDAndCheckID(
		ctx,
		req.TeamID,
		req.MonitorID,
		req.CheckID,
	)
	if err != nil {
		if errors.Is(err, check.ErrNotFound) {
			c.Log.WithError(err).Debug("check not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get monitor check")

		return echo.ErrInternalServerError
	}

	resp := h.newGetMonitorCheckResponse(*result)

	return c.JSON(http.StatusOK, resp)
}

type GeFailedMonitorCheckRequest struct {
	TeamID             uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID          uint `param:"monitorId" validate:"required,numeric,gte=0"`
	MonitorAssertionID uint `param:"monitorAssertionId" validate:"required,numeric,gte=0"`
	Offset             *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit              *int `query:"limit" validate:"omitempty,numeric,gte=0"`
}

type GetFailedMonitorCheckResponse struct {
	GetMonitorChecksResponseCheck
}

func (h *Handlers) GetFailedMonitorChecks(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GeFailedMonitorCheckRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	monitorAssertion, err := h.MonitorService.GetMonitorAssertionByID(ctx, req.MonitorAssertionID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitorAssertion")
		return echo.ErrInternalServerError
	}

	// map monitorassertion to filter

	result, err := h.CheckService.GetMonitorIDAndAssertions(
		ctx,
		monitorAssertion.MonitorID,
		[]string{"status_code == 200"},
	)
	if err != nil {
		if errors.Is(err, check.ErrNotFound) {
			c.Log.WithError(err).Debug("check not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get monitor check")

		return echo.ErrInternalServerError
	}

	resp := h.newGetMonitorChecksResponse(result)

	return c.JSON(http.StatusOK, resp)
}
