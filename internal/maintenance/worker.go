package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Start(ctx context.Context) error
}

type worker struct {
	logger            *logrus.Entry
	maintenance       Service
	monitor           monitor.Service
	statusPageService statuspage.Service
	emailSender       email.Sender
	interval          time.Duration
	statusPageBaseURL string
}

func NewWorker(logger *logrus.Entry, maintenance Service, monitor monitor.Service, statusPageService statuspage.Service, emailSender email.Sender, interval time.Duration, statusPageBaseURL string) Worker {
	return &worker{
		logger:            logger.WithField("component", "maintenance-worker"),
		maintenance:       maintenance,
		monitor:           monitor,
		statusPageService: statusPageService,
		emailSender:       emailSender,
		interval:          interval,
		statusPageBaseURL: statusPageBaseURL,
	}
}

func (w *worker) Start(ctx context.Context) error {
	w.logger.Info("starting maintenance worker")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping maintenance worker")
			return nil
		case <-ticker.C:
			w.processMaintenanceWindows(ctx)
		}
	}
}

func (w *worker) processMaintenanceWindows(ctx context.Context) {
	now := time.Now()

	unnotifiedMaintenances, err := w.maintenance.GetUnnotified(ctx, now)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch unnotified maintenance windows")
	} else {
		for _, m := range *unnotifiedMaintenances {
			w.notifySubscribers(ctx, &m)
			
			m.Settings.Notified = true
			if err := w.maintenance.Update(ctx, &m); err != nil {
				w.logger.WithError(err).Error("failed to mark maintenance as notified")
			}
		}
	}

	activeMaintenances, err := w.maintenance.GetActive(ctx, now)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch active maintenance windows")
		return
	}

	teamsUnderMaintenance := make(map[uint]bool)
	teamsWithFullMaintenance := make(map[uint]bool)
	monitorsUnderMaintenance := make(map[uint]bool)
	
	for _, m := range *activeMaintenances {
		teamsUnderMaintenance[m.TeamID] = true

		if len(m.Monitors) == 0 {
			teamsWithFullMaintenance[m.TeamID] = true
		} else {
			for _, mon := range m.Monitors {
				monitorsUnderMaintenance[mon.ID] = true
			}
		}
	}

	monitors, err := w.monitor.GetMonitorsByStates(ctx, []entities.MonitorState{
		entities.MonitorStateActive,
		entities.MonitorStateMaintenance,
	})
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch monitors for maintenance processing")
		return
	}

	for _, m := range *monitors {
		inMaintenance := teamsWithFullMaintenance[m.TeamID] || monitorsUnderMaintenance[m.ID]

		if m.State == entities.MonitorStateActive && inMaintenance {
			w.logger.WithFields(logrus.Fields{
				"monitor_id": m.ID,
				"team_id":    m.TeamID,
			}).Info("setting monitor state to maintenance")
			
			if err := w.monitor.SetState(ctx, m.TeamID, m.ID, entities.MonitorStateMaintenance); err != nil {
				w.logger.WithError(err).Error("failed to set monitor state to maintenance")
			}
		} else if m.State == entities.MonitorStateMaintenance && !inMaintenance {
			w.logger.WithFields(logrus.Fields{
				"monitor_id": m.ID,
				"team_id":    m.TeamID,
			}).Info("setting monitor state back to active")
			
			if err := w.monitor.SetState(ctx, m.TeamID, m.ID, entities.MonitorStateActive); err != nil {
				w.logger.WithError(err).Error("failed to set monitor state back to active")
			}
		}
	}
}

func (w *worker) notifySubscribers(ctx context.Context, m *entities.Maintenance) {
	// 1. Get all status pages for this team
	statusPages, err := w.statusPageService.GetByTeamID(ctx, m.TeamID)
	if err != nil {
		w.logger.WithError(err).Error("failed to fetch status pages for notification")
		return
	}

	// Determine if this maintenance applies to specific monitors
	maintenanceMonitors := make(map[uint]bool)
	isFullMaintenance := len(m.Monitors) == 0
	for _, mon := range m.Monitors {
		maintenanceMonitors[mon.ID] = true
	}

	for _, sp := range statusPages {
		// Check if this status page is affected by the maintenance
		isAffected := isFullMaintenance
		if !isAffected {
			for _, spMon := range sp.Monitors {
				if maintenanceMonitors[spMon.ID] {
					isAffected = true
					break
				}
			}
		}

		if !isAffected {
			continue
		}

		// 2. Get verified subscribers for each status page
		subs, err := w.statusPageService.GetVerifiedSubscribers(ctx, sp.ID)
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch subscribers")
			continue
		}

		// 3. Dispatch email to each subscriber
		statusPageURL := fmt.Sprintf("%s/%s", w.statusPageBaseURL, sp.Domain)
		for _, sub := range subs {
			unsubscribeURL := fmt.Sprintf("%s/%s/subscribe/%s", w.statusPageBaseURL, sp.Domain, sub.Token)
			err := w.emailSender.Send(ctx, "", sub.Email, &templates.MaintenanceAlertTemplate{
				StatusPageName:   sp.Name,
				MaintenanceTitle: m.Title,
				StatusPageURL:    statusPageURL,
				UnsubscribeURL:   unsubscribeURL,
			})
			if err != nil {
				w.logger.WithError(err).Error("failed to dispatch maintenance email")
			}

			w.logger.WithFields(logrus.Fields{
				"email":          sub.Email,
				"status_page_id": sp.ID,
				"maintenance":    m.Title,
			}).Info("Dispatched maintenance email to subscriber")
		}
	}
}
