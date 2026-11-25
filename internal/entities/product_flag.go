package entities

import "time"

// ProductVisibility controls who can see and access a suite product.
type ProductVisibility string

const (
	// ProductVisibilityDisabled turns the product off for all users, including platform owners.
	ProductVisibilityDisabled ProductVisibility = "disabled"
	// ProductVisibilityOwnerOnly restricts the product to platform owners (ideal for active development).
	ProductVisibilityOwnerOnly ProductVisibility = "owner_only"
	// ProductVisibilityTargeted enables the product for platform owners plus explicitly allowlisted users or teams.
	ProductVisibilityTargeted ProductVisibility = "targeted"
	// ProductVisibilityEnabled makes the product available to all users.
	ProductVisibilityEnabled ProductVisibility = "enabled"
)

// ProductFeatureFlag stores global and targeted visibility settings for a suite product.
type ProductFeatureFlag struct {
	ID             uint              `gorm:"primaryKey" json:"id"`
	Product        ProductSlug       `gorm:"uniqueIndex;not null" json:"product"`
	Visibility     ProductVisibility `gorm:"not null;default:'enabled'" json:"visibility"`
	AllowedUserIDs []uint            `gorm:"serializer:json" json:"allowedUserIds"`
	AllowedTeamIDs []uint            `gorm:"serializer:json" json:"allowedTeamIds"`
	Description    string            `gorm:"type:text" json:"description"`
	UpdatedBy      *uint             `json:"updatedBy,omitempty"`
	CreatedAt      time.Time         `gorm:"index" json:"createdAt"`
	UpdatedAt      time.Time         `gorm:"index" json:"updatedAt"`
}

func (ProductFeatureFlag) TableName() string {
	return "product_feature_flags"
}

// AllProducts returns the canonical list of toggleable suite products.
func AllProducts() []ProductSlug {
	return []ProductSlug{
		ProductMonitoring,
		ProductLogSearch,
		ProductManagedAgent,
		ProductPantomath,
	}
}

func IsValidProduct(p ProductSlug) bool {
	for _, candidate := range AllProducts() {
		if candidate == p {
			return true
		}
	}
	return false
}

func IsValidVisibility(v ProductVisibility) bool {
	switch v {
	case ProductVisibilityDisabled,
		ProductVisibilityOwnerOnly,
		ProductVisibilityTargeted,
		ProductVisibilityEnabled:
		return true
	default:
		return false
	}
}
