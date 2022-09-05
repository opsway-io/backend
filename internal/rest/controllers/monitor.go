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
	TeamID int `param:"team_id" validate:"required,numeric"`
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
	TeamID    int `param:"team_id" validate:"required,numeric"`
	MonitorID int `param:"monitor_id" validate:"required,numeric"`
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
