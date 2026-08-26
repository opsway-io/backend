package events

import (
	"github.com/opsway-io/backend/internal/entities"
)

const (
	EventTypeIncidentPostMortemPublished EventType = "incident:post_mortem_published"
)

type IncidentPostMortemPublishedEvent struct {
	Incident *entities.Incident `json:"incident"`
}

func (e IncidentPostMortemPublishedEvent) Name() string {
	return string(EventTypeIncidentPostMortemPublished)
}
