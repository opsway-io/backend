package incident

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/llm"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
)

type RCAWorker interface {
	Start(ctx context.Context) error
}

type GetHeartbeatFunc func(ctx context.Context, teamID uint, heartbeatID uint) (*entities.Heartbeat, error)

type rcaWorker struct {
	eventService      event.Service
	incidentSvc       Service
	monitorService    monitor.Service
	checkService      check.Service
	getHeartbeat      GetHeartbeatFunc
	llmClient         llm.Client
	logger            *logrus.Entry
}

func NewRCAWorker(
	eventService event.Service,
	incidentSvc Service,
	monitorService monitor.Service,
	checkService check.Service,
	getHeartbeat GetHeartbeatFunc,
	llmClient llm.Client,
	logger *logrus.Entry,
) RCAWorker {
	return &rcaWorker{
		eventService:      eventService,
		incidentSvc:       incidentSvc,
		monitorService:    monitorService,
		checkService:      checkService,
		getHeartbeat:      getHeartbeat,
		llmClient:         llmClient,
		logger:            logger.WithField("component", "rca_worker"),
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

	if incident.MonitorID != nil {
		m, err := w.monitorService.GetMonitorAndSettingsByTeamIDAndID(ctx, incident.TeamID, *incident.MonitorID)
		if err == nil && m != nil {
			prompt += fmt.Sprintf("Monitor URL: %s\nMonitor Method: %s\n", m.Settings.URL, m.Settings.Method)
		}
		
		offset := 0
		limit := 10
		checks, err := w.checkService.GetByTeamIDAndMonitorIDPaginated(ctx, incident.TeamID, *incident.MonitorID, &offset, &limit)
		if err == nil && checks != nil && len(*checks) > 0 {
			prompt += "Recent Checks:\n"
			for _, c := range *checks {
				prompt += fmt.Sprintf("- [%s] Status: %d, Timing: %v ms\n", c.CreatedAt.Format("2006-01-02T15:04:05Z"), c.StatusCode, c.Timing.Total.Milliseconds())
			}
		}
	}

	if incident.HeartbeatID != nil && w.getHeartbeat != nil {
		hb, err := w.getHeartbeat(ctx, incident.TeamID, *incident.HeartbeatID)
		if err == nil && hb != nil {
			prompt += fmt.Sprintf("Heartbeat Name: %s\nStatus: %s\nInterval: %v\nGrace: %v\n", hb.Name, hb.Status, hb.Interval, hb.Grace)
			if hb.LastPing != nil {
				prompt += fmt.Sprintf("Last Ping: %s\n", hb.LastPing.Format("2006-01-02T15:04:05Z"))
			} else {
				prompt += "Last Ping: Never\n"
			}
		}
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
