package controllers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
)

type GetMonitorsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	Offset int  `query:"offset" validate:"numeric,min=0"`
	Limit  int  `query:"limit" validate:"numeric,min=0,max=100" default:"25"`
}

type GetMonitorsResponse struct {
	Monitors []GetMonitorResponseMonitor `json:"monitors"`
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
	Body      *string           `json:"body"`
	Frequency time.Duration     `json:"frequency"`
}

func (h *Handlers) GetMonitors(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorsRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetMonitorsAndSettingsByTeamID(ctx.Request().Context(), req.TeamID, req.Offset, req.Limit)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	resp, err := newGetMonitorsResponse(monitors)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to create GetMonitorsResponse")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, resp)
}

func newGetMonitorsResponse(monitors *[]entities.Monitor) (*GetMonitorsResponse, error) {
	res := GetMonitorsResponse{
		Monitors: make([]GetMonitorResponseMonitor, len(*monitors)),
	}

	for i, monitor := range *monitors {
		headers, err := monitor.Settings.GetHeaders()
		if err != nil {
			return nil, err
		}

		res.Monitors[i] = GetMonitorResponseMonitor{
			ID:   monitor.ID,
			Name: monitor.Name,
			Tags: *monitor.Tags,
			Settings: GetMonitorResponseMonitorSettings{
				Method:    monitor.Settings.Method,
				URL:       monitor.Settings.URL,
				Headers:   headers,
				Body:      pointer.String(string(*monitor.Settings.Body)),
				Frequency: monitor.Settings.Frequency,
			},
		}
	}

	return &res, nil
}

type GetMonitorRequest struct {
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	ID        uint                       `json:"id"`
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
	Body      string            `json:"body"`
	Frequency time.Duration     `json:"frequency"`
}

func (h *Handlers) GetMonitor(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}

	m, err := h.MonitorService.GetMonitorAndSettingsByID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	resp, err := newGetMonitorResponse(m)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to create GetMonitorResponse")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusNotImplemented, resp)
}

func newGetMonitorResponse(m *entities.Monitor) (*GetMonitorResponse, error) {
	headers, err := m.Settings.GetHeaders()
	if err != nil {
		return nil, err
	}

	return &GetMonitorResponse{
		ID:   m.ID,
		Name: m.Name,
		Tags: *m.Tags,
		Settings: GetMonitorResponseSettings{
			Method:    m.Settings.Method,
			URL:       m.Settings.URL,
			Headers:   headers,
			Body:      string(*m.Settings.Body),
			Frequency: m.Settings.Frequency,
		},
	}, nil
}

type DeleteMonitorRequest struct {
	TeamID    int `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID int `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteMonitor(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteMonitorRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind DeleteMonitorRequest")

		return echo.ErrBadRequest
	}

	if err := h.MonitorService.Delete(ctx.Request().Context(), req.MonitorID); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		ctx.Log.WithError(err).Error("failed to delete monitor")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
