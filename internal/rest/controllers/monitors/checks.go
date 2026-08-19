package monitors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type ForecasterTimingMetric struct {
	ResponseTime     float64 `json:"response_time"`
	DNSLookup        float64 `json:"dns_lookup"`
	TCPConnection    float64 `json:"tcp_connection"`
	TLSHandshake     float64 `json:"tls_handshake"`
	ServerProcessing float64 `json:"server_processing"`
	ContentTransfer  float64 `json:"content_transfer"`
	CreatedAt        string  `json:"created_at,omitempty"`
}

type ChecksPredictRequest struct {
	MonitorID int                      `json:"monitor_id"`
	Timings   []ForecasterTimingMetric `json:"timings"`
}

type ForecasterPredictResponse struct {
	Anomalies   []bool    `json:"anomalies"`
	Predictions []float64 `json:"predictions"`
	UpperBounds []float64 `json:"upper_bounds"`
	LowerBounds []float64 `json:"lower_bounds"`
}

func getChecksAnomalies(monitorID uint, checks *[]check.Check) ([]bool, error) {
	if len(*checks) == 0 {
		return nil, nil
	}

	timings := make([]ForecasterTimingMetric, len(*checks))
	for i, c := range *checks {
		timings[i] = ForecasterTimingMetric{
			ResponseTime:     float64(c.Timing.Total.Milliseconds()),
			DNSLookup:        float64(c.Timing.DNSLookup.Milliseconds()),
			TCPConnection:    float64(c.Timing.TCPConnection.Milliseconds()),
			TLSHandshake:     float64(c.Timing.TLSHandshake.Milliseconds()),
			ServerProcessing: float64(c.Timing.ServerProcessing.Milliseconds()),
			ContentTransfer:  float64(c.Timing.ContentTransfer.Milliseconds()),
			CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		}
	}

	reqBody, err := json.Marshal(ChecksPredictRequest{
		MonitorID: int(monitorID),
		Timings:   timings,
	})
	if err != nil {
		return nil, err
	}

	forecasterURL := os.Getenv("FORECASTER_API_URL")
	if forecasterURL == "" {
		forecasterURL = "http://forecaster-api:8000"
	}
	url := fmt.Sprintf("%s/predict", forecasterURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		// Fallback to localhost if running outside docker network
		url = "http://localhost:8000/predict"
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecaster api returned status %d", resp.StatusCode)
	}

	var forecasterResp ForecasterPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecasterResp); err != nil {
		return nil, err
	}

	return forecasterResp.Anomalies, nil
}

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
	Location   string                         `json:"location"`
	Timing     GetMonitorChecksResponseTiming `json:"timing"`
	TLS        *GetMonitorChecksResponseTLS   `json:"tls,omitempty"`
	CreatedAt  string                         `json:"createdAt"`
	Anomaly    bool                           `json:"anomaly"`
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

	anomalies, err := getChecksAnomalies(req.MonitorID, results)
	if err != nil {
		c.Log.WithError(err).Warn("failed to fetch anomalies from forecaster")
	}

	resp := h.newGetMonitorChecksResponse(results, anomalies)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetMonitorChecksResponse(checks *[]check.Check, anomalies []bool) GetMonitorChecksResponse {
	checkRes := make([]GetMonitorChecksResponseCheck, len(*checks))

	for i, c := range *checks {
		isAnomaly := false
		if i < len(anomalies) {
			isAnomaly = anomalies[i]
		}
		checkRes[i] = h.newGetMonitorCheckResponse(c, isAnomaly)
	}

	return GetMonitorChecksResponse{
		Checks: checkRes,
	}
}

func (h *Handlers) newGetMonitorCheckResponse(check check.Check, anomaly bool) GetMonitorChecksResponseCheck {
	c := GetMonitorChecksResponseCheck{
		ID:         check.ID,
		StatusCode: check.StatusCode,
		Method:     check.Method,
		URL:        check.URL,
		Location:   check.Location,
		Timing: GetMonitorChecksResponseTiming{
			DNSLookup:        check.Timing.DNSLookup,
			TCPConnection:    check.Timing.TCPConnection,
			TLSHandshake:     check.Timing.TLSHandshake,
			ServerProcessing: check.Timing.ServerProcessing,
			ContentTransfer:  check.Timing.ContentTransfer,
			Total:            check.Timing.Total,
		},
		CreatedAt: check.CreatedAt.Format(time.UnixDate),
		Anomaly:   anomaly,
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

	singleCheckList := []check.Check{*result}
	anomalies, err := getChecksAnomalies(req.MonitorID, &singleCheckList)
	isAnomaly := false
	if err == nil && len(anomalies) > 0 {
		isAnomaly = anomalies[0]
	}

	resp := h.newGetMonitorCheckResponse(*result, isAnomaly)

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

	resp := h.newGetMonitorChecksResponse(result, nil)

	return c.JSON(http.StatusOK, resp)
}
