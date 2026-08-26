package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/escalation"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type WorkerConfig struct {
	ApplicationURL    string `mapstructure:"application_url" default:"http://localhost:5173"`
	StatusPageBaseURL string `mapstructure:"status_page_base_url" default:"http://localhost:5174"`
}

type Worker interface {
	Start(ctx context.Context) error
}

type worker struct {
	config        WorkerConfig
	eventService  event.Service
	alertService  Service
	teamService   team.Service
	monitorSvc    monitor.Service
	statusPageSvc statuspage.Service
	emailSender   email.Sender
	escalationSvc escalation.Service
	incidentSvc   incident.Service
	logger        *logrus.Entry
}

func NewWorker(
	config WorkerConfig,
	eventService event.Service,
	alertService Service,
	teamService team.Service,
	monitorSvc monitor.Service,
	statusPageSvc statuspage.Service,
	emailSender email.Sender,
	escalationSvc escalation.Service,
	incidentSvc incident.Service,
	logger *logrus.Entry,
) Worker {
	return &worker{
		config:        config,
		eventService:  eventService,
		alertService:  alertService,
		teamService:   teamService,
		monitorSvc:    monitorSvc,
		statusPageSvc: statusPageSvc,
		emailSender:   emailSender,
		escalationSvc: escalationSvc,
		incidentSvc:   incidentSvc,
		logger:        logger.WithField("component", "alerting_worker"),
	}
}

func (w *worker) Start(ctx context.Context) error {
	w.logger.Info("Starting alerting worker...")

	incidentMessages, err := w.eventService.Subscribe(ctx, string(events.EventTypeIncidentCreated))
	if err != nil {
		return fmt.Errorf("failed to subscribe to incident stream: %w", err)
	}

	maintenanceMessages, err := w.eventService.Subscribe(ctx, string(events.EventTypeMaintenance))
	if err != nil {
		return fmt.Errorf("failed to subscribe to maintenance stream: %w", err)
	}

	incidentPostMortemMessages, err := w.eventService.Subscribe(ctx, string(events.EventTypeIncidentPostMortemPublished))
	if err != nil {
		return fmt.Errorf("failed to subscribe to incident post_mortem stream: %w", err)
	}

	go func() {
		for msg := range incidentMessages {
			w.processMessage(ctx, msg.Payload)
			msg.Ack()
		}
	}()

	go func() {
		for msg := range incidentPostMortemMessages {
			w.processPostMortemMessage(ctx, msg.Payload)
			msg.Ack()
		}
	}()

	go func() {
		for msg := range maintenanceMessages {
			w.processMaintenanceMessage(ctx, msg.Payload)
			msg.Ack()
		}
	}()

	<-ctx.Done()
	return nil
}

func (w *worker) processPostMortemMessage(ctx context.Context, payload []byte) {
	var ev events.IncidentPostMortemPublishedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		w.logger.WithError(err).Error("failed to unmarshal event")
		return
	}

	incident := ev.Incident
	if incident == nil {
		return
	}

	w.notifyStatusPageSubscribersForPostMortem(ctx, incident)
}

func (w *worker) processMessage(ctx context.Context, payload []byte) {
	var ev events.IncidentCreatedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		w.logger.WithError(err).Error("failed to unmarshal event")
		return
	}

	incident := ev.Incident
	if incident == nil {
		return
	}

	rules, err := w.alertService.GetAllByTeamID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get alert rules")
		return
	}

	w.notifyStatusPageSubscribers(ctx, incident)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// MVP match: simple substring or check if condition is present in title
		if strings.Contains(strings.ToLower(incident.Title), strings.ToLower(rule.Condition)) ||
			rule.Condition == "monitor_down" || // Fallback matching for the UI example
			rule.Condition == "*" { // Wildcard
			w.triggerRule(ctx, incident, &rule, 1) // Base tier
		}
	}

	// For escalation check
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if strings.Contains(strings.ToLower(incident.Title), strings.ToLower(rule.Condition)) || rule.Condition == "monitor_down" || rule.Condition == "*" {
			w.scheduleEscalationCheck(ctx, incident, &rule)
			break // Only schedule once per incident
		}
	}
}

func (w *worker) processMaintenanceMessage(ctx context.Context, payload []byte) {
	var ev events.MaintenanceEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		w.logger.WithError(err).Error("failed to unmarshal maintenance event")
		return
	}

	maintenance := ev.Maintenance
	if maintenance == nil {
		return
	}

	rules, err := w.alertService.GetAllByTeamID(ctx, maintenance.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get alert rules")
		return
	}

	w.notifyMaintenanceStatusPageSubscribers(ctx, maintenance, ev.Action)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if strings.Contains(strings.ToLower(maintenance.Title), strings.ToLower(rule.Condition)) || rule.Condition == "*" {
			// Trigger rules for maintenance
			// We can adapt triggerRule to accept maintenance, but for now we'll mock or create a mock incident just to send the alert, or add a new trigger method.
			w.triggerMaintenanceRule(ctx, maintenance, &rule, ev.Action)
		}
	}
}

func (w *worker) notifyStatusPageSubscribers(ctx context.Context, incident *entities.Incident) {
	statusPages, err := w.statusPageSvc.GetByTeamID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch status pages for notification")
		return
	}

	for _, sp := range statusPages {
		hasMonitor := false
		for _, m := range sp.Monitors {
			if incident.MonitorID != nil && m.ID == *incident.MonitorID {
				hasMonitor = true
				break
			}
		}

		if !hasMonitor {
			continue
		}

		subs, err := w.statusPageSvc.GetVerifiedSubscribers(ctx, sp.ID)
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch subscribers for status page")
			continue
		}

		statusPageURL := fmt.Sprintf("%s/%s", w.config.StatusPageBaseURL, sp.Domain)
		for _, sub := range subs {
			unsubscribeURL := fmt.Sprintf("%s/%s/subscribe/%s", w.config.StatusPageBaseURL, sp.Domain, sub.Token)
			err := w.emailSender.Send(ctx, "", sub.Email, &templates.SubscriberIncidentAlertTemplate{
				StatusPageName: sp.Name,
				IncidentTitle:  incident.Title,
				StatusPageURL:  statusPageURL,
				IsResolved:     incident.Resolved,
				UnsubscribeURL: unsubscribeURL,
			})
			if err != nil {
				w.logger.WithError(err).Error("failed to send subscriber incident alert email")
			}
		}
	}
}

func (w *worker) notifyStatusPageSubscribersForPostMortem(ctx context.Context, incident *entities.Incident) {
	statusPages, err := w.statusPageSvc.GetByTeamID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch status pages for post mortem notification")
		return
	}

	for _, sp := range statusPages {
		hasMonitor := false
		for _, m := range sp.Monitors {
			if incident.MonitorID != nil && m.ID == *incident.MonitorID {
				hasMonitor = true
				break
			}
		}

		if !hasMonitor {
			continue
		}

		subs, err := w.statusPageSvc.GetVerifiedSubscribers(ctx, sp.ID)
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch subscribers for status page")
			continue
		}

		statusPageURL := fmt.Sprintf("%s/%s", w.config.StatusPageBaseURL, sp.Domain)
		for _, sub := range subs {
			unsubscribeURL := fmt.Sprintf("%s/%s/subscribe/%s", w.config.StatusPageBaseURL, sp.Domain, sub.Token)
			err := w.emailSender.Send(ctx, "", sub.Email, &templates.PostMortemPublishedTemplate{
				StatusPageName: sp.Name,
				IncidentTitle:  incident.Title,
				StatusPageURL:  statusPageURL,
				UnsubscribeURL: unsubscribeURL,
			})
			if err != nil {
				w.logger.WithError(err).Error("failed to send post mortem alert email")
			}
		}
	}
}

func (w *worker) notifyMaintenanceStatusPageSubscribers(ctx context.Context, maintenance *entities.Maintenance, action string) {
	statusPages, err := w.statusPageSvc.GetByTeamID(ctx, maintenance.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch status pages for notification")
		return
	}

	for _, sp := range statusPages {
		hasMonitor := false
		if len(maintenance.Monitors) == 0 {
			hasMonitor = true // global maintenance
		} else {
			for _, spm := range sp.Monitors {
				for _, mm := range maintenance.Monitors {
					if spm.ID == mm.ID {
						hasMonitor = true
						break
					}
				}
				if hasMonitor {
					break
				}
			}
		}

		if !hasMonitor {
			continue
		}

		subs, err := w.statusPageSvc.GetVerifiedSubscribers(ctx, sp.ID)
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch subscribers for status page")
			continue
		}

		statusPageURL := fmt.Sprintf("%s/%s", w.config.StatusPageBaseURL, sp.Domain)
		titleWithAction := fmt.Sprintf("[%s] %s", strings.ToUpper(action), maintenance.Title)
		for _, sub := range subs {
			unsubscribeURL := fmt.Sprintf("%s/%s/subscribe/%s", w.config.StatusPageBaseURL, sp.Domain, sub.Token)
			err := w.emailSender.Send(ctx, "", sub.Email, &templates.MaintenanceAlertTemplate{
				StatusPageName:   sp.Name,
				MaintenanceTitle: titleWithAction,
				StatusPageURL:    statusPageURL,
				UnsubscribeURL:   unsubscribeURL,
			})
			if err != nil {
				w.logger.WithError(err).Error("failed to send subscriber maintenance alert email")
			}
		}
	}
}

func (w *worker) scheduleEscalationCheck(ctx context.Context, incident *entities.Incident, rule *entities.AlertRule) {
	// Simple goroutine delay for MVP. In production, use a durable queue (e.g. Watermill delayed messages)
	policy, err := w.escalationSvc.GetPolicyByTeamID(ctx, incident.TeamID)
	if err != nil || policy == nil {
		return // No escalation policy
	}

	rotations, err := w.escalationSvc.GetRotationsByPolicyID(ctx, policy.ID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get rotations for escalation")
		return
	}

	maxTier := 1
	for _, r := range rotations {
		if r.Tier > maxTier {
			maxTier = r.Tier
		}
	}

	if maxTier <= 1 {
		return // No higher tiers to escalate to
	}

	go func() {
		bgCtx := context.Background()

		for currentTier := 2; currentTier <= maxTier; currentTier++ {
			w.logger.Infof("Scheduling escalation check for incident %d to Tier %d in %d minutes", incident.ID, currentTier, policy.EscalationTimeoutMinutes)
			time.Sleep(time.Duration(policy.EscalationTimeoutMinutes) * time.Minute)

			inc, err := w.incidentSvc.GetByID(bgCtx, incident.ID)
			if err != nil {
				w.logger.WithError(err).Error("failed to get incident during escalation check")
				return
			}

			if inc.Acknowledged || inc.Resolved {
				w.logger.Infof("Incident %d acknowledged or resolved, stopping escalation", incident.ID)
				return // Stop escalation loop
			}

			w.logger.Infof("Escalating incident %d to Tier %d", incident.ID, currentTier)
			w.triggerRule(bgCtx, inc, rule, currentTier)
		}
	}()
}

func (w *worker) triggerMaintenanceRule(ctx context.Context, maintenance *entities.Maintenance, rule *entities.AlertRule, action string) {
	// Create a mock incident to reuse the incident alerting channels
	mockTitle := fmt.Sprintf("Maintenance %s: %s", strings.ToUpper(action), maintenance.Title)
	mockDesc := maintenance.Description

	mockIncident := &entities.Incident{
		ID:       maintenance.ID, // use maintenance ID just so it's non-zero
		TeamID:   maintenance.TeamID,
		Title:    mockTitle,
		Resolved: action == "completed",
	}

	if mockDesc != nil && *mockDesc != "" {
		mockIncident.Description = mockDesc
	}

	if len(maintenance.Monitors) > 0 {
		mockIncident.MonitorID = &maintenance.Monitors[0].ID
	}

	w.triggerRule(ctx, mockIncident, rule, 1)
}

func (w *worker) triggerRule(ctx context.Context, incident *entities.Incident, rule *entities.AlertRule, tier int) {
	var channels []string
	if err := json.Unmarshal([]byte(rule.Channels), &channels); err != nil {
		w.logger.WithError(err).Error("failed to parse channels array")
		return
	}

	for _, channel := range channels {
		switch channel {
		case "email":
			w.sendEmailAlert(ctx, incident, tier)
		case "slack":
			w.sendSlackAlert(ctx, incident)
		case "microsoft_teams":
			w.sendMicrosoftTeamsAlert(ctx, incident)
		case "webhook":
			w.sendGenericWebhookAlert(ctx, incident)
		case "discord":
			w.sendDiscordAlert(ctx, incident)
		case "telegram":
			w.sendTelegramAlert(ctx, incident)
		case "sms":
			w.sendSmsAlert(ctx, incident, tier)
		case "voice":
			w.sendVoiceAlert(ctx, incident, tier)
		case "datadog":
			w.sendDatadogAlert(ctx, incident)
		case "new_relic":
			w.sendNewRelicAlert(ctx, incident)
		}
	}
}

func (w *worker) sendEmailAlert(ctx context.Context, incident *entities.Incident, tier int) {
	offset := 0
	limit := 100
	query := ""

	users, err := w.teamService.GetUsersByID(ctx, incident.TeamID, &offset, &limit, &query, nil)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team users")
		return
	}

	onCallUserIDs, err := w.escalationSvc.GetOnCallUsersByTeamID(ctx, incident.TeamID, tier)
	if err != nil {
		w.logger.WithError(err).Error("failed to get on call users")
	}

	onCallMap := make(map[uint]bool)
	for _, id := range onCallUserIDs {
		onCallMap[id] = true
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		// Ideally we would fetch the heartbeat name here, but for MVP "Unknown Monitor / Heartbeat" or generic is fine.
		monitorName = "Heartbeat Monitor"
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/incidents", w.config.ApplicationURL)

	for _, u := range *users {
		// If escalation policy exists for this team, only send to on-call users
		if len(onCallUserIDs) > 0 && !onCallMap[u.ID] {
			continue
		}

		userName := "Team Member"
		if u.DisplayName != nil {
			userName = *u.DisplayName
		}

		if incident.Title == "Anomaly Detected" {
			tpl := &templates.PerformanceDegradationTemplate{
				MonitorName:    monitorName,
				CurrentLatency: "Unexpected Spike",
				Threshold:      "Normal Baseline",
				DashboardURL:   dashboardURL,
			}
			err = w.emailSender.Send(ctx, "", u.Email, tpl)
		} else {
			tpl := &templates.IncidentAlertTemplate{
				Name:          userName,
				MonitorName:   monitorName,
				IncidentTitle: incident.Title,
				DashboardURL:  dashboardURL,
			}
			err = w.emailSender.Send(ctx, "", u.Email, tpl)
		}

		if err != nil {
			w.logger.WithError(err).Error("failed to send incident alert email")
		}
	}
}

func (w *worker) sendSlackAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for slack alert")
		return
	}

	if team.SlackWebhookURL == nil || *team.SlackWebhookURL == "" {
		w.logger.Debug("slack webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/incidents", w.config.ApplicationURL)

	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]interface{}{
					"type":  "plain_text",
					"text":  "🚨 Incident Alert",
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]interface{}{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Monitor:*\n%s", monitorName),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Issue:*\n%s", incident.Title),
					},
				},
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("View details and manage this incident on the dashboard:\n<%s|Go to Dashboard>", dashboardURL),
				},
			},
			{
				"type": "actions",
				"elements": []map[string]interface{}{
					{
						"type": "button",
						"text": map[string]interface{}{
							"type": "plain_text",
							"text": "Acknowledge",
						},
						"style":     "primary",
						"value":     fmt.Sprintf("ack_%d", incident.ID),
						"action_id": "acknowledge_incident",
					},
					{
						"type": "button",
						"text": map[string]interface{}{
							"type": "plain_text",
							"text": "Resolve",
						},
						"style":     "danger",
						"value":     fmt.Sprintf("resolve_%d", incident.ID),
						"action_id": "resolve_incident",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.SlackWebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create slack webhook request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send slack webhook")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("slack webhook returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("slack webhook sent successfully")
	}
}

func (w *worker) sendDiscordAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for discord alert")
		return
	}

	if team.DiscordWebhookURL == nil || *team.DiscordWebhookURL == "" {
		w.logger.Debug("discord webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/incidents", w.config.ApplicationURL)

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "🚨 Incident Alert: " + monitorName,
				"description": fmt.Sprintf("**Issue:**\n%s\n\n[View details and manage this incident on the dashboard](%s)", incident.Title, dashboardURL),
				"color":       16711680, // Red
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.DiscordWebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create discord webhook request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send discord webhook")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("discord webhook returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("discord webhook sent successfully")
	}
}

func (w *worker) sendTelegramAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for telegram alert")
		return
	}

	if team.TelegramChatID == nil || *team.TelegramChatID == "" {
		w.logger.Debug("telegram chat id not configured for team")
		return
	}

	// Assuming bot token is passed via env, using mock if empty
	// For production we'd inject this via WorkerConfig
	botToken := "mock_telegram_bot_token"

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/incidents", w.config.ApplicationURL)
	message := fmt.Sprintf("🚨 *Incident Alert*\n\n*Monitor:*\n%s\n\n*Issue:*\n%s\n\n[Go to Dashboard](%s)", monitorName, incident.Title, dashboardURL)

	payload := map[string]interface{}{
		"chat_id":    *team.TelegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create telegram request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send telegram alert")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("telegram alert returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("telegram alert sent successfully")
	}
}

func (w *worker) sendSmsAlert(ctx context.Context, incident *entities.Incident, tier int) {
	offset := 0
	limit := 100
	query := ""

	users, err := w.teamService.GetUsersByID(ctx, incident.TeamID, &offset, &limit, &query, nil)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team users for sms")
		return
	}

	onCallUserIDs, _ := w.escalationSvc.GetOnCallUsersByTeamID(ctx, incident.TeamID, tier)
	onCallMap := make(map[uint]bool)
	for _, id := range onCallUserIDs {
		onCallMap[id] = true
	}

	for _, u := range *users {
		if len(onCallUserIDs) > 0 && !onCallMap[u.ID] {
			continue
		}
		if u.PhoneNumber == nil || *u.PhoneNumber == "" {
			continue
		}

		w.logger.WithField("phone", *u.PhoneNumber).Info("mock sending SMS alert")
		// In production, we would use the Twilio Go SDK or HTTP request here
		// twilioClient.SendMessage("mock_twilio_from", *u.PhoneNumber, "Incident Alert: "+incident.Title)
	}
}

func (w *worker) sendVoiceAlert(ctx context.Context, incident *entities.Incident, tier int) {
	offset := 0
	limit := 100
	query := ""

	users, err := w.teamService.GetUsersByID(ctx, incident.TeamID, &offset, &limit, &query, nil)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team users for voice")
		return
	}

	onCallUserIDs, _ := w.escalationSvc.GetOnCallUsersByTeamID(ctx, incident.TeamID, tier)
	onCallMap := make(map[uint]bool)
	for _, id := range onCallUserIDs {
		onCallMap[id] = true
	}

	for _, u := range *users {
		if len(onCallUserIDs) > 0 && !onCallMap[u.ID] {
			continue
		}
		if u.PhoneNumber == nil || *u.PhoneNumber == "" {
			continue
		}

		w.logger.WithField("phone", *u.PhoneNumber).Info("mock sending Voice alert")
		// In production, we would use the Twilio Go SDK or HTTP request here
		// twilioClient.MakeCall("mock_twilio_from", *u.PhoneNumber, "<twiml>url</twiml>")
	}
}

func (w *worker) sendDatadogAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for datadog alert")
		return
	}

	if team.DatadogWebhookURL == nil || *team.DatadogWebhookURL == "" {
		w.logger.Debug("datadog webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	// Format as Datadog Event Payload
	payload := map[string]interface{}{
		"title":            "Opsway Incident: " + monitorName,
		"text":             fmt.Sprintf("An incident has been created in Opsway.\n\nIssue: %s", incident.Title),
		"alert_type":       "error",
		"source_type_name": "Opsway",
		"tags":             []string{"source:opsway", fmt.Sprintf("monitor:%s", monitorName)},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.DatadogWebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create datadog request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send datadog alert")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("datadog alert returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("datadog alert sent successfully")
	}
}

func (w *worker) sendNewRelicAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for new relic alert")
		return
	}

	if team.NewRelicWebhookURL == nil || *team.NewRelicWebhookURL == "" {
		w.logger.Debug("new relic webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	payload := map[string]interface{}{
		"eventType":     "OpswayIncident",
		"monitorName":   monitorName,
		"incidentTitle": incident.Title,
		"description":   "An incident has been created in Opsway.",
		"timestamp":     time.Now().Unix(),
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.NewRelicWebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create new relic request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send new relic alert")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("new relic alert returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("new relic alert sent successfully")
	}
}

func (w *worker) sendMicrosoftTeamsAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for microsoft teams alert")
		return
	}

	if team.MicrosoftTeamsWebhookURL == nil || *team.MicrosoftTeamsWebhookURL == "" {
		w.logger.Debug("microsoft teams webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/incidents", w.config.ApplicationURL)

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "FF0000",
		"summary":    "Incident Alert",
		"sections": []map[string]interface{}{
			{
				"activityTitle":    "🚨 Incident Alert: " + monitorName,
				"activitySubtitle": incident.Title,
				"text":             fmt.Sprintf("[View details and manage this incident on the dashboard](%s)", dashboardURL),
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.MicrosoftTeamsWebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create microsoft teams request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send microsoft teams alert")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("microsoft teams alert returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("microsoft teams alert sent successfully")
	}
}

func (w *worker) sendGenericWebhookAlert(ctx context.Context, incident *entities.Incident) {
	team, err := w.teamService.GetByID(ctx, incident.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team for generic webhook alert")
		return
	}

	if team.WebhookURL == nil || *team.WebhookURL == "" {
		w.logger.Debug("generic webhook not configured for team")
		return
	}

	monitorName := "Unknown Monitor / Heartbeat"
	if incident.MonitorID != nil {
		mon, err := w.monitorSvc.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && mon != nil {
			monitorName = mon.Name
		}
	} else if incident.HeartbeatID != nil {
		monitorName = "Heartbeat Monitor"
	}

	payload := map[string]interface{}{
		"event":         "incident_alert",
		"incidentId":    incident.ID,
		"incidentTitle": incident.Title,
		"monitorName":   monitorName,
		"teamId":        incident.TeamID,
		"timestamp":     time.Now().Unix(),
		"resolved":      incident.Resolved,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *team.WebhookURL, bytes.NewReader(body))
	if err != nil {
		w.logger.WithError(err).Error("failed to create generic webhook request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("failed to send generic webhook alert")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		w.logger.Errorf("generic webhook alert returned error status: %d", resp.StatusCode)
	} else {
		w.logger.Info("generic webhook alert sent successfully")
	}
}
