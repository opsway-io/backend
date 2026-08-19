package incident

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/llm"
	"github.com/sirupsen/logrus"
)

type RCAWorker interface {
	Start(ctx context.Context) error
}

type rcaWorker struct {
	eventService event.Service
	incidentSvc  Service
	llmClient    llm.Client
	logger       *logrus.Entry
}

func NewRCAWorker(
	eventService event.Service,
	incidentSvc Service,
	llmClient llm.Client,
	logger *logrus.Entry,
) RCAWorker {
	return &rcaWorker{
		eventService: eventService,
		incidentSvc:  incidentSvc,
		llmClient:    llmClient,
		logger:       logger.WithField("component", "rca_worker"),
	}
}

func (w *rcaWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting RCA worker...")

	messages, err := w.eventService.Subscribe(ctx, string(events.EventTypeIncidentCreated))
	if err != nil {
		return fmt.Errorf("failed to subscribe to incident stream: %w", err)
	}

	for msg := range messages {
		w.processMessage(ctx, msg.Payload)
		msg.Ack()
	}

	return nil
}

func (w *rcaWorker) processMessage(ctx context.Context, payload []byte) {
	var ev events.IncidentCreatedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		w.logger.WithError(err).Error("failed to unmarshal event")
		return
	}

	incident := ev.Incident
	if incident == nil || incident.Resolved {
		return
	}

	// Generate prompt based on incident details
	prompt := fmt.Sprintf("Incident Title: %s\n", incident.Title)
	if incident.Description != nil {
		prompt += fmt.Sprintf("Description: %s\n", *incident.Description)
	}
	prompt += "Provide a brief Root Cause Analysis."

	rca, err := w.llmClient.GenerateRCA(ctx, prompt)
	if err != nil {
		w.logger.WithError(err).Error("failed to generate RCA")
		return
	}

	// Update the incident with the RCA
	incident.RootCauseAnalysis = &rca
	if err := w.incidentSvc.Update(ctx, incident); err != nil {
		w.logger.WithError(err).Error("failed to save RCA to incident")
	} else {
		w.logger.Infof("Successfully attached AI RCA to incident %d", incident.ID)
	}
}
