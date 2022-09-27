package entities

import (
	"time"
)

type ProbeResult struct {
	ID        uint
	Body      *[]byte   `gorm:"type:bytea"`
	MonitorID int       `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (ProbeResult) TableName() string {
	return "ProbeResults"
}
