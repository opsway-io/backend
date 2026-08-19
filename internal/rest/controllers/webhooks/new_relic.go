package webhooks

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
)

type NewRelicWebhookPayload struct {
	ConditionName string `json:"condition_name"`
	Details       string `json:"details"`
}

func (h *Handlers) PostNewRelicWebhook(c echo.Context) error {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid team id")
	}

	// Verify team exists
	_, err = h.TeamService.GetByID(c.Request().Context(), uint(teamID))
	if err != nil {
		c.Logger().Errorf("failed to fetch team %d: %v", teamID, err)
		return c.String(http.StatusNotFound, "team not found")
	}

	var payload NewRelicWebhookPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		c.Logger().Errorf("failed to decode new relic payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	incidents := []entities.Incident{
		{
			TeamID:      uint(teamID),
			Title:       "New Relic Alert: " + payload.ConditionName,
			Description: &payload.Details,
		},
	}

	if err := h.IncidentService.Create(c.Request().Context(), &incidents); err != nil {
		c.Logger().Errorf("failed to create incident for new relic alert: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}
