package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type GetMonitorsRequest struct {
	TeamID int `param:"team_id" validate:"required,numeric,gte=0"`
	Offset int `query:"offset" validate:"numeric,min=0"`
	Limit  int `query:"limit" validate:"numeric,min=0,max=100" default:"100"`
}

type GetMonitorsResponse struct {
	Monitors []models.Monitor `json:"monitors"`
}

func (h *Handlers) GetMonitors(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[GetMonitorsRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	monitors, err := h.MonitorService.GetByTeamID(ctx.Request().Context(), req.TeamID, req.Offset, req.Limit)
	if err != nil {
		l.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, GetMonitorsResponse{
		Monitors: models.MonitorsToResponse(*monitors),
	})
}

type GetMonitorRequest struct {
	TeamID    int `param:"team_id" validate:"required,numeric,gte=0"`
	MonitorID int `param:"monitor_id" validate:"required,numeric,gte=0"`
}

type GetMonitorResponse struct {
	models.Monitor
}

func (h *Handlers) GetMonitor(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[GetMonitorRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind GetMonitorRequest")

		return echo.ErrBadRequest
	}

	m, err := h.MonitorService.GetByIDAndTeamID(ctx.Request().Context(), req.MonitorID, req.TeamID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			l.WithError(err).Debug("monitor not found")

			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.MonitorToResponse(*m))
}

type PostMonitorRequest struct {
	TeamID int `param:"team_id" validate:"required,numeric,gte=0"`
	models.Monitor
}

type PostMonitorResponse struct {
	models.Monitor
}

func (h *Handlers) PostMonitor(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[PostMonitorRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PostMonitorRequest")

		return echo.ErrBadRequest
	}

	m := models.RequestToMonitor(req.Monitor)
	m.TeamID = req.TeamID

	if err := h.MonitorService.Create(ctx.Request().Context(), &m); err != nil {
		l.WithError(err).Error("failed to create monitor")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusCreated, models.MonitorToResponse(m))
}

type PutMonitorRequest struct {
	TeamID    int `param:"team_id" validate:"required,numeric,gte=0"`
	MonitorID int `param:"monitor_id" validate:"required,numeric,gte=0"`
	models.Monitor
}

type PutMonitorResponse struct {
	models.Monitor
}

func (h *Handlers) PutMonitor(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[PutMonitorRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind PutMonitorRequest")

		return echo.ErrBadRequest
	}

	m := models.RequestToMonitor(req.Monitor)
	m.ID = req.MonitorID
	m.TeamID = req.TeamID

	if err := h.MonitorService.Update(ctx.Request().Context(), &m); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			l.WithError(err).Debug("monitor not found")

			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to update monitor")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, models.MonitorToResponse(m))
}

type DeleteMonitorRequest struct {
	TeamID    int `param:"team_id" validate:"required,numeric,gte=0"`
	MonitorID int `param:"monitor_id" validate:"required,numeric,gte=0"`
}

type DeleteMonitorResponse struct {
	models.Monitor
}

func (h *Handlers) DeleteMonitor(ctx handlers.AuthenticatedContext, l *logrus.Entry) error {
	req, err := helpers.Bind[DeleteMonitorRequest](ctx)
	if err != nil {
		l.WithError(err).Debug("failed to bind DeleteMonitorRequest")

		return echo.ErrBadRequest
	}

	if err := h.MonitorService.Delete(ctx.Request().Context(), req.MonitorID); err != nil {
		l.WithError(err).Error("failed to delete monitor")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}
