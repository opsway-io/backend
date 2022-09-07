package incident

import (
	"time"
)

type Incident struct {
	ID          uint
	Title       string `gorm:"index;not null"`
	Description *string
	TeamID      uint      `gorm:"index;not null"`
	MonitorID   uint      `gorm:"index;not null"`
	Comments    []Comment `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time `gorm:"index"`
}

func (Incident) TableName() string {
	return "incident"
}

type Comment struct {
	ID         uint
	Content    string
	UserID     uint      `gorm:"index;not null"`
	IncidentID uint      `gorm:"index;not null"`
	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time `gorm:"index"`
}

func (Comment) TableName() string {
	return "incident_comments"
}
