package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/webhook"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// newIncidentAlertHook returns the incident created hook that evaluates the team's alert rules
// and dispatches notifications (best-effort, asynchronously) for every new incident.
func newIncidentAlertHook(conf *Config, l *logrus.Logger, db *gorm.DB) incident.Option {
	emailSender := newEmailSender(conf)

	logger := l.WithField("module", "alerting")

	dispatcher := alerting.NewDispatcher(
		logger,
		db,
		emailSender,
		webhook.NewSender(webhook.DefaultTimeout),
		conf.Team.ApplicationURL,
	)

	alertingService := alerting.NewService(logger, alerting.NewRepository(db), dispatcher)

	return incident.WithCreatedHook(alertingService.OnIncidentCreated)
}

// newIncidentSubscriberHooks notifies verified status page subscribers when a status page visible
// incident opens (created visible, or made visible while unresolved) or is resolved.
func newIncidentSubscriberHooks(conf *Config, l *logrus.Logger, db *gorm.DB) []incident.Option {
	notifier := alerting.NewSubscriberNotifier(
		l.WithField("module", "status_page_notifier"),
		db,
		newEmailSender(conf),
		webhook.NewPublicSender(webhook.DefaultTimeout),
	)

	return []incident.Option{
		incident.WithCreatedHook(func(_ context.Context, in *entities.Incident) {
			notifier.Notify(in, alerting.IncidentEventOpened)
		}),
		incident.WithChangedHook(func(_ context.Context, in *entities.Incident, change incident.Change) {
			switch {
			case change == incident.ChangeResolved:
				notifier.Notify(in, alerting.IncidentEventResolved)
			case change == incident.ChangeMadeVisible && !in.Resolved:
				notifier.Notify(in, alerting.IncidentEventOpened)
			}
		}),
	}
}

func newEmailSender(conf *Config) email.Sender {
	if conf.Email.Debug {
		return email.NewConsoleSender()
	}

	return email.NewSendgridSender(conf.Email)
}

// proberIncidentOptions are the incident service hooks used by the prober, which creates incidents.
func proberIncidentOptions(conf *Config, l *logrus.Logger, db *gorm.DB) []incident.Option {
	return append([]incident.Option{newIncidentAlertHook(conf, l, db)}, newIncidentSubscriberHooks(conf, l, db)...)
}
