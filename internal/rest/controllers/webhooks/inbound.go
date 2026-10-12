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

type PrometheusAlert struct {
	Status      string            `json:"status"` // firing or resolved
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}

type PrometheusAlertmanagerPayload struct {
	Receiver string             `json:"receiver"`
	Status   string             `json:"status"` // firing or resolved
	Alerts   []PrometheusAlert  `json:"alerts"`
}

type CloudWatchAlarmPayload struct {
	AlarmName        string `json:"AlarmName"`
	NewStateValue    string `json:"NewStateValue"` // ALARM, OK, INSUFFICIENT_DATA
	NewStateReason   string `json:"NewStateReason"`
	StateChangeTime  string `json:"StateChangeTime"`
	Region           string `json:"Region"`
	AlarmDescription string `json:"AlarmDescription"`
}

type GenericInboundAlertPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "down", "firing", "outage", "degraded", or "up", "resolved", "ok"
	Source      string `json:"source"` // optional source label, e.g. "Pingdom", "Grafana"
}

func (h *Handlers) PostPrometheusWebhook(c echo.Context) error {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid team id")
	}

	_, err = h.TeamService.GetByID(c.Request().Context(), uint(teamID))
	if err != nil {
		c.Logger().Errorf("failed to fetch team %d: %v", teamID, err)
		return c.String(http.StatusNotFound, "team not found")
	}

	var payload PrometheusAlertmanagerPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		c.Logger().Errorf("failed to decode prometheus payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	for _, alert := range payload.Alerts {
		alertName := alert.Labels["alertname"]
		if alertName == "" {
			alertName = "Alertmanager Notification"
		}

		desc := alert.Annotations["description"]
		if desc == "" {
			desc = alert.Annotations["summary"]
		}
		if desc == "" {
			desc = fmt.Sprintf("Alertmanager status: %s", alert.Status)
		}

		isResolved := strings.ToLower(alert.Status) == "resolved" || strings.ToLower(payload.Status) == "resolved"

		inc := entities.Incident{
			TeamID:      uint(teamID),
			Title:       "Prometheus: " + alertName,
			Description: &desc,
			Resolved:    isResolved,
		}

		if err := h.IncidentService.Create(c.Request().Context(), &[]entities.Incident{inc}); err != nil {
			c.Logger().Errorf("failed to create incident for prometheus alert: %v", err)
		}
	}

	return c.String(http.StatusOK, "ok")
}

func (h *Handlers) PostCloudWatchWebhook(c echo.Context) error {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid team id")
	}

	_, err = h.TeamService.GetByID(c.Request().Context(), uint(teamID))
	if err != nil {
		c.Logger().Errorf("failed to fetch team %d: %v", teamID, err)
		return c.String(http.StatusNotFound, "team not found")
	}

	var payload CloudWatchAlarmPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		c.Logger().Errorf("failed to decode cloudwatch payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	isResolved := payload.NewStateValue == "OK"
	desc := fmt.Sprintf("%s (Reason: %s)", payload.AlarmDescription, payload.NewStateReason)

	inc := entities.Incident{
		TeamID:      uint(teamID),
		Title:       "AWS CloudWatch: " + payload.AlarmName,
		Description: &desc,
		Resolved:    isResolved,
	}

	if err := h.IncidentService.Create(c.Request().Context(), &[]entities.Incident{inc}); err != nil {
		c.Logger().Errorf("failed to create incident for cloudwatch alert: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}

func (h *Handlers) PostGenericInboundWebhook(c echo.Context) error {
	teamIDStr := c.Param("teamId")
	teamID, err := strconv.ParseUint(teamIDStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid team id")
	}

	_, err = h.TeamService.GetByID(c.Request().Context(), uint(teamID))
	if err != nil {
		c.Logger().Errorf("failed to fetch team %d: %v", teamID, err)
		return c.String(http.StatusNotFound, "team not found")
	}

	var payload GenericInboundAlertPayload
	if err := json.NewDecoder(c.Request().Body).Decode(&payload); err != nil {
		c.Logger().Errorf("failed to decode generic inbound payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	if payload.Title == "" {
		payload.Title = "External Monitoring Alert"
	}

	source := "External"
	if payload.Source != "" {
		source = payload.Source
	}

	st := strings.ToLower(payload.Status)
	isResolved := st == "up" || st == "ok" || st == "resolved"

	inc := entities.Incident{
		TeamID:      uint(teamID),
		Title:       fmt.Sprintf("[%s] %s", source, payload.Title),
		Description: &payload.Description,
		Resolved:    isResolved,
	}

	if err := h.IncidentService.Create(c.Request().Context(), &[]entities.Incident{inc}); err != nil {
		c.Logger().Errorf("failed to create incident for inbound webhook: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}
