package entities

import (
	"time"

	"github.com/lib/pq"
)

type Maintenance struct {
	ID     uint `json:"id"`
	TeamID uint `gorm:"index;not null" json:"teamId"`

	Title       string               `gorm:"index;not null" json:"title"`
	Description *string              `json:"description"`
	Settings    MaintenanceSettings  `gorm:"constraint:OnDelete:CASCADE" json:"settings"`
	Comments    []MaintenanceComment `gorm:"constraint:OnDelete:CASCADE" json:"comments"`
	Monitors    []Monitor            `gorm:"many2many:maintenance_monitors;constraint:OnDelete:CASCADE" json:"monitors"`

	UpdatedAt time.Time `gorm:"index" json:"updatedAt"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

func (Maintenance) TableName() string {
	return "maintenance"
}

type MaintenanceSettings struct {
	ID            uint `json:"id"`
	MaintenanceID uint `gorm:"index;not null" json:"maintenanceId"`

	StartAt  time.Time       `gorm:"index;not null" json:"startAt"`
	EndAt    time.Time       `gorm:"index;not null" json:"endAt"`
	Tags              *pq.StringArray `gorm:"type:text[]" json:"tags"`
	Notified          bool            `gorm:"default:false" json:"notified"`
	Reminded          bool            `gorm:"default:false" json:"reminded"`
	ConcludedNotified bool            `gorm:"default:false" json:"concludedNotified"`

	UpdatedAt time.Time `gorm:"index" json:"updatedAt"`
}

func (MaintenanceSettings) TableName() string {
	return "maintenance_settings"
}

type MaintenanceComment struct {
	ID            uint `json:"id"`
	UserID        uint `gorm:"index;not null" json:"userId"`
	MaintenanceID uint `gorm:"index;not null" json:"maintenanceId"`

	Content string `json:"content"`

	CreatedAt time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time `gorm:"index" json:"updatedAt"`
}

func (MaintenanceComment) TableName() string {
	return "maintenance_comments"
}
