package entities

import (
	"time"

	"github.com/lib/pq"
)

type Maintenance struct {
	ID     uint
	TeamID uint `gorm:"index;not null"`

	Title       string `gorm:"index;not null"`
	Description *string
	Settings    MaintenanceSettings  `gorm:"constraint:OnDelete:CASCADE"`
	Comments    []MaintenanceComment `gorm:"constraint:OnDelete:CASCADE"`

	// Monitors affected by the maintenance window. Empty means all monitors of the team.
	Monitors []Monitor `gorm:"many2many:maintenance_monitors;constraint:OnDelete:CASCADE"`

	UpdatedAt time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"index"`
}

func (Maintenance) TableName() string {
	return "maintenance"
}

type MaintenanceStatus string

const (
	MaintenanceStatusScheduled MaintenanceStatus = "SCHEDULED"
	MaintenanceStatusActive    MaintenanceStatus = "ACTIVE"
	MaintenanceStatusEnded     MaintenanceStatus = "ENDED"
)

// StatusAt derives the status of the maintenance window at the given time.
// The window is active in the half-open interval [StartAt, EndAt).
func (m *Maintenance) StatusAt(t time.Time) MaintenanceStatus {
	switch {
	case t.Before(m.Settings.StartAt):
		return MaintenanceStatusScheduled
	case t.Before(m.Settings.EndAt):
		return MaintenanceStatusActive
	default:
		return MaintenanceStatusEnded
	}
}

// AppliesToMonitor reports whether the window covers the given monitor.
// A window without explicit monitors covers all monitors of the team.
func (m *Maintenance) AppliesToMonitor(monitorID uint) bool {
	if len(m.Monitors) == 0 {
		return true
	}

	for _, mon := range m.Monitors {
		if mon.ID == monitorID {
			return true
		}
	}

	return false
}

type MaintenanceSettings struct {
	ID            uint
	MaintenanceID uint `gorm:"index;not null"`

	StartAt time.Time       `gorm:"index;not null"`
	EndAt   time.Time       `gorm:"index;not null"`
	Tags    *pq.StringArray `gorm:"type:text[]"`

	UpdatedAt time.Time `gorm:"index"`
}

func (MaintenanceSettings) TableName() string {
	return "maintenance_settings"
}

type MaintenanceComment struct {
	ID            uint
	UserID        uint `gorm:"index;not null"`
	MaintenanceID uint `gorm:"index;not null"`

	Content string

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (MaintenanceComment) TableName() string {
	return "maintenance_comments"
}
