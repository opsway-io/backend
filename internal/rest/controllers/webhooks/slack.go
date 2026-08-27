package webhooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type SlackAction struct {
	ActionID string `json:"action_id"`
	Value    string `json:"value"`
}

type SlackMessage struct {
	Blocks []map[string]interface{} `json:"blocks"`
}

type SlackPayload struct {
	Type        string        `json:"type"`
	Actions     []SlackAction `json:"actions"`
	ResponseURL string        `json:"response_url"`
	Message     SlackMessage  `json:"message"`
}

func (h *Handlers) PostSlackInteractive(c echo.Context) error {
	payloadStr := c.FormValue("payload")
	if payloadStr == "" {
		return c.String(http.StatusBadRequest, "missing payload")
	}

	var payload SlackPayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		c.Logger().Errorf("failed to unmarshal slack payload: %v", err)
		return c.String(http.StatusBadRequest, "invalid payload")
	}

	if len(payload.Actions) == 0 {
		return c.String(http.StatusOK, "ok")
	}

	action := payload.Actions[0]
	var incidentIDStr string
	var isAck bool

	if strings.HasPrefix(action.Value, "ack_") {
		incidentIDStr = strings.TrimPrefix(action.Value, "ack_")
		isAck = true
	} else if strings.HasPrefix(action.Value, "resolve_") {
		incidentIDStr = strings.TrimPrefix(action.Value, "resolve_")
		isAck = false
	} else {
		return c.String(http.StatusOK, "ok")
	}

	incidentID, err := strconv.ParseUint(incidentIDStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid incident id")
	}

	// Fetch incident
	incident, err := h.IncidentService.GetByID(c.Request().Context(), uint(incidentID))
	if err != nil {
		c.Logger().Errorf("failed to fetch incident %d: %v", incidentID, err)
		return c.String(http.StatusInternalServerError, "error fetching incident")
	}

	if isAck {
		incident.Acknowledged = true
		now := time.Now()
		incident.AcknowledgedAt = &now
		if err := h.IncidentService.Update(c.Request().Context(), incident); err != nil {
			c.Logger().Errorf("failed to update incident %d: %v", incidentID, err)
		}
	} else {
		incident.Resolved = true
		if err := h.IncidentService.Update(c.Request().Context(), incident); err != nil {
			c.Logger().Errorf("failed to update incident %d: %v", incidentID, err)
		}
	}

	// Post back to payload.ResponseURL to update the message visually
	go func() {
		statusText := "✅ Incident Resolved"
		if isAck {
			statusText = "👀 Incident Acknowledged"
		}

		resPayload := map[string]interface{}{
			"replace_original": true,
			"blocks": []map[string]interface{}{
				{
					"type": "section",
					"text": map[string]interface{}{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*%s*\nIssue: %s", statusText, incident.Title),
					},
				},
			},
		}

		body, _ := json.Marshal(resPayload)
		req, err := http.NewRequest(http.MethodPost, payload.ResponseURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err := http.DefaultClient.Do(req); err == nil {
				defer resp.Body.Close()
			}
		}
	}()

	return c.String(http.StatusOK, "ok")
}
