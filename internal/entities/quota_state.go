package entities

import (
	"time"
)

type QuotaEnforcementState string

const (
	QuotaStateNormal         QuotaEnforcementState = "NORMAL"
	QuotaStateWarning        QuotaEnforcementState = "WARNING"
	QuotaStateGrace          QuotaEnforcementState = "GRACE"
	QuotaStateOverageBilling QuotaEnforcementState = "OVERAGE_BILLING"
	QuotaStateThrottled      QuotaEnforcementState = "THROTTLED"
)

// QuotaNotificationState tracks per-billing-cycle enforcement state and deduplicated
// notification timestamps for each team's product quota.
type QuotaNotificationState struct {
	ID                     uint                  `gorm:"primaryKey" json:"id"`
	TeamID                 uint                  `gorm:"uniqueIndex:idx_quota_state_cycle;not null" json:"teamId"`
	Product                ProductSlug           `gorm:"uniqueIndex:idx_quota_state_cycle;size:32;not null" json:"product"`
	QuotaKey               string                `gorm:"uniqueIndex:idx_quota_state_cycle;size:64;not null" json:"quotaKey"`
	BillingCycleStart      string                `gorm:"uniqueIndex:idx_quota_state_cycle;size:10;not null" json:"billingCycleStart"` // YYYY-MM-DD
	State                  QuotaEnforcementState `gorm:"size:32;not null;default:'NORMAL'" json:"state"`
	UsedValue              float64               `gorm:"not null;default:0" json:"usedValue"`
	LimitValue             float64               `gorm:"not null;default:0" json:"limitValue"`
	EstimatedOverageCents  int                   `gorm:"not null;default:0" json:"estimatedOverageCents"`
	TopConsumerHint        string                `gorm:"type:text" json:"topConsumerHint,omitempty"`
	GraceStartedAt         *time.Time            `json:"graceStartedAt,omitempty"`
	GraceExpiresAt         *time.Time            `json:"graceExpiresAt,omitempty"`
	NotifiedWarningAt      *time.Time            `json:"notifiedWarningAt,omitempty"`
	NotifiedGraceAt        *time.Time            `json:"notifiedGraceAt,omitempty"`
	NotifiedOverageAt      *time.Time            `json:"notifiedOverageAt,omitempty"`
	NotifiedThrottledAt    *time.Time            `json:"notifiedThrottledAt,omitempty"`
	NotifiedRecoveredAt    *time.Time            `json:"notifiedRecoveredAt,omitempty"`
	CreatedAt              time.Time             `json:"createdAt"`
	UpdatedAt              time.Time             `json:"updatedAt"`
}

func (QuotaNotificationState) TableName() string {
	return "quota_notification_states"
}
