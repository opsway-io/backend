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

type DatadogWebhookPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (h *Handlers) PostDatadogWebhook(c echo.Context) error {
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

	var payload DatadogWebhookPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		c.Logger().Errorf("failed to decode datadog payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	var sb strings.Builder
	sb.WriteString("### Datadog Monitor Alert\n\n")
	if payload.Title != "" {
		sb.WriteString(fmt.Sprintf("**Alert Title**: %s\n\n", payload.Title))
	}
	if payload.Body != "" {
		sb.WriteString("### Alert Body & Diagnostics\n")
		sb.WriteString(payload.Body + "\n\n")
	}
	sb.WriteString("### Recommended Remediation Steps\n")
	sb.WriteString("1. Open the Datadog Monitors console and inspect the associated metric query graphs.\n")
	sb.WriteString("2. Correlate with recent APM service traces, error rates, and host CPU/memory telemetry.\n")
	sb.WriteString("3. Check for deployment markers or configuration changes around the trigger timestamp.\n")

	desc := sb.String()

	incidents := []entities.Incident{
		{
			TeamID:      uint(teamID),
			Title:       "Datadog Alert: " + payload.Title,
			Description: &desc,
		},
	}

	if err := h.IncidentService.Create(c.Request().Context(), &incidents); err != nil {
		c.Logger().Errorf("failed to create incident for datadog alert: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}
