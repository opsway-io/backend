package entities

import (
	"time"
)

type EscalationPolicy struct {
	ID        uint
	TeamID    uint `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	
	// How many minutes to wait before escalating from Tier 1 to Tier 2
	EscalationTimeoutMinutes int `gorm:"not null;default:5"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (EscalationPolicy) TableName() string {
	return "escalation_policies"
}

type OnCallRotation struct {
	ID                 uint
	EscalationPolicyID uint `gorm:"index;not null"`
	UserID             uint `gorm:"index;not null"`
	
	// 1 for Primary, 2 for Secondary, etc.
	Tier int `gorm:"not null;default:1"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (OnCallRotation) TableName() string {
	return "on_call_rotations"
}
