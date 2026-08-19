package heartbeats

import (
	"context"
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

		desc := "Heartbeat missed its check-in window"
		hbId := hb.ID

		incident := entities.Incident{
			TeamID:      hb.TeamID,
			HeartbeatID: &hbId,
			Title:       "Heartbeat Down",
			Description: &desc,
		}

		if err := w.incidentService.Create(ctx, &[]entities.Incident{incident}); err != nil {
			l.WithError(err).Error("failed to create incident for expired heartbeat")
		}
	}
}
