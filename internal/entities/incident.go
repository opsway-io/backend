package entities

import (
	"time"
)

type Incident struct {
	ID        uint
	TeamID    uint `gorm:"index;not null"`
	MonitorID uint `gorm:"index;not null"`
	// Only one unresolved incident may exist per assertion (partial unique index).
	MonitorAssertionID uint `gorm:"uniqueIndex:idx_incidents_unresolved_assertion,where:resolved = false;not null"`
	Resolved           bool `gorm:"index;not null;default:false"`
	ResolvedAt         *time.Time
	ResolvedByUserID   *uint
	ResolvedBy         *User `gorm:"foreignKey:ResolvedByUserID;constraint:OnDelete:SET NULL"`

	Acknowledged              bool `gorm:"index;not null;default:false"`
	AcknowledgedAt            *time.Time
	AcknowledgedByUserID      *uint
	AcknowledgedBy            *User `gorm:"foreignKey:AcknowledgedByUserID;constraint:OnDelete:SET NULL"`
	AcknowledgedByIntegration *string

	IsStatusPageVisible bool `gorm:"not null;default:false"`
	RootCauseAnalysis   *string

	Title       string `gorm:"index;not null"`
	Description *string
	Comments    []IncidentComment    `gorm:"constraint:OnDelete:CASCADE"`
	Occurrences []IncidentOccurrence `gorm:"constraint:OnDelete:CASCADE"`

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

// IncidentOccurrence records each time the failure behind an incident was observed.
type IncidentOccurrence struct {
	ID         uint
	IncidentID uint `gorm:"index;not null"`

	CreatedAt time.Time `gorm:"index"`
}

func (IncidentOccurrence) TableName() string {
	return "incident_occurrences"
}
