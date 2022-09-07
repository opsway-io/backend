package maintenance

import (
	"time"

	"github.com/lib/pq"
)

type Maintenance struct {
	ID          uint
	Title       string `gorm:"index;not null"`
	Description *string
	TeamID      uint      `gorm:"index;not null"`
	Settings    Settings  `gorm:"constraint:OnDelete:CASCADE"`
	Comments    []Comment `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time `gorm:"index"`
}

func (Maintenance) TableName() string {
	return "maintenance"
}

type Settings struct {
	ID            uint
	StartAt       time.Time       `gorm:"index;not null"`
	EndAt         time.Time       `gorm:"index;not null"`
	Tags          *pq.StringArray `gorm:"type:text[]"`
	MaintenanceID uint            `gorm:"index;not null"`
	CreatedAt     time.Time       `gorm:"index"`
	UpdatedAt     time.Time       `gorm:"index"`
}

func (Settings) TableName() string {
	return "maintenance_settings"
}

type Comment struct {
	ID            uint
	Content       string
	UserID        uint      `gorm:"index;not null"`
	MaintenanceID uint      `gorm:"index;not null"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time `gorm:"index"`
}

func (Comment) TableName() string {
	return "maintenance_comments"
}
