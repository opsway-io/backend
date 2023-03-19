package monitors

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
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
	Name      string                            `json:"name"`
	Tags      []string                          `json:"tags"`
	Settings  GetMonitorResponseMonitorSettings `json:"settings"`
	CreatedAt time.Time                         `json:"createdAt"`
	UpdatedAt time.Time                         `json:"updatedAt"`
}

type GetMonitorResponseMonitorSettings struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	BodyType  string            `json:"bodyType"`
	Body      *string           `json:"body"`
	Frequency uint64            `json:"frequency"` // milliseconds
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

	resp, err := newGetMonitorsResponse(monitors)
	if err != nil {
		c.Log.WithError(err).Error("failed to create GetMonitorsResponse")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, resp)
}

func newGetMonitorsResponse(monitors *[]monitor.MonitorWithTotalCount) (*GetMonitorsResponse, error) {
	res := make([]GetMonitorResponseMonitor, len(*monitors))

	for i, m := range *monitors {
		headers, err := m.Settings.GetHeaders()
		if err != nil {
			return nil, err
		}

		res[i] = GetMonitorResponseMonitor{
			ID:        m.ID,
			Name:      m.Name,
			Tags:      m.Tags,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			Settings: GetMonitorResponseMonitorSettings{
				Method:    m.Settings.Method,
				URL:       m.Settings.URL,
				Headers:   headers,
				BodyType:  m.Settings.BodyType,
				Body:      m.GetBodyStr(),
				Frequency: m.Settings.GetFrequencyMilliseconds(),
			},
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
	Tags      []string                   `json:"tags"`
	Settings  GetMonitorResponseSettings `json:"settings"`
	CreatedAt time.Time                  `json:"createdAt"`
	UpdatedAt time.Time                  `json:"updatedAt"`
}

type GetMonitorResponseSettings struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	BodyType  string            `json:"bodyType"`
	Body      *string           `json:"body"`
	Frequency uint64            `json:"frequency"`
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
		State:     m.StateString(),
		Name:      m.Name,
		Tags:      m.Tags,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Settings: GetMonitorResponseSettings{
			Method:    m.Settings.Method,
			URL:       m.Settings.URL,
			Headers:   headers,
			BodyType:  m.Settings.BodyType,
			Body:      m.GetBodyStr(),
			Frequency: m.Settings.GetFrequencyMilliseconds(),
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

	if err := h.MonitorService.Delete(c.Request().Context(), req.MonitorID); err != nil {
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
	Tags     []string                   `json:"tags" validate:"required,max=10,dive,max=255"`
	Settings PostMonitorRequestSettings `json:"settings" validate:"required,dive"`
}

type PostMonitorRequestSettings struct {
	Method    string            `json:"method" validate:"required,monitorMethod"`
	URL       string            `json:"url" validate:"required,url"`
	Headers   map[string]string `json:"headers" validate:"required,dive,max=255"`
	BodyType  string            `json:"bodyType" validate:"required,monitorBodyType"`
	Body      string            `json:"body"`
	Frequency uint64            `json:"frequency" validate:"required,numeric,gte=0,monitorFrequency"`
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
	s.SetFrequencyMilliseconds(req.Settings.Frequency)

	m := &entities.Monitor{
		TeamID:   req.TeamID,
		Name:     req.Name,
		Tags:     req.Tags,
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
