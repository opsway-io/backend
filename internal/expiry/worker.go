package expiry

import (
	"context"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Start(ctx context.Context) error
}

type worker struct {
	logger         *logrus.Entry
	monitorService monitor.Service
	checkService   check.Service
	teamService    team.Service
	emailSender    email.Sender
	interval       time.Duration
	applicationURL string
}

func NewWorker(
	logger *logrus.Entry,
	monitorService monitor.Service,
	checkService check.Service,
	teamService team.Service,
	emailSender email.Sender,
	interval time.Duration,
	applicationURL string,
) Worker {
	return &worker{
		logger:         logger.WithField("component", "expiry_worker"),
		monitorService: monitorService,
		checkService:   checkService,
		teamService:    teamService,
		emailSender:    emailSender,
		interval:       interval,
		applicationURL: applicationURL,
	}
}

func (w *worker) Start(ctx context.Context) error {
	w.logger.Info("starting expiry worker")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping expiry worker")
			return nil
		case <-ticker.C:
			w.processExpiries(ctx)
		}
	}
}

func (w *worker) processExpiries(ctx context.Context) {
	w.logger.Info("checking for SSL and Domain expiries")

	offset := 0
	limit := 1000

	for {
		monitors, err := w.monitorService.GetMonitorsAndSettings(ctx, &offset, &limit, nil)
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch monitors")
			return
		}

		if len(*monitors) == 0 {
			break
		}

		for _, m := range *monitors {
			if m.Settings.TLS.CheckExpiration != nil && *m.Settings.TLS.CheckExpiration {
				w.checkSSLExpiry(ctx, &m)
			}
			
			// We can check domain expiry similarly if we have domain expiry checking enabled.
			// For MVP, we will only do SSL if TLS.CheckExpiration is true.
			// If we wanted to add domain expiry, we could do it here.
		}

		offset += limit
	}
}

func (w *worker) checkSSLExpiry(ctx context.Context, m *monitor.MonitorAndSettings) {
	thresholdDays := uint(7)
	if m.Settings.TLS.ExpirationThresholdDays != nil {
		thresholdDays = *m.Settings.TLS.ExpirationThresholdDays
	}

	latestCheck, err := w.checkService.GetLatestByMonitorID(ctx, m.ID)
	if err != nil || latestCheck == nil || latestCheck.TLS == nil {
		return
	}

	daysRemaining := int(latestCheck.TLS.NotAfter.Sub(time.Now()).Hours() / 24)
	
	if daysRemaining <= int(thresholdDays) && daysRemaining >= 0 {
		// Needs notification
		if m.Settings.SslExpiryNotifiedAt == nil || time.Since(*m.Settings.SslExpiryNotifiedAt) > 24*time.Hour {
			w.notifySSLExpiry(ctx, m, daysRemaining, latestCheck.TLS.NotAfter)

			now := time.Now()
			m.Settings.SslExpiryNotifiedAt = &now
			
			// Update the monitor settings using monitorService
			err := w.monitorService.UpdateSettings(ctx, m.TeamID, m.ID, &m.Settings)
			if err != nil {
				w.logger.WithError(err).Error("failed to update ssl expiry notified at")
			}
		}
	}
}

func (w *worker) notifySSLExpiry(ctx context.Context, m *monitor.MonitorAndSettings, daysRemaining int, expirationDate time.Time) {
	offset := 0
	limit := 100
	query := ""
	users, err := w.teamService.GetUsersByID(ctx, m.TeamID, &offset, &limit, &query, nil)
	if err != nil {
		w.logger.WithError(err).Error("failed to get team users")
		return
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/monitors/%d", w.applicationURL, m.ID)

	for _, u := range *users {
		tpl := &templates.SSLExpiryTemplate{
			MonitorName:    m.Name,
			MonitorURL:     m.Settings.URL,
			DashboardURL:   dashboardURL,
			DaysRemaining:  daysRemaining,
			ExpirationDate: expirationDate.Format(time.RFC1123),
		}

		err = w.emailSender.Send(ctx, "", u.Email, tpl)
		if err != nil {
			w.logger.WithError(err).Error("failed to send ssl expiry alert email")
		}
	}
}
