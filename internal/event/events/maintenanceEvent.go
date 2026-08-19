package events

import (
	"github.com/opsway-io/backend/internal/entities"
)

const (
	EventTypeMaintenance EventType = "maintenance:event"
)

type MaintenanceEvent struct {
	Maintenance *entities.Maintenance `json:"maintenance"`
	Action      string                `json:"action"` // "created", "updated", "completed"
}

func (e MaintenanceEvent) Name() string {
	return string(EventTypeMaintenance)
}
