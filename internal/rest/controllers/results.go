package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetMonitorResultsRequest struct {
	TeamID    uint64 `param:"teamId" validate:"required,numeric,gte=0"`
	MonitorID uint64 `param:"monitorId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetResults(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetMonitorResultsRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetMonitorsRequest")

		return echo.ErrBadRequest
	}

	results, err := h.HttpResultService.GetMonitorResultsID(ctx.Request().Context(), req.MonitorID)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to get monitors")

		return echo.ErrInternalServerError
	}

	// TODO CREATE RESONSE MAPPING

	return ctx.JSON(http.StatusOK, results)
}
