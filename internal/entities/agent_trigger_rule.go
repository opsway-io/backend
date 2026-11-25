package entities

import (
	"time"
)

type AgentTriggerRule struct {
	ID                       uint                     `gorm:"primaryKey" json:"id"`
	TeamID                   uint                     `gorm:"index;not null" json:"teamId"`
	Product                  ProductSlug              `gorm:"index;not null" json:"product"` // "monitoring", "log_search", "pantomath", "webhook"
	TriggerType              string                   `gorm:"index;not null" json:"triggerType"`
	Name                     string                   `gorm:"not null" json:"name"`
	Description              string                   `gorm:"type:text" json:"description,omitempty"`
	Enabled                  bool                     `gorm:"not null;default:true" json:"enabled"`
	ServiceFilter            string                   `json:"serviceFilter,omitempty"` // Optional service name / glob (empty = all services)
	Threshold                float64                  `gorm:"not null;default:0" json:"threshold"`
	WindowSeconds            int                      `gorm:"not null;default:300" json:"windowSeconds"`
	Pattern                  string                   `json:"pattern,omitempty"` // Optional regex/keyword for logs.pattern_match
	CooldownMinutes          int                      `gorm:"not null;default:15" json:"cooldownMinutes"`
	CreateIncident           bool                     `gorm:"not null;default:false" json:"createIncident"`
	RemediationLevelOverride RemediationAutonomyLevel `json:"remediationLevelOverride,omitempty"`
	CreatedAt                time.Time                `gorm:"index" json:"createdAt"`
	UpdatedAt                time.Time                `gorm:"index" json:"updatedAt"`
}

func (AgentTriggerRule) TableName() string {
	return "agent_trigger_rules"
}
