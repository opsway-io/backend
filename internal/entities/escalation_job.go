package entities

import "time"

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusProcessed JobStatus = "processed"
	JobStatusFailed    JobStatus = "failed"
)

type EscalationJob struct {
	ID           uint      `gorm:"primarykey"`
	IncidentID   uint      `gorm:"index"`
	TeamID       uint      `gorm:"index"`
	TargetTier   int       `gorm:"not null"`
	ScheduledFor time.Time `gorm:"index;not null"`
	Status       JobStatus `gorm:"index;not null;default:'pending'"`
	CreatedAt    time.Time `gorm:"index"`
	UpdatedAt    time.Time `gorm:"index"`
}

func (EscalationJob) TableName() string {
	return "escalation_jobs"
}
