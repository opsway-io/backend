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

	UpdatedAt time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"index"`
}

func (Maintenance) TableName() string {
	return "maintenance"
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
