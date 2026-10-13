package entities

import "time"

type EscalationPolicy struct {
	ID                       uint
	TeamID                   uint   `gorm:"uniqueIndex;not null"`
	Name                     string `gorm:"not null"`
	EscalationTimeoutMinutes uint   `gorm:"not null;default:15"`
	EnforcedChannel          *string

	Rotations []EscalationRotation `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (EscalationPolicy) TableName() string {
	return "escalation_policies"
}

type EscalationRotation struct {
	ID                 uint
	EscalationPolicyID uint `gorm:"index;not null"`
	UserID             uint `gorm:"index;not null"`
	Tier               uint `gorm:"not null;default:1"`

	User *User `gorm:"constraint:OnDelete:CASCADE"`
}

func (EscalationRotation) TableName() string {
	return "escalation_rotations"
}
