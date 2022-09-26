package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorsRequest struct {
	TeamID int `param:"teamId" validate:"required,numeric,gte=0"`
	Offset int `query:"offset" validate:"numeric,min=0"`
	Limit  int `query:"limit" validate:"numeric,min=0,max=100" default:"25"`
}

type GetMonitorsResponse struct {
	// TODO
}

func (h *Handlers) GetMonitors(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[GetMonitorsRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.JSON(http.StatusNotImplemented, nil)
}

type GetMonitorRequest struct {
	TeamID    int `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID int `param:"monitorId" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	// TODO
}

func (h *Handlers) GetMonitor(ctx hs.AuthenticatedContext) error {
	_, err := helpers.Bind[GetMonitorRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}

	// TODO

	return ctx.JSON(http.StatusNotImplemented, nil)
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
		ctx.Log.WithError(err).Error("failed to delete monitor")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
