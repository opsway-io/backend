package v1

import (
	"errors"
	"net/http"

	"github.com/creasty/defaults"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
)

type GetMonitorsRequest struct {
	TeamID int `param:"team_id" validate:"required,numeric"`
	Offset int `query:"offset" validate:"numeric,min=0"`
	Limit  int `query:"limit" validate:"numeric,min=0,max=100" default:"100"`
}

type GetMonitorsResponse struct {
	Monitors []monitor.Monitor `json:"monitors"`
}

func (h *Handlers) GetMonitors(ctx echo.Context, l *logrus.Entry) error {
	var req GetMonitorsRequest
	if err := ctx.Bind(&req); err != nil {
		l.WithError(err).Debug("failed to bind request")

		return echo.ErrBadRequest
	}

	if err := ctx.Validate(&req); err != nil {
		l.WithError(err).Debug("request failed validation")

		return echo.ErrBadRequest
	}

	if err := defaults.Set(&req); err != nil {
		l.WithError(err).Debug("failed to set defaults")

		return echo.ErrInternalServerError
	}

	monitors, err := h.MonitorService.GetByTeamID(ctx.Request().Context(), req.TeamID, req.Offset, req.Limit)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, GetMonitorsResponse{
		Monitors: monitors,
	})
}

type GetMonitorRequest struct {
	TeamID    int `param:"team_id" validate:"required,numeric"`
	MonitorID int `param:"monitor_id" validate:"required,numeric"`
}

func (h *Handlers) GetMonitor(ctx echo.Context, l *logrus.Entry) error {
	var req GetMonitorRequest
	if err := ctx.Bind(&req); err != nil {
		l.WithError(err).Debug("failed to bind request")

		return echo.ErrBadRequest
	}

	if err := ctx.Validate(&req); err != nil {
		l.WithError(err).Debug("request failed validation")

		return echo.ErrBadRequest
	}

	m, err := h.MonitorService.GetByTeamIDAndID(ctx.Request().Context(), req.TeamID, req.MonitorID)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			return echo.ErrNotFound
		}

		l.WithError(err).Error("failed to get monitor")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, m)
}
