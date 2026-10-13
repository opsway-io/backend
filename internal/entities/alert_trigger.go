package entities

import (
	"time"
)

type AlertTrigger struct {
	ID          uint      `json:"id"`
	TeamID      uint      `gorm:"index;not null" json:"teamId"`
	AlertRuleID uint      `gorm:"index;not null" json:"alertRuleId"`
	IncidentID  *uint     `gorm:"index" json:"incidentId"`
	Channels    string    `gorm:"type:jsonb;default:'[]'" json:"channels"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	AlertRule *AlertRule `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Incident  *Incident  `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

func (AlertTrigger) TableName() string {
	return "alert_triggers"
}
