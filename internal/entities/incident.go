package entities

import (
	"time"
)

type Incident struct {
	ID                 uint
	TeamID             uint  `gorm:"index;not null"`
	MonitorID          *uint `gorm:"index"`
	MonitorAssertionID *uint `gorm:"uniqueIndex:unresolved_monitor_incident"`
	HeartbeatID        *uint `gorm:"uniqueIndex:unresolved_heartbeat_incident"`
	Resolved           bool  `gorm:"not null;default:false"`
	Acknowledged       bool  `gorm:"not null;default:false"`
	AcknowledgedBy     *uint `gorm:"index"`
	AcknowledgedAt     *time.Time

	Title             string `gorm:"index;not null"`
	Description       *string
	Occurrences       int `gorm:"not null;default:1"`
	RootCauseAnalysis *string
	RootCauseNotified bool                 `gorm:"not null;default:false"`
	IsStatusPageVisible bool               `gorm:"not null;default:false"`
	Comments          []IncidentComment    `gorm:"constraint:OnDelete:CASCADE"`
	OccurrencesList   []IncidentOccurrence `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Incident) TableName() string {
	return "incidents"
}

type IncidentComment struct {
	ID         uint
	UserID     uint `gorm:"index;not null"`
	IncidentID uint `gorm:"index;not null"`

	Content string

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (IncidentComment) TableName() string {
	return "incident_comments"
}

type IncidentOccurrence struct {
	ID         uint
	IncidentID uint      `gorm:"index;not null"`
	CreatedAt  time.Time `gorm:"index"`
}

func (IncidentOccurrence) TableName() string {
	return "incident_occurrences"
}
