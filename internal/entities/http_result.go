package entities

import (
	"time"
)

type HttpResult struct {
	ID        uint
	Body      *[]byte   `gorm:"type:bytea"`
	MonitorID int       `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (HttpResult) TableName() string {
	return "HttpResults"
}
