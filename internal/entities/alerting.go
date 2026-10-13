package entities

import (
	"time"
)

type AlertRule struct {
	ID        uint      `json:"id"`
	TeamID    uint      `gorm:"index;not null" json:"teamId"`
	Name      string    `json:"name"`
	Condition string    `json:"condition"`
	Channels  string    `gorm:"type:jsonb;default:'[]'" json:"channels"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
