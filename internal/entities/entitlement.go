package entities

import (
	"strings"
	"time"
)

type ProductSlug string

const (
	ProductMonitoring   ProductSlug = "monitoring"
	ProductLogSearch    ProductSlug = "log_search"
	ProductPantomath    ProductSlug = "pantomath"
	ProductManagedAgent ProductSlug = "managed_agent"
)

type EntitlementTier string

const (
	EntitlementTierFree       EntitlementTier = "FREE"
	EntitlementTierPro        EntitlementTier = "PRO"
	EntitlementTierEnterprise EntitlementTier = "ENTERPRISE"
)

type EntitlementSource string

const (
	EntitlementSourceSuite EntitlementSource = "SUITE"
	EntitlementSourceAddon EntitlementSource = "ADDON"
)

// NormalizeTier maps legacy ("TEAM") and canonical ("FREE", "PRO", "ENTERPRISE")
// tier strings into a valid EntitlementTier.
func NormalizeTier(raw string) EntitlementTier {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "ENTERPRISE":
		return EntitlementTierEnterprise
	case "PRO", "TEAM":
		return EntitlementTierPro
	default:
		return EntitlementTierFree
	}
}

// TierRank returns a numeric ordering for comparing tiers (FREE < PRO < ENTERPRISE).
func TierRank(t EntitlementTier) int {
	switch NormalizeTier(string(t)) {
	case EntitlementTierEnterprise:
		return 2
	case EntitlementTierPro:
		return 1
	default:
		return 0
	}
}

type TeamEntitlement struct {
	ID                       uint              `gorm:"primaryKey" json:"id"`
	TeamID                   uint              `gorm:"uniqueIndex:idx_team_product;not null" json:"teamId"`
	Product                  ProductSlug       `gorm:"uniqueIndex:idx_team_product;not null" json:"product"`
	Tier                     EntitlementTier   `gorm:"not null;default:'FREE'" json:"tier"`
	Source                   EntitlementSource `gorm:"not null;default:'SUITE'" json:"source"`
	Enabled                  bool              `gorm:"not null;default:true" json:"enabled"`
	StripeSubscriptionItemID *string           `json:"stripeSubscriptionItemId,omitempty"`
	MonthlyQuota             int               `gorm:"not null;default:0" json:"monthlyQuota"`
	CurrentUsage             int               `gorm:"not null;default:0" json:"currentUsage"`
	UsageResetAt             *time.Time        `json:"usageResetAt,omitempty"`
	CreatedAt                time.Time         `gorm:"index" json:"createdAt"`
	UpdatedAt                time.Time         `gorm:"index" json:"updatedAt"`
}

func (TeamEntitlement) TableName() string {
	return "team_entitlements"
}

func DefaultTeamMemberLimit(tier EntitlementTier) int {
	switch NormalizeTier(string(tier)) {
	case EntitlementTierPro:
		return 5
	case EntitlementTierEnterprise:
		return -1
	default:
		return 3
	}
}

func DefaultStatusPageLimit(tier EntitlementTier) int {
	switch NormalizeTier(string(tier)) {
	case EntitlementTierPro:
		return 5
	case EntitlementTierEnterprise:
		return -1
	default:
		return 1
	}
}

func DefaultMonthlyQuota(product ProductSlug, tier EntitlementTier) int {
	switch product {
	case ProductMonitoring:
		switch tier {
		case EntitlementTierFree:
			return 5
		case EntitlementTierPro:
			return 50
		case EntitlementTierEnterprise:
			return -1
		}
	case ProductLogSearch:
		switch tier {
		case EntitlementTierFree:
			return 1 // 1 GB / month
		case EntitlementTierPro:
			return 50 // 50 GB / month
		case EntitlementTierEnterprise:
			return -1
		}
	case ProductManagedAgent:
		switch tier {
		case EntitlementTierFree:
			return 3 // 3 trial investigations
		case EntitlementTierPro:
			return 30 // 30 managed investigations / month
		case EntitlementTierEnterprise:
			return 500
		}
	case ProductPantomath:
		switch tier {
		case EntitlementTierFree:
			return 1
		case EntitlementTierPro:
			return 5
		case EntitlementTierEnterprise:
			return -1
		}
	}
	return 0
}
