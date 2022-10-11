package entities

import (
	"time"
)

type HttpResult struct {
	ID         uint
	StatusCode uint64 `gorm:"index; not null"`
	Timing     string
	TLS        string
	MonitorID  uint64    `gorm:"index;not null"`
	CreatedAt  time.Time `gorm:"index"`
}

func (HttpResult) TableName() string {
	return "http"
}
