package entities

import (
	"time"
)

type HttpResult struct {
	ID        uint
	Result    string
	MonitorID int       `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (HttpResult) TableName() string {
	return "HttpResults"
}
