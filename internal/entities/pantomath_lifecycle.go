package entities

import (
	"time"
)

type PantomathLifecycleStatus string

const (
	PantomathStatusActive          PantomathLifecycleStatus = "ACTIVE"
	PantomathStatusScheduledDelete PantomathLifecycleStatus = "SCHEDULED_DELETE"
	PantomathStatusDeleted         PantomathLifecycleStatus = "DELETED"
)

// PantomathProductLifecycle tracks which indexed Pantomath products are actively searchable
// vs. locked and scheduled for deletion (7 days) after a plan downgrade.
type PantomathProductLifecycle struct {
	ID        uint                     `gorm:"primaryKey" json:"id"`
	TeamID    uint                     `gorm:"uniqueIndex:idx_panto_lifecycle;not null" json:"teamId"`
	Product   string                   `gorm:"uniqueIndex:idx_panto_lifecycle;size:128;not null" json:"product"`
	Status    PantomathLifecycleStatus `gorm:"size:32;not null;default:'ACTIVE'" json:"status"`
	DeleteAt  *time.Time               `json:"deleteAt,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

func (PantomathProductLifecycle) TableName() string {
	return "pantomath_product_lifecycles"
}
