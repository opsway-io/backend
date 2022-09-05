package team

import (
	"time"
)

type Team struct {
	ID        int
	Name      string `gorm:"not null"`
	Logo      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
