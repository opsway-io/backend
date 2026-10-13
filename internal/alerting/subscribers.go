package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/notification/webhook"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type IncidentEvent string

const (
	IncidentEventOpened   IncidentEvent = "opened"
	IncidentEventResolved IncidentEvent = "resolved"
)

// SubscriberNotifier informs verified status page subscribers about status page visible incidents.
type SubscriberNotifier interface {
	// Notify sends notifications asynchronously and best-effort; it never blocks on delivery.
	Notify(incident *entities.Incident, event IncidentEvent)
}

type SubscriberNotifierImpl struct {
	logger  *logrus.Entry
	db      *gorm.DB
	email   email.Sender
	webhook webhook.Sender
}

// NewSubscriberNotifier creates a notifier. webhookSender should be a webhook.NewPublicSender, since
// subscriber destinations are supplied by anonymous users.
func NewSubscriberNotifier(logger *logrus.Entry, db *gorm.DB, emailSender email.Sender, webhookSender webhook.Sender) SubscriberNotifier {
	return &SubscriberNotifierImpl{logger: logger, db: db, email: emailSender, webhook: webhookSender}
}

type pageSubscriber struct {
	entities.StatusPageSubscriber
	PageName   string `gorm:"column:page_name"`
	PageDomain string `gorm:"column:page_domain"`
}

func (n *SubscriberNotifierImpl) Notify(incident *entities.Incident, event IncidentEvent) {
	if incident == nil || !incident.IsStatusPageVisible {
		return
	}

	in := *incident

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
		defer cancel()

		n.notify(ctx, &in, event)
	}()
}

func (n *SubscriberNotifierImpl) notify(ctx context.Context, in *entities.Incident, event IncidentEvent) {
	l := n.logger.WithFields(logrus.Fields{"incident_id": in.ID, "event": event})

	var subs []pageSubscriber
	if err := n.db.WithContext(ctx).
		Table("status_page_subscribers AS s").
		Select("s.*, p.name AS page_name, p.domain AS page_domain").
		Joins("INNER JOIN status_pages p ON p.id = s.status_page_id").
		Where("p.team_id = ? AND s.verified_at IS NOT NULL", in.TeamID).
		Where(
			"? = ANY(p.monitor_ids) OR EXISTS (SELECT 1 FROM status_page_groups g WHERE g.status_page_id = p.id AND ? = ANY(g.monitor_ids))",
			in.MonitorID, in.MonitorID,
		).
		Scan(&subs).Error; err != nil {
		l.WithError(err).Error("failed to get status page subscribers")

		return
	}

	for i := range subs {
		if err := n.send(ctx, &subs[i], in, event); err != nil {
			l.WithError(err).WithField("subscriber_id", subs[i].ID).Warn("failed to notify status page subscriber")
		}
	}

	if len(subs) > 0 {
		l.WithField("subscribers", len(subs)).Info("notified status page subscribers")
	}
}

func (n *SubscriberNotifierImpl) send(ctx context.Context, sub *pageSubscriber, in *entities.Incident, event IncidentEvent) error {
	link := fmt.Sprintf("https://%s", sub.PageDomain)
	text := subscriberText(sub.PageName, in, event, link)

	switch sub.Channel {
	case entities.StatusPageSubscriberChannelEmail:
		return n.email.Send(ctx, sub.Email, sub.Email, &subscriberIncidentTemplate{
			PageName: sub.PageName,
			Text:     text,
			Link:     link,
			Event:    event,
		})
	case entities.StatusPageSubscriberChannelSlack:
		return n.webhook.Post(ctx, sub.Destination, map[string]string{"text": text})
	case entities.StatusPageSubscriberChannelWebhook:
		return n.webhook.Post(ctx, sub.Destination, map[string]any{
			"event":      "incident." + string(event),
			"text":       text,
			"statusPage": map[string]string{"name": sub.PageName, "url": link},
			"incident": map[string]any{
				"id":        in.ID,
				"title":     in.Title,
				"createdAt": in.CreatedAt.Format(time.RFC3339),
				"resolved":  in.Resolved,
			},
		})
	default:
		return fmt.Errorf("unknown subscriber channel %q", sub.Channel)
	}
}

func subscriberText(pageName string, in *entities.Incident, event IncidentEvent, link string) string {
	if event == IncidentEventResolved {
		return fmt.Sprintf("[%s] Resolved: %s. All affected services are operational again. %s", pageName, in.Title, link)
	}

	return fmt.Sprintf("[%s] Investigating: %s. We are aware of the issue and are working on it. %s", pageName, in.Title, link)
}

type subscriberIncidentTemplate struct {
	templates.BaseTemplate

	PageName string
	Text     string
	Link     string
	Event    IncidentEvent
}

func (t *subscriberIncidentTemplate) Subject() string {
	if t.Event == IncidentEventResolved {
		return fmt.Sprintf("[%s] Incident resolved", t.PageName)
	}

	return fmt.Sprintf("[%s] New incident", t.PageName)
}

func (t *subscriberIncidentTemplate) HTML() string {
	return t.Render(`<p>{{text}}</p><p><a href="{{link}}">View status page</a></p>`, map[string]any{
		"title": t.Subject(),
		"text":  t.Text,
		"link":  t.Link,
	})
}

func (t *subscriberIncidentTemplate) PlainText() string {
	return t.Text
}
