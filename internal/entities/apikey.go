package entities

import (
	"time"
)

type APIKey struct {
	ID        uint   `gorm:"primaryKey"`
	TeamID    uint   `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	KeyHash   string `gorm:"not null"`
	CreatedAt time.Time
}

func (APIKey) TableName() string {
	return "api_keys"
}
