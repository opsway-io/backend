package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorsRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"omitempty"`
}

type GetMonitorsResponse struct {
	Monitors   []GetMonitorResponseMonitor `json:"monitors"`
	TotalCount int                         `json:"totalCount"`
}

type GetMonitorResponseMonitor struct {
	ID        uint                              `json:"id"`
	State     string                            `json:"state"`
	Name      string                            `json:"name"`
	Settings  GetMonitorResponseMonitorSettings `json:"settings"`
	CreatedAt time.Time                         `json:"createdAt"`
	UpdatedAt time.Time                         `json:"updatedAt"`
	P99       uint                              `json:"p99"`
	P95       uint                              `json:"p95"`
	Stats     []float64                         `json:"stats"`
}

type GetMonitorResponseMonitorSettings struct {
	Method           string            `json:"method"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	BodyType         string            `json:"bodyType"`
	Body             *string           `json:"body"`
	FrequencySeconds uint64            `json:"frequencySeconds"`
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

	resp, err := newGetMonitorsResponse(monitors, monitorStats)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorsResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorsResponse(monitors *[]monitor.MonitorWithTotalCount, stats *[]check.MonitorOverviews) (*GetMonitorsResponse, error) {
	res := make([]GetMonitorResponseMonitor, len(*monitors))

	monitorStatsMap := make(map[uint]check.MonitorOverviews, len(*stats))
	for _, m := range *stats {
		monitorStatsMap[m.MonitorID] = m
	}

	for i, m := range *monitors {
		headers, err := m.Settings.GetHeaders()
		if err != nil {
			return nil, err
		}

		res[i] = GetMonitorResponseMonitor{
			ID:        m.ID,
			State:     m.GetStateString(),
			Name:      m.Name,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			Settings: GetMonitorResponseMonitorSettings{
				Method:           m.Settings.Method,
				URL:              m.Settings.URL,
				Headers:          headers,
				BodyType:         m.Settings.BodyType,
				Body:             m.GetBodyStr(),
				FrequencySeconds: m.Settings.GetFrequencySeconds(),
			},
		}

		stat, ok := monitorStatsMap[m.ID]
		if ok {
			res[i].P99 = uint(stat.P99)
			res[i].P95 = uint(stat.P95)
			res[i].Stats = stat.Stats
		}
	}

	totalCount := 0
	if len(*monitors) > 0 {
		totalCount = (*monitors)[0].TotalCount
	}

	return &GetMonitorsResponse{
		Monitors:   res,
		TotalCount: totalCount,
	}, nil
}

type GetMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	ID        uint                       `json:"id"`
	State     string                     `json:"state"`
	Name      string                     `json:"name"`
	Settings  GetMonitorResponseSettings `json:"settings"`
	CreatedAt time.Time                  `json:"createdAt"`
	UpdatedAt time.Time                  `json:"updatedAt"`
}

type GetMonitorResponseSettings struct {
	Method           string            `json:"method"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers"`
	BodyType         string            `json:"bodyType"`
	Body             *string           `json:"body"`
	FrequencySeconds uint64            `json:"frequencySeconds"`
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

	resp, err := newGetMonitorResponse(m)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorResponse(m *entities.Monitor) (*GetMonitorResponse, error) {
	headers, err := m.Settings.GetHeaders()
	if err != nil {
		return nil, err
	}

	return &GetMonitorResponse{
		ID:        m.ID,
		State:     m.GetStateString(),
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Settings: GetMonitorResponseSettings{
			Method:           m.Settings.Method,
			URL:              m.Settings.URL,
			Headers:          headers,
			BodyType:         m.Settings.BodyType,
			Body:             m.GetBodyStr(),
			FrequencySeconds: m.Settings.GetFrequencySeconds(),
		},
	}, nil
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
	TeamID   uint                       `param:"teamId" validate:"required,numeric,gte=0"`
	Name     string                     `json:"name" validate:"required,max=255"`
	Settings PostMonitorRequestSettings `json:"settings" validate:"required,dive"`
}

type PostMonitorRequestSettings struct {
	Method           string            `json:"method" validate:"required,monitorMethod"`
	URL              string            `json:"url" validate:"required,url"`
	Headers          map[string]string `json:"headers" validate:"required,dive,max=255"`
	BodyType         string            `json:"bodyType" validate:"required,monitorBodyType"`
	Body             string            `json:"body"`
	FrequencySeconds uint64            `json:"frequencySeconds" validate:"required,numeric,gte=0,monitorFrequency"`
}

func (h *Handlers) PostMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorRequest")

		return echo.ErrBadRequest
	}

	s := entities.MonitorSettings{
		Method:   req.Settings.Method,
		URL:      req.Settings.URL,
		BodyType: req.Settings.BodyType,
	}
	s.SetFrequencySeconds(req.Settings.FrequencySeconds)

	m := &entities.Monitor{
		TeamID:   req.TeamID,
		Name:     req.Name,
		Settings: s,
	}
	m.SetBodyStr(req.Settings.Body)
	m.SetHeaders(req.Settings.Headers)

	if err := h.MonitorService.Create(c.Request().Context(), m); err != nil {
		c.Log.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type PutMonitorRequest struct {
	TeamID    uint                      `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint                      `param:"monitorId" validate:"required,numeric,gte=0"`
	Name      string                    `json:"name" validate:"required,max=255"`
	State     string                    `json:"state" validate:"required,monitorState"`
	Settings  PutMonitorRequestSettings `json:"settings" validate:"required,dive"`
}

type PutMonitorRequestSettings struct {
	Method           string            `json:"method" validate:"required,monitorMethod"`
	URL              string            `json:"url" validate:"required,url"`
	Headers          map[string]string `json:"headers" validate:"required,dive,max=255"`
	BodyType         string            `json:"bodyType" validate:"required,monitorBodyType"`
	Body             string            `json:"body"`
	FrequencySeconds uint64            `json:"frequencySeconds" validate:"required,numeric,gte=0,monitorFrequency"`
}

func (h *Handlers) PutMonitor(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutMonitorRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutMonitorRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	s := entities.MonitorSettings{
		Method:   req.Settings.Method,
		URL:      req.Settings.URL,
		BodyType: req.Settings.BodyType,
	}
	s.SetFrequencySeconds(req.Settings.FrequencySeconds)

	m := &entities.Monitor{
		ID:       req.MonitorID,
		TeamID:   req.TeamID,
		Name:     req.Name,
		Settings: s,
	}
	m.SetBodyStr(req.Settings.Body)
	m.SetHeaders(req.Settings.Headers)
	m.SetStateFromString(req.State)

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
