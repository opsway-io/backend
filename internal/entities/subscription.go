package entities

import "time"

// TeamSubscription persists Stripe subscription lifecycle metadata for a team
// so the platform can track billing periods, cancel-at-period-end, and past_due states.
type TeamSubscription struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	TeamID               uint            `gorm:"uniqueIndex;not null" json:"teamId"`
	StripeCustomerID     string          `gorm:"index" json:"stripeCustomerId,omitempty"`
	StripeSubscriptionID string          `gorm:"uniqueIndex" json:"stripeSubscriptionId,omitempty"`
	BaseTier             EntitlementTier `gorm:"not null;default:'FREE'" json:"baseTier"`
	Status               string          `gorm:"not null;default:'active'" json:"status"`
	CurrentPeriodStart   *time.Time      `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd     *time.Time      `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd    bool            `gorm:"not null;default:false" json:"cancelAtPeriodEnd"`
	CreatedAt            time.Time       `gorm:"index" json:"createdAt"`
	UpdatedAt            time.Time       `gorm:"index" json:"updatedAt"`
}

func (TeamSubscription) TableName() string {
	return "team_subscriptions"
}

// StripeWebhookEvent deduplicates incoming Stripe webhook deliveries so retried
// events are processed idempotently.
type StripeWebhookEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     string    `gorm:"uniqueIndex;not null" json:"eventId"`
	EventType   string    `gorm:"not null" json:"eventType"`
	ProcessedAt time.Time `gorm:"index;not null" json:"processedAt"`
}

func (StripeWebhookEvent) TableName() string {
	return "stripe_webhook_events"
}
