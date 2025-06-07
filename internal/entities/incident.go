package entities

import (
	"time"
)

type Incident struct {
	ID                 uint
	TeamID             uint `gorm:"index;not null"`
	MonitorID          uint `gorm:"index;not null"`
	MonitorAssertionID uint `gorm:"uniqueIndex:unresolved_incident;not null"`
	Resolved           bool `gorm:"uniqueIndex:unresolved_incident;not null;default:false"`

	Title       string `gorm:"index;not null"`
	Description *string
	Comments    []IncidentComment `gorm:"constraint:OnDelete:CASCADE"`

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
