package entities

import (
	"time"
)

type Incident struct {
	ID          uint
	Title       string `gorm:"index;not null"`
	Description *string
	TeamID      uint              `gorm:"index;not null"`
	MonitorID   uint              `gorm:"index;not null"`
	Comments    []IncidentComment `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time         `gorm:"index"`
	UpdatedAt   time.Time         `gorm:"index"`
}

func (Incident) TableName() string {
	return "incident"
}

type IncidentComment struct {
	ID         uint
	Content    string
	UserID     uint      `gorm:"index;not null"`
	IncidentID uint      `gorm:"index;not null"`
	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time `gorm:"index"`
}

func (IncidentComment) TableName() string {
	return "incident_comments"
}
