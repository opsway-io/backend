package check

import (
	"time"
)

type Check struct {
	ID         uint64
	StatusCode uint64 `gorm:"index; not null"`
	Timing     string
	TLS        string
	MonitorID  uint64    `gorm:"index;not null"`
	CreatedAt  time.Time `gorm:"index"`
}

func (Check) TableName() string {
	return "checks"
}
