package entities

import (
	"time"
)

type PaymentPlan string

const (
	PaymentPlanFree       PaymentPlan = "FREE"
	PaymentPlanTeam       PaymentPlan = "TEAM"
	PaymentPlanEnterprise PaymentPlan = "ENTERPRISE"
)

type Billing struct {
	ID          uint
	PaymentPlan PaymentPlan `gorm:"default:FREE"`
	StripeID    *string
	TeamID      uint      `gorm:"index;not null"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time `gorm:"index"`
}

func (Billing) TableName() string {
	return "billings"
}
