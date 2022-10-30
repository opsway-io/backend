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

	for i, m := range *monitors {
		headers, err := m.Settings.GetHeaders()
		if err != nil {
			return nil, err
		}

		res.Monitors[i] = GetMonitorResponseMonitor{
			ID:   m.ID,
			Name: m.Name,
			Tags: m.GetTags(),
			Settings: GetMonitorResponseMonitorSettings{
				Method:    m.Settings.Method,
				URL:       m.Settings.URL,
				Headers:   headers,
				Body:      m.GetBodyStr(),
				Frequency: m.Settings.Frequency,
			},
		}
	}

	return &res, nil
}

type GetMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
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
	Body      *string           `json:"body"`
	Frequency time.Duration     `json:"frequency"`
}

func (h *Handlers) GetMonitor(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}

	m, err := h.MonitorService.GetMonitorAndSettingsByTeamIDAndID(ctx.Request().Context(), req.TeamID, req.MonitorID)
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

	return ctx.JSON(http.StatusOK, resp)
}

func newGetMonitorResponse(m *entities.Monitor) (*GetMonitorResponse, error) {
	headers, err := m.Settings.GetHeaders()
	if err != nil {
		return nil, err
	}

	return &GetMonitorResponse{
		ID:   m.ID,
		Name: m.Name,
		Tags: m.GetTags(),
		Settings: GetMonitorResponseSettings{
			Method:    m.Settings.Method,
			URL:       m.Settings.URL,
			Headers:   headers,
			Body:      m.GetBodyStr(),
			Frequency: m.Settings.Frequency,
		},
	}, nil
}

type DeleteMonitorRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
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

type PostMonitorRequest struct {
	TeamID   uint                       `param:"teamId" validate:"required,numeric,gte=0"`
	Name     string                     `json:"name" validate:"required,max=255"`
	Tags     []string                   `json:"tags" validate:"required,max=255,dive,max=255"`
	Settings PostMonitorRequestSettings `json:"settings" validate:"required,dive"`
}

type PostMonitorRequestSettings struct {
	Method    string            `json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE"`
	URL       string            `json:"url" validate:"required,url"`
	Headers   map[string]string `json:"headers" validate:"required,dive,max=255"`
	Body      string            `json:"body"`
	Frequency time.Duration     `json:"frequency" validate:"required"`
}

func (h *Handlers) PostMonitor(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostMonitorRequest")

		return echo.ErrBadRequest
	}

	m := &entities.Monitor{
		TeamID: req.TeamID,
		Name:   req.Name,
		Settings: entities.MonitorSettings{
			Method:    req.Settings.Method,
			URL:       req.Settings.URL,
			Frequency: req.Settings.Frequency,
		},
	}

	m.SetTags(req.Tags)
	m.SetBodyStr(req.Settings.Body)
	m.SetHeaders(req.Settings.Headers)

	if err := h.MonitorService.Create(ctx.Request().Context(), m); err != nil {
		ctx.Log.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusCreated)
}

type GetMonitorChecksRequest struct {
	TeamID    uint `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetMonitorChecks(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorChecksRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	results, err := h.CheckService.GetMonitorChecksByID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	// TODO CREATE RESONSE MAPPING

	return ctx.JSON(http.StatusOK, results)
}

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
