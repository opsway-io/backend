package entities

import (
	"time"

	"github.com/lib/pq"
)

type Maintenance struct {
	ID          uint
	Title       string `gorm:"index;not null"`
	Description *string
	TeamID      uint                 `gorm:"index;not null"`
	Settings    MaintenanceSettings  `gorm:"constraint:OnDelete:CASCADE"`
	Comments    []MaintenanceComment `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time            `gorm:"index"`
	UpdatedAt   time.Time            `gorm:"index"`
}

func (Maintenance) TableName() string {
	return "maintenance"
}

type MaintenanceSettings struct {
	ID            uint
	StartAt       time.Time       `gorm:"index;not null"`
	EndAt         time.Time       `gorm:"index;not null"`
	Tags          *pq.StringArray `gorm:"type:text[]"`
	MaintenanceID uint            `gorm:"index;not null"`
	CreatedAt     time.Time       `gorm:"index"`
	UpdatedAt     time.Time       `gorm:"index"`
}

func (MaintenanceSettings) TableName() string {
	return "maintenance_settings"
}

type MaintenanceComment struct {
	ID            uint
	Content       string
	UserID        uint      `gorm:"index;not null"`
	MaintenanceID uint      `gorm:"index;not null"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time `gorm:"index"`
}

func (MaintenanceComment) TableName() string {
	return "maintenance_comments"
}
