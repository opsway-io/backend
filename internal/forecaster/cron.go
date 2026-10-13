package forecaster

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Start(ctx context.Context) error
}

type worker struct {
	monitorService monitor.Service
	eventService   event.Service
	logger         *logrus.Entry
	interval       time.Duration
}

func NewWorker(monitorService monitor.Service, eventService event.Service, logger *logrus.Entry, interval time.Duration) Worker {
	return &worker{
		monitorService: monitorService,
		eventService:   eventService,
		logger:         logger.WithField("component", "forecaster_worker"),
		interval:       interval,
	}
}

func (w *worker) Start(ctx context.Context) error {
	w.logger.Info("Starting forecaster cron worker...")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run once immediately
	w.triggerTraining(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping forecaster cron worker...")
			return nil
		case <-ticker.C:
			w.triggerTraining(ctx)
		}
	}
}

func (w *worker) triggerTraining(ctx context.Context) {
	w.logger.Info("Triggering daily forecaster training for all monitors")
	monitors, err := w.monitorService.GetMonitorsByStates(ctx, []entities.MonitorState{entities.MonitorStateActive})
	if err != nil {
		w.logger.WithError(err).Error("failed to get monitors for training")
		return
	}

	for _, m := range *monitors {
		e := events.TrainForecasterEvent{
			MonitorID: m.ID,
		}
		if err := w.eventService.Publish(e); err != nil {
			w.logger.WithError(err).Error("failed to publish TrainForecasterEvent")
		}
	}
}
