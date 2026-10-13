package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### New Relic Alert: %s\n\n", payload.ConditionName))
	if payload.Details != "" {
		sb.WriteString("### Violation Details & Diagnostics\n")
		sb.WriteString(payload.Details + "\n\n")
	}
	sb.WriteString("### Recommended Remediation Steps\n")
	sb.WriteString("1. Inspect the New Relic incident violation timeline and telemetry charts for anomalies.\n")
	sb.WriteString("2. Correlate with recent service deployments, error rates, and throughput metrics.\n")
	sb.WriteString("3. Check downstream database queries and external service call latencies.\n")

	desc := sb.String()

	incidents := []entities.Incident{
		{
			TeamID:      uint(teamID),
			Title:       "New Relic Alert: " + payload.ConditionName,
			Description: &desc,
		},
	}

	if err := h.IncidentService.Create(c.Request().Context(), &incidents); err != nil {
		c.Logger().Errorf("failed to create incident for new relic alert: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}
