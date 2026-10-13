package entities

import "time"

type NotificationJob struct {
	ID           uint        `gorm:"primarykey"`
	IncidentID   uint        `gorm:"index"`
	UserID       uint        `gorm:"index"`
	Channel      ChannelType `gorm:"not null"`
	ScheduledFor time.Time   `gorm:"index;not null"`
	Status       JobStatus   `gorm:"index;not null;default:'pending'"`
	CreatedAt    time.Time   `gorm:"index"`
	UpdatedAt    time.Time   `gorm:"index"`
}

func (NotificationJob) TableName() string {
	return "notification_jobs"
}
