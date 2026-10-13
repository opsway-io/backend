package alerting

import (
	"context"
	"fmt"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/notification/webhook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Channel names as used by the dashboard
const (
	ChannelEmail          = "email"
	ChannelSlack          = "slack"
	ChannelMicrosoftTeams = "microsoft_teams"
	ChannelWebhook        = "webhook"
	ChannelDiscord        = "discord"
	ChannelTelegram       = "telegram"
	ChannelSMS            = "sms"
	ChannelVoice          = "voice"
	ChannelDatadog        = "datadog"
	ChannelNewRelic       = "new_relic"
)

var ErrChannelNotConfigured = errors.New("channel is not configured for team")

// Alert is everything a channel needs to render a notification.
type Alert struct {
	Rule       *entities.AlertRule
	Incident   *entities.Incident
	Team       *entities.Team
	Recipients []Recipient
	Link       string
}

func (a *Alert) text() string {
	desc := ""
	if a.Incident.Description != nil && *a.Incident.Description != a.Incident.Title {
		desc = " - " + *a.Incident.Description
	}

	return fmt.Sprintf(
		":rotating_light: [opsway] New incident #%d on monitor %d: %s%s (rule %q) %s",
		a.Incident.ID, a.Incident.MonitorID, a.Incident.Title, desc, a.Rule.Name, a.Link,
	)
}

// genericPayload is posted to generic, Datadog and New Relic webhooks.
type genericPayload struct {
	Event    string               `json:"event"`
	Text     string               `json:"text"`
	Link     string               `json:"link"`
	Rule     genericPayloadRule   `json:"rule"`
	Incident genericPayloadIncdnt `json:"incident"`
}

type genericPayloadRule struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Condition string `json:"condition"`
}

type genericPayloadIncdnt struct {
	ID          uint    `json:"id"`
	TeamID      uint    `json:"teamId"`
	MonitorID   uint    `json:"monitorId"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

// Recipient is a team member that receives alert notifications.
type Recipient struct {
	UserID uint
	Name   string
	Email  string
}

// Dispatcher delivers the notifications of a triggered rule. It is best-effort: failures are logged.
type Dispatcher interface {
	Dispatch(ctx context.Context, rule *entities.AlertRule, incident *entities.Incident, channels []string)
}

type DispatcherImpl struct {
	logger  *logrus.Entry
	db      *gorm.DB
	email   email.Sender
	webhook webhook.Sender
	appURL  string
}

func NewDispatcher(logger *logrus.Entry, db *gorm.DB, emailSender email.Sender, webhookSender webhook.Sender, appURL string) Dispatcher {
	return &DispatcherImpl{logger: logger, db: db, email: emailSender, webhook: webhookSender, appURL: appURL}
}

func (d *DispatcherImpl) Dispatch(ctx context.Context, rule *entities.AlertRule, incident *entities.Incident, channels []string) {
	l := d.logger.WithFields(logrus.Fields{"alert_rule_id": rule.ID, "incident_id": incident.ID})

	var team entities.Team
	if err := d.db.WithContext(ctx).First(&team, incident.TeamID).Error; err != nil {
		l.WithError(err).Error("failed to get team for alert")

		return
	}

	var recipients []Recipient
	if err := d.db.WithContext(ctx).
		Table("users").
		Select("users.id AS user_id, COALESCE(users.display_name, users.name) AS name, users.email").
		Joins("INNER JOIN team_users tu ON tu.user_id = users.id").
		Where("tu.team_id = ?", incident.TeamID).
		Scan(&recipients).Error; err != nil {
		l.WithError(err).Error("failed to get alert recipients")
	}

	alert := &Alert{
		Rule:       rule,
		Incident:   incident,
		Team:       &team,
		Recipients: recipients,
		Link:       fmt.Sprintf("%s/incidents/%d", d.appURL, incident.ID),
	}

	for _, ch := range channels {
		if err := d.dispatchChannel(ctx, ch, alert); err != nil {
			l.WithError(err).WithField("channel", ch).Warn("failed to deliver alert")
		}
	}
}

func (d *DispatcherImpl) dispatchChannel(ctx context.Context, channel string, alert *Alert) error {
	t := alert.Team

	switch channel {
	case ChannelEmail:
		return d.sendEmail(ctx, alert)
	case ChannelSlack:
		return d.post(ctx, t.SlackWebhookURL, map[string]string{"text": alert.text()})
	case ChannelMicrosoftTeams:
		return d.post(ctx, t.MicrosoftTeamsWebhookURL, map[string]string{"text": alert.text()})
	case ChannelDiscord:
		return d.post(ctx, t.DiscordWebhookURL, map[string]string{"content": alert.text()})
	case ChannelWebhook:
		return d.post(ctx, t.WebhookURL, newGenericPayload(alert))
	case ChannelDatadog:
		return d.post(ctx, t.DatadogWebhookURL, newGenericPayload(alert))
	case ChannelNewRelic:
		return d.post(ctx, t.NewRelicWebhookURL, newGenericPayload(alert))
	case ChannelTelegram, ChannelSMS, ChannelVoice:
		// No provider credentials (bot token / SMS gateway) are configured yet
		return errors.Wrapf(ErrChannelNotConfigured, "no %s provider available", channel)
	default:
		return fmt.Errorf("unknown channel %q", channel)
	}
}

func (d *DispatcherImpl) post(ctx context.Context, url *string, payload any) error {
	if url == nil || *url == "" {
		return ErrChannelNotConfigured
	}

	return d.webhook.Post(ctx, *url, payload)
}

func (d *DispatcherImpl) sendEmail(ctx context.Context, alert *Alert) error {
	if len(alert.Recipients) == 0 {
		return ErrChannelNotConfigured
	}

	desc := ""
	if alert.Incident.Description != nil {
		desc = *alert.Incident.Description
	}

	var firstErr error
	for _, r := range alert.Recipients {
		if err := d.email.Send(ctx, r.Name, r.Email, &templates.IncidentAlertTemplate{
			Name:                r.Name,
			RuleName:            alert.Rule.Name,
			IncidentTitle:       alert.Incident.Title,
			IncidentDescription: desc,
			IncidentLink:        alert.Link,
		}); err != nil && firstErr == nil {
			firstErr = errors.Wrapf(err, "failed to send alert email to user %d", r.UserID)
		}
	}

	return firstErr
}

func newGenericPayload(a *Alert) genericPayload {
	return genericPayload{
		Event: "incident.created",
		Text:  a.text(),
		Link:  a.Link,
		Rule: genericPayloadRule{
			ID:        a.Rule.ID,
			Name:      a.Rule.Name,
			Condition: a.Rule.Condition,
		},
		Incident: genericPayloadIncdnt{
			ID:          a.Incident.ID,
			TeamID:      a.Incident.TeamID,
			MonitorID:   a.Incident.MonitorID,
			Title:       a.Incident.Title,
			Description: a.Incident.Description,
			CreatedAt:   a.Incident.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
}
