package entities

import (
	"time"
)

type HeartbeatStatus string

const (
	HeartbeatStatusUp     HeartbeatStatus = "UP"
	HeartbeatStatusDown   HeartbeatStatus = "DOWN"
	HeartbeatStatusPaused HeartbeatStatus = "PAUSED"
)

type Heartbeat struct {
	ID        uint            `json:"id"`
	TeamID    uint            `gorm:"index;not null" json:"teamId"`
	Name      string          `json:"name"`
	Status    HeartbeatStatus `gorm:"default:'PAUSED'" json:"status"`
	Interval  time.Duration   `json:"interval"`
	Grace     time.Duration   `json:"grace"`
	LastPing  *time.Time      `json:"lastPing"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
