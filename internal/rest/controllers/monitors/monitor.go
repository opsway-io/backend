package monitors

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

/*
	Handlers
*/

type GetMonitorsRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"omitempty"`
}

type GetMonitorsResponse struct {
	Monitors   []GetMonitorsResponseMonitor `json:"monitors"`
	TotalCount int                          `json:"totalCount"`
}

type GetMonitorsResponseMonitor struct {
	Monitor
	Stats GetMonitorsResponseMonitorStats `json:"stats"`
}

type GetMonitorsResponseMonitorStats struct {
	UptimePercentage     float64   `json:"uptimePercentage"`
	AverageResponseTimes []float64 `json:"averageResponseTimes"`
	P99                  uint      `json:"p99"`
	P95                  uint      `json:"p95"`
}

func (h *Handlers) GetMonitors(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetMonitorsAndSettingsByTeamID(c.Request().Context(), req.TeamID, req.Offset, req.Limit, req.Query)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	monitorStats, err := h.CheckService.GetMonitorOverviewsByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor overviews")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetMonitorsResponse(monitors, monitorStats))
}

func newGetMonitorsResponse(monitors *[]monitor.MonitorWithTotalCount, stats *[]check.MonitorOverviews) *GetMonitorsResponse {
	if monitors == nil {
		monitors = &[]monitor.MonitorWithTotalCount{}
	}

	res := make([]GetMonitorsResponseMonitor, len(*monitors))

	monitorStatsMap := map[uint]check.MonitorOverviews{}
	if stats != nil {
		for _, m := range *stats {
			monitorStatsMap[m.MonitorID] = m
		}
	}

	for i := range *monitors {
		m := &(*monitors)[i]

		res[i] = GetMonitorsResponseMonitor{
			Monitor: newMonitorResponse(&m.Monitor),
			Stats: GetMonitorsResponseMonitorStats{
				AverageResponseTimes: []float64{},
			},
		}

		if stat, ok := monitorStatsMap[m.ID]; ok {
			res[i].Stats = GetMonitorsResponseMonitorStats{
				UptimePercentage:     0, // TODO: Implement
				AverageResponseTimes: stat.Stats,
				P99:                  uint(stat.P99),
				P95:                  uint(stat.P95),
			}
		}
	}

	totalCount := 0
	if len(*monitors) > 0 {
		totalCount = (*monitors)[0].TotalCount
	}

	return &GetMonitorsResponse{
		Monitors:   res,
		TotalCount: totalCount,
	}
}

type GetMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	Monitor
	Stats GetMonitorResponseStats `json:"stats"`
}

type GetMonitorResponseStats struct {
	UptimePercentage    float64 `json:"uptimePercentage"`
	AverageResponseTime float64 `json:"averageResponseTime"`
}

func (h *Handlers) GetMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}
	m, err := h.MonitorService.GetMonitorAndSettingsByTeamIDAndID(c.Request().Context(), req.TeamID, req.MonitorID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	stats, err := h.CheckService.GetMonitorStatsByMonitorID(c.Request().Context(), m.ID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor stats")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetMonitorResponse(m, stats))
}

func newGetMonitorResponse(m *entities.Monitor, stats *check.MonitorStats) *GetMonitorResponse {
	resp := GetMonitorResponse{
		Monitor: newMonitorResponse(m),
	}

	if stats != nil {
		resp.Stats = GetMonitorResponseStats{
			UptimePercentage:    finiteOrZero(stats.UptimePercentage),    // TODO: Implement
			AverageResponseTime: finiteOrZero(stats.AverageResponseTime), // TODO: Implement
		}
	}

	return &resp
}

// finiteOrZero guards against NaN/Inf from aggregates over zero checks, which
// encoding/json cannot serialize.
func finiteOrZero(v float32) float64 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}

	return f
}

type DeleteMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteMonitorRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if err := h.MonitorService.Delete(
		ctx,
		req.TeamID,
		req.MonitorID,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostMonitorRequest struct {
	TeamID   uint            `param:"teamId" validate:"required,numeric,gte=0"`
	Name     string          `json:"name" validate:"required,max=255"`
	Settings MonitorSettings `json:"settings"`
	Steps    []MonitorStep   `json:"steps" validate:"required,min=1,dive"`
}

func (h *Handlers) PostMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorRequest")

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	m, err := newMonitorEntity(req.TeamID, req.Name, req.Settings, req.Steps, h.MonitorService.Locations())
	if err != nil {
		return badRequest(err)
	}

	if err := h.MonitorService.Create(c.Request().Context(), m); err != nil {
		c.Log.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type PostMonitorsBulkRequest struct {
	TeamID   uint                      `param:"teamId" validate:"required,numeric,gte=0"`
	Monitors []PostMonitorsBulkMonitor `json:"monitors" validate:"required,min=1,dive"`
}

type PostMonitorsBulkMonitor struct {
	Name     string          `json:"name" validate:"required,max=255"`
	Settings MonitorSettings `json:"settings"`
	Steps    []MonitorStep   `json:"steps" validate:"required,min=1,dive"`
}

func (h *Handlers) PostMonitorsBulk(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorsBulkRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorsBulkRequest")

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(req.Monitors) > maxMonitorsPerBulk {
		return echo.NewHTTPError(http.StatusBadRequest, "too many monitors in one request")
	}

	monitors := make([]*entities.Monitor, len(req.Monitors))
	for i, rm := range req.Monitors {
		m, err := newMonitorEntity(req.TeamID, rm.Name, rm.Settings, rm.Steps, h.MonitorService.Locations())
		if err != nil {
			var vErr *ValidationError
			if errors.As(err, &vErr) {
				return echo.NewHTTPError(http.StatusBadRequest, "monitors["+itoa(i)+"]: "+vErr.Error())
			}

			return badRequest(err)
		}

		monitors[i] = m
	}

	if err := h.MonitorService.CreateMany(c.Request().Context(), monitors); err != nil {
		c.Log.WithError(err).Error("failed to create monitors")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type GetMonitorIncidents struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type GetMonitorIncidentsResponse struct {
	Monitors []MonitorWithIncidents `json:"monitors"`
}

type MonitorWithIncidents struct {
	ID        uint       `json:"id"`
	TeamID    uint       `json:"teamId"`
	State     string     `json:"state" validate:"required,monitorState"`
	Name      string     `json:"name" validate:"required,max=255"`
	Incidents []Incident `json:"incidents" validate:"required,incidents"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Incident struct {
	ID                 uint      `json:"id"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	MonitorAssertionID uint      `json:"monitorAssertionId"`
}

func (h *Handlers) GetMonitorIncidents(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorIncidents](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetMonitorIncidents")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetMonitorsAndIncidentsByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get monitor incidents")
		return echo.ErrInternalServerError
	}

	// manuel filtering should be in query :P
	filteredMonitors := make([]entities.Monitor, 0, len(*monitors))
	for _, m := range *monitors {
		if len(m.Incidents) > 0 {
			filteredMonitors = append(filteredMonitors, m)
		}
	}

	resp, err := newGetMonitorWithIncidentsResponse(&filteredMonitors)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorIncidentsResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorWithIncidentsResponse(monitors *[]entities.Monitor) (*GetMonitorIncidentsResponse, error) {
	res := make([]MonitorWithIncidents, len(*monitors))

	for i, m := range *monitors {
		monitorWithIncidents := MonitorWithIncidents{
			ID:        m.ID,
			TeamID:    m.TeamID,
			State:     m.GetStateString(),
			Name:      m.Name,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			Incidents: make([]Incident, len(m.Incidents)),
		}
		for j, incident := range m.Incidents {
			monitorWithIncidents.Incidents[j] = Incident{
				ID:                 incident.ID,
				CreatedAt:          incident.CreatedAt,
				UpdatedAt:          incident.UpdatedAt,
				MonitorAssertionID: incident.MonitorAssertionID,
			}
		}
		res[i] = monitorWithIncidents

	}
	return &GetMonitorIncidentsResponse{
		Monitors: res,
	}, nil
}

type PutMonitorRequest struct {
	TeamID    uint            `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint            `param:"monitorId" validate:"required,numeric,gte=0"`
	Name      string          `json:"name" validate:"required,max=255"`
	State     string          `json:"state" validate:"required,monitorState"`
	Settings  MonitorSettings `json:"settings"`
	Steps     []MonitorStep   `json:"steps" validate:"required,min=1,dive"`
}

func (h *Handlers) PutMonitor(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorRequest")

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	m, err := newMonitorEntity(req.TeamID, req.Name, req.Settings, req.Steps, h.MonitorService.Locations())
	if err != nil {
		return badRequest(err)
	}

	if err := m.SetStateString(req.State); err != nil {
		return echo.ErrBadRequest
	}

	if err := h.MonitorService.Update(
		ctx,
		req.TeamID,
		req.MonitorID,
		m,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to update monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PutMonitorStateRequest struct {
	TeamID    uint   `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint   `param:"monitorId" validate:"required,numeric,gte=0"`
	State     string `json:"state" validate:"required,monitorState"`
}

func (h *Handlers) PutMonitorState(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutMonitorStateRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorStateRequest")

		return echo.ErrBadRequest
	}

	stateEnum := entities.GetMonitorStateEnumFromString(req.State)

	if err := h.MonitorService.SetState(
		ctx,
		req.TeamID,
		req.MonitorID,
		stateEnum,
	); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to set monitor state")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

func badRequest(err error) error {
	var vErr *ValidationError
	if errors.As(err, &vErr) {
		return echo.NewHTTPError(http.StatusBadRequest, vErr.Error())
	}

	return echo.ErrBadRequest
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
