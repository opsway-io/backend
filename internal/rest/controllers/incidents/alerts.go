package incidents

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetIncidentAlertsRequest struct {
	TeamID     uint `param:"teamId" validate:"required,numeric,gte=0"`
	IncidentID uint `param:"incidentId" validate:"required,numeric,gte=0"`
	Offset     *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit      *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetIncidentAlertsResponse struct {
	TotalCount int                              `json:"totalCount"`
	Alerts     []GetIncidentAlertsResponseAlert `json:"alerts"`
}

type GetIncidentAlertsResponseAlert struct {
	ID          uint   `json:"id"`
	AlertRuleID uint   `json:"alertRuleId"`
	Channels    string `json:"channels"`
	CreatedAt   string `json:"createdAt"`
}

func (h *Handlers) GetIncidentAlerts(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetIncidentAlertsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetIncidentAlertsRequest")
		return echo.ErrBadRequest
	}

	totalCount, alerts, err := h.AlertingService.GetTriggersByIncidentID(c.Request().Context(), req.TeamID, req.IncidentID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get incident alerts")
		return echo.ErrInternalServerError
	}

	resp := &GetIncidentAlertsResponse{
		TotalCount: totalCount,
		Alerts:     make([]GetIncidentAlertsResponseAlert, len(alerts)),
	}

	for i, a := range alerts {
		resp.Alerts[i] = GetIncidentAlertsResponseAlert{
			ID:          a.ID,
			AlertRuleID: a.AlertRuleID,
			Channels:    a.Channels,
			CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, resp)
}
