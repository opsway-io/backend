package entities

import "time"

type HeartbeatStatus string

const (
	HeartbeatStatusUp     HeartbeatStatus = "UP"
	HeartbeatStatusDown   HeartbeatStatus = "DOWN"
	HeartbeatStatusPaused HeartbeatStatus = "PAUSED"
)

type Heartbeat struct {
	ID     uint
	TeamID uint `gorm:"index;not null"`

	Name string `gorm:"not null"`

	// Expected time between pings, in minutes.
	Interval uint `gorm:"not null"`
	// Extra time allowed after the interval before the heartbeat is DOWN, in minutes.
	Grace uint `gorm:"not null;default:0"`

	Paused     bool `gorm:"not null;default:false"`
	LastPingAt *time.Time
	// DownSince is set by the periodic heartbeat checker when the deadline passed
	// without a ping, and cleared by the next ping.
	DownSince *time.Time `gorm:"index"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Heartbeat) TableName() string {
	return "heartbeats"
}

// DeadlineAt returns the time after which the heartbeat is considered DOWN when
// no further ping arrives. Before the first ping the creation time is used as reference.
func (h *Heartbeat) DeadlineAt() time.Time {
	ref := h.CreatedAt
	if h.LastPingAt != nil {
		ref = *h.LastPingAt
	}

	return ref.Add(time.Duration(h.Interval+h.Grace) * time.Minute)
}

// StatusAt derives the heartbeat status at the given time.
func (h *Heartbeat) StatusAt(t time.Time) HeartbeatStatus {
	if h.Paused {
		return HeartbeatStatusPaused
	}

	if t.After(h.DeadlineAt()) {
		return HeartbeatStatusDown
	}

	return HeartbeatStatusUp
}
