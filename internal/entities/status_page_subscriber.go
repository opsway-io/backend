package entities

import (
	"time"
)

type StatusPageSubscriber struct {
	ID           uint      `gorm:"primaryKey"`
	StatusPageID uint      `gorm:"index;not null"`
	Email        string    `gorm:"index;not null"`
	Verified     bool      `gorm:"default:false"`
	Token        string    `gorm:"uniqueIndex;not null"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (StatusPageSubscriber) TableName() string {
	return "status_page_subscribers"
}
