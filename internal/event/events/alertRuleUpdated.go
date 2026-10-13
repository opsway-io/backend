package events

import (
	"github.com/opsway-io/backend/internal/entities"
)

const (
	EventTypeAlertRuleUpdated EventType = "alert_rule:updated"
)

type AlertRuleUpdatedEvent struct {
	Rule *entities.AlertRule `json:"rule"`
}

func (e AlertRuleUpdatedEvent) Name() string {
	return string(EventTypeAlertRuleUpdated)
}
