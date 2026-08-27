package alerting

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetAlertRuleTriggersRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	RuleID uint `param:"ruleId" validate:"required,numeric,gte=0"`
	Offset *int `query:"offset" validate:"omitempty,numeric,gte=0"`
	Limit  *int `query:"limit" validate:"omitempty,numeric,gte=0,max=255"`
}

type GetAlertRuleTriggersResponse struct {
	Triggers []GetAlertRuleTriggersResponseTrigger `json:"triggers"`
}

type GetAlertRuleTriggersResponseTrigger struct {
	ID         uint   `json:"id"`
	IncidentID *uint  `json:"incidentId"`
	Channels   string `json:"channels"`
	CreatedAt  string `json:"createdAt"`
}

func (h *Handlers) GetAlertRuleTriggers(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetAlertRuleTriggersRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetAlertRuleTriggersRequest")
		return echo.ErrBadRequest
	}

	triggers, err := h.AlertingService.GetTriggersByRuleID(c.Request().Context(), req.TeamID, req.RuleID, req.Offset, req.Limit)
	if err != nil {
		c.Log.WithError(err).Error("failed to get alert rule triggers")
		return echo.ErrInternalServerError
	}

	resp := &GetAlertRuleTriggersResponse{
		Triggers: make([]GetAlertRuleTriggersResponseTrigger, len(triggers)),
	}

	for i, t := range triggers {
		resp.Triggers[i] = GetAlertRuleTriggersResponseTrigger{
			ID:         t.ID,
			IncidentID: t.IncidentID,
			Channels:   t.Channels,
			CreatedAt:  t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, resp)
}
