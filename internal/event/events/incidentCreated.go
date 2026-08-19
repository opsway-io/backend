package events

import (
	"github.com/opsway-io/backend/internal/entities"
)

type EventType string

const (
	EventTypeIncidentCreated EventType = "incident:created"
)

type IncidentCreatedEvent struct {
	Incident *entities.Incident `json:"incident"`
}

func (e IncidentCreatedEvent) Name() string {
	return string(EventTypeIncidentCreated)
}
