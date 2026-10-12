package entities

import (
	"time"
)

type StatusPageSubscriber struct {
	ID           uint   `gorm:"primaryKey"`
	StatusPageID uint   `gorm:"index;not null"`
	Email        string `gorm:"index;not null"`
	Channel      string `gorm:"default:'EMAIL';not null"` // EMAIL, WEBHOOK, SMS, SLACK
	Destination  string `gorm:"default:'';not null"`      // webhook URL, phone number, or slack webhook URL
	Verified     bool   `gorm:"default:false"`
	Token        string `gorm:"uniqueIndex;not null"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (StatusPageSubscriber) TableName() string {
	return "status_page_subscribers"
}
