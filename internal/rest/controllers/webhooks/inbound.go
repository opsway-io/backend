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

		desc := formatPrometheusIncidentDescription(alert, payload.Status)
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
	desc := formatCloudWatchIncidentDescription(payload)

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
	desc := formatGenericInboundIncidentDescription(payload)

	inc := entities.Incident{
		TeamID:      uint(teamID),
		Title:       fmt.Sprintf("[%s] %s", source, payload.Title),
		Description: &desc,
		Resolved:    isResolved,
	}

	if err := h.IncidentService.Create(c.Request().Context(), &[]entities.Incident{inc}); err != nil {
		c.Logger().Errorf("failed to create incident for inbound webhook: %v", err)
		return c.String(http.StatusInternalServerError, "failed to create incident")
	}

	return c.String(http.StatusOK, "ok")
}

func formatPrometheusIncidentDescription(alert PrometheusAlert, payloadStatus string) string {
	var sb strings.Builder
	sb.WriteString("### Prometheus Alertmanager Notification\n\n")

	summary := alert.Annotations["summary"]
	description := alert.Annotations["description"]
	if summary != "" {
		sb.WriteString(fmt.Sprintf("**Summary**: %s\n\n", summary))
	}
	if description != "" {
		sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", description))
	}

	sb.WriteString("### Alert Context & Target Details\n")
	if sev := alert.Labels["severity"]; sev != "" {
		sb.WriteString(fmt.Sprintf("- **Severity**: `%s`\n", strings.ToUpper(sev)))
	}
	sb.WriteString(fmt.Sprintf("- **Alert Status**: `%s`\n", alert.Status))
	if inst := alert.Labels["instance"]; inst != "" {
		sb.WriteString(fmt.Sprintf("- **Target Instance**: `%s`\n", inst))
	}
	if pod := alert.Labels["pod"]; pod != "" {
		sb.WriteString(fmt.Sprintf("- **Kubernetes Pod**: `%s`\n", pod))
	}
	if ns := alert.Labels["namespace"]; ns != "" {
		sb.WriteString(fmt.Sprintf("- **Namespace**: `%s`\n", ns))
	}
	if cluster := alert.Labels["cluster"]; cluster != "" {
		sb.WriteString(fmt.Sprintf("- **Cluster**: `%s`\n", cluster))
	}
	if job := alert.Labels["job"]; job != "" {
		sb.WriteString(fmt.Sprintf("- **Scrape Job**: `%s`\n", job))
	}
	if alert.StartsAt != "" {
		sb.WriteString(fmt.Sprintf("- **Triggered At**: `%s`\n", alert.StartsAt))
	}
	sb.WriteString("\n")

	if len(alert.Labels) > 0 {
		sb.WriteString("### Active Prometheus Labels\n")
		sb.WriteString("```yaml\n")
		for k, v := range alert.Labels {
			sb.WriteString(fmt.Sprintf("%s: \"%s\"\n", k, v))
		}
		sb.WriteString("```\n\n")
	}

	if runbook := alert.Annotations["runbook_url"]; runbook != "" {
		sb.WriteString(fmt.Sprintf("🔗 **Runbook**: [%s](%s)\n\n", runbook, runbook))
	}
	if dashboard := alert.Annotations["dashboard_url"]; dashboard != "" {
		sb.WriteString(fmt.Sprintf("📊 **Dashboard**: [%s](%s)\n\n", dashboard, dashboard))
	}

	return sb.String()
}

func formatCloudWatchIncidentDescription(payload CloudWatchAlarmPayload) string {
	var sb strings.Builder
	sb.WriteString("### AWS CloudWatch Alarm Notification\n\n")
	if payload.AlarmDescription != "" {
		sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", payload.AlarmDescription))
	}

	sb.WriteString("### Alarm Details & Threshold Breach\n")
	sb.WriteString(fmt.Sprintf("- **Alarm Name**: `%s`\n", payload.AlarmName))
	sb.WriteString(fmt.Sprintf("- **New State**: `%s`\n", payload.NewStateValue))
	sb.WriteString(fmt.Sprintf("- **Trigger Reason**: %s\n", payload.NewStateReason))
	if payload.Region != "" {
		sb.WriteString(fmt.Sprintf("- **AWS Region**: `%s`\n", payload.Region))
	}
	if payload.StateChangeTime != "" {
		sb.WriteString(fmt.Sprintf("- **State Change Time**: `%s`\n", payload.StateChangeTime))
	}
	sb.WriteString("\n")

	sb.WriteString("### Recommended Remediation Steps\n")
	if payload.Region != "" {
		sb.WriteString(fmt.Sprintf("1. Open the [AWS CloudWatch Alarms Console](https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#alarmsV2:) in region `%s`.\n", payload.Region, payload.Region, payload.Region))
	} else {
		sb.WriteString("1. Open the AWS CloudWatch Alarms Console.\n")
	}
	sb.WriteString(fmt.Sprintf("2. Review the metric graph and alarm history for `%s`.\n", payload.AlarmName))
	sb.WriteString("3. Correlate with AWS CloudTrail events or recent infrastructure deployments in the region.\n")

	return sb.String()
}

func formatGenericInboundIncidentDescription(payload GenericInboundAlertPayload) string {
	var sb strings.Builder
	sourceName := payload.Source
	if sourceName == "" {
		sourceName = "External Monitoring System"
	}
	sb.WriteString(fmt.Sprintf("### Alert Ingested from %s\n\n", sourceName))
	if payload.Description != "" {
		sb.WriteString(fmt.Sprintf("**Details**: %s\n\n", payload.Description))
	}
	sb.WriteString("### Incident Metadata\n")
	sb.WriteString(fmt.Sprintf("- **Source**: `%s`\n", sourceName))
	if payload.Status != "" {
		sb.WriteString(fmt.Sprintf("- **Reported Status**: `%s`\n", payload.Status))
	}
	if payload.Title != "" {
		sb.WriteString(fmt.Sprintf("- **Alert Title**: `%s`\n", payload.Title))
	}
	sb.WriteString("\n")
	sb.WriteString("### Recommended Action\n")
	sb.WriteString(fmt.Sprintf("Investigate the originating event in `%s` and review affected services.\n", sourceName))
	return sb.String()
}
