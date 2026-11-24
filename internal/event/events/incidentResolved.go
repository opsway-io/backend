package events

import (
	"github.com/opsway-io/backend/internal/entities"
)

const (
	EventTypeIncidentResolved EventType = "incident:resolved"
)

type IncidentResolvedEvent struct {
	Incident *entities.Incident `json:"incident"`
}

func (e IncidentResolvedEvent) Name() string {
	return string(EventTypeIncidentResolved)
}
