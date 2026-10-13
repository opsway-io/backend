package entities

import "time"

// AlertRule decides which incidents trigger notifications on which channels.
// Condition is "*" (any incident) or a value matched against the incident title/description
// (e.g. an assertion source such as STATUS_CODE). Channels is a JSON encoded array of channel names.
type AlertRule struct {
	ID        uint
	TeamID    uint   `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	Condition string `gorm:"not null;default:'*'"`
	Channels  string `gorm:"not null;default:'[]'"`
	Enabled   bool   `gorm:"index;not null;default:true"`

	Triggers []AlertTrigger `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// AlertTrigger records that an alert rule fired, optionally for an incident.
type AlertTrigger struct {
	ID          uint
	TeamID      uint   `gorm:"index;not null"`
	AlertRuleID uint   `gorm:"index;not null"`
	IncidentID  *uint  `gorm:"index"`
	Channels    string `gorm:"not null;default:'[]'"`

	Incident *Incident `gorm:"constraint:OnDelete:SET NULL"`

	CreatedAt time.Time `gorm:"index"`
}

func (AlertTrigger) TableName() string {
	return "alert_triggers"
}
