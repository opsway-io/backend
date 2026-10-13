package heartbeats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Start(ctx context.Context) error
}

type worker struct {
	service         Service
	incidentService incident.Service
	logger          *logrus.Entry
	interval        time.Duration
}

func NewWorker(service Service, incidentService incident.Service, logger *logrus.Entry, interval time.Duration) Worker {
	return &worker{
		service:         service,
		incidentService: incidentService,
		logger:          logger.WithField("component", "heartbeat_worker"),
		interval:        interval,
	}
}

func (w *worker) Start(ctx context.Context) error {
	w.logger.Info("Starting heartbeat cron worker...")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping heartbeat cron worker...")
			return nil
		case <-ticker.C:
			w.processExpiredHeartbeats(ctx)
		}
	}
}

func FormatHeartbeatFailureDescription(hb entities.Heartbeat) string {
	var sb strings.Builder

	hbName := hb.Name
	if hbName == "" {
		hbName = fmt.Sprintf("Heartbeat #%d", hb.ID)
	}

	sb.WriteString(fmt.Sprintf("Heartbeat \"%s\" (ID: %d) missed its expected check-in window.\n\n", hbName, hb.ID))
	sb.WriteString("=== Heartbeat Configuration ===\n")
	sb.WriteString(fmt.Sprintf("• Configured Interval: %v\n", hb.Interval))
	sb.WriteString(fmt.Sprintf("• Grace Period:        %v\n", hb.Grace))
	sb.WriteString(fmt.Sprintf("• Allowed Window:      %v\n", hb.Interval+hb.Grace))

	sb.WriteString("\n=== Status & Timing ===\n")
	if hb.LastPing != nil {
		elapsed := time.Since(*hb.LastPing).Round(time.Second)
		deadline := hb.LastPing.Add(hb.Interval + hb.Grace)
		overdueBy := time.Since(deadline).Round(time.Second)
		if overdueBy < 0 {
			overdueBy = 0
		}
		sb.WriteString(fmt.Sprintf("• Last Successful Ping: %s (%v ago)\n", hb.LastPing.UTC().Format(time.RFC3339), elapsed))
		sb.WriteString(fmt.Sprintf("• Expected Deadline:    %s\n", deadline.UTC().Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("• Overdue By:           %v\n", overdueBy))
	} else {
		sb.WriteString("• Last Successful Ping: None (no check-in received since creation)\n")
	}

	sb.WriteString("\n=== Ping Endpoint & Verification ===\n")
	sb.WriteString("To send a check-in ping manually or test host reachability, execute:\n")
	sb.WriteString(fmt.Sprintf("  curl -fsS -m 10 --retry 3 https://api.opsway.eu/v1/heartbeats/%d/ping\n", hb.ID))

	sb.WriteString("\n=== Investigation Checklist ===\n")
	sb.WriteString("1. Confirm the scheduled task (cron job, systemd timer, Celery/worker process) is active on host.\n")
	sb.WriteString("2. Inspect job execution logs for failures, fatal errors, unhandled exceptions, or panics.\n")
	sb.WriteString("3. Check if the task is taking longer than usual (exceeding interval + grace threshold).\n")
	sb.WriteString("4. Verify outbound network connectivity and DNS resolution to https://api.opsway.eu.\n")

	return strings.TrimSpace(sb.String())
}

func (w *worker) processExpiredHeartbeats(ctx context.Context) {
	heartbeats, err := w.service.GetExpiredHeartbeats(ctx)
	if err != nil {
		w.logger.WithError(err).Error("failed to get expired heartbeats")
		return
	}

	for _, hb := range heartbeats {
		l := w.logger.WithFields(logrus.Fields{
			"heartbeat_id": hb.ID,
			"team_id":      hb.TeamID,
		})

		l.Info("Heartbeat expired, marking as DOWN and triggering incident")

		if err := w.service.MarkAsDown(ctx, hb.ID); err != nil {
			l.WithError(err).Error("failed to mark heartbeat as DOWN")
			continue
		}

		desc := FormatHeartbeatFailureDescription(hb)
		hbId := hb.ID

		title := "Heartbeat Down"
		if hb.Name != "" {
			title = fmt.Sprintf("Heartbeat Down: %s", hb.Name)
		}

		incident := entities.Incident{
			TeamID:      hb.TeamID,
			HeartbeatID: &hbId,
			Title:       title,
			Description: &desc,
		}

		if err := w.incidentService.Create(ctx, &[]entities.Incident{incident}); err != nil {
			l.WithError(err).Error("failed to create incident for expired heartbeat")
		}
	}
}
