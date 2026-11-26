package entities

import (
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var (
	/*
		The regex allows:
		- lowercase letters
		- numbers
		- dashes
		it does not allow:
		- uppercase letters
		- underscores
		- spaces
		- special characters
		- empty string
		- two dashes in a row
		- a dash at the beginning or end
	*/
	NameFormatRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	ErrIllegalTeamNameFormat = errors.New("illegal name format")
)

type PaymentPlan string

const (
	PaymentPlanFree       PaymentPlan = "FREE"
	PaymentPlanPro        PaymentPlan = "PRO"
	PaymentPlanTeam       PaymentPlan = "TEAM" // Legacy alias for PRO
	PaymentPlanEnterprise PaymentPlan = "ENTERPRISE"
)

// NormalizePaymentPlan converts legacy "TEAM" to canonical "PRO".
func NormalizePaymentPlan(p PaymentPlan) PaymentPlan {
	switch NormalizeTier(string(p)) {
	case EntitlementTierEnterprise:
		return PaymentPlanEnterprise
	case EntitlementTierPro:
		return PaymentPlanPro
	default:
		return PaymentPlanFree
	}
}

type Team struct {
	ID                       uint
	Name                     string      `gorm:"uniqueIndex;not null"`
	DisplayName              *string     `gorm:"index"`
	PaymentPlan              PaymentPlan `gorm:"default:FREE;not null"`
	StripeCustomerID         *string     `gorm:"index"`
	AllowOverage             bool        `gorm:"not null;default:false" json:"allowOverage"`
	OverageSpendCapCents     *int        `json:"overageSpendCapCents,omitempty"`
	SlackWebhookURL          *string     `gorm:"type:text"`
	DiscordWebhookURL        *string     `gorm:"type:text"`
	TelegramChatID           *string     `gorm:"type:text"`
	DatadogWebhookURL        *string     `gorm:"type:text"`
	NewRelicWebhookURL       *string     `gorm:"type:text"`
	MicrosoftTeamsWebhookURL *string     `gorm:"type:text"`
	WebhookURL               *string     `gorm:"type:text"`
	HasAvatar                bool

	Users       []User           `gorm:"many2many:team_users;constraint:OnDelete:CASCADE;"`
	Monitors    []Monitor        `gorm:"constraint:OnDelete:CASCADE"`
	Maintenance []Maintenance    `gorm:"constraint:OnDelete:CASCADE"`
	Incidents   []Incident       `gorm:"constraint:OnDelete:CASCADE"`
	Changelogs  []Changelog      `gorm:"constraint:OnDelete:CASCADE"`
	Invitations []TeamInvitation `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Team) TableName() string {
	return "teams"
}

func (t *Team) GetMonitorLimit() int {
	return DefaultMonthlyQuota(ProductMonitoring, NormalizeTier(string(t.PaymentPlan)))
}

func (t *Team) GetTeamMemberLimit() int {
	return DefaultTeamMemberLimit(NormalizeTier(string(t.PaymentPlan)))
}

func (t *Team) GetStatusPageLimit() int {
	return DefaultStatusPageLimit(NormalizeTier(string(t.PaymentPlan)))
}

func (t *Team) GetManagedInvestigationLimit() int {
	return DefaultMonthlyQuota(ProductManagedAgent, NormalizeTier(string(t.PaymentPlan)))
}

func (t *Team) GetAgentTokenLimitPerRun() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 32000
	case PaymentPlanPro, PaymentPlanTeam:
		return 100000
	case PaymentPlanEnterprise:
		return 250000
	default:
		return 32000
	}
}

func (t *Team) GetAgentMonthlyTokenLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 100000
	case PaymentPlanPro, PaymentPlanTeam:
		return 2000000
	case PaymentPlanEnterprise:
		return -1
	default:
		return 100000
	}
}

func (t *Team) GetAgentTriggerRuleLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 3
	case PaymentPlanPro, PaymentPlanTeam:
		return 25
	case PaymentPlanEnterprise:
		return -1
	default:
		return 3
	}
}

func (t *Team) GetMCPGatewayRateLimitPerMin() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 60
	case PaymentPlanPro, PaymentPlanTeam:
		return 300
	case PaymentPlanEnterprise:
		return 1000
	default:
		return 60
	}
}

func (t *Team) GetPantomathRepoLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 3
	case PaymentPlanPro, PaymentPlanTeam:
		return 25
	case PaymentPlanEnterprise:
		return -1
	default:
		return 3
	}
}

func (t *Team) GetPantomathVectorChunkLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 25000
	case PaymentPlanPro, PaymentPlanTeam:
		return 250000
	case PaymentPlanEnterprise:
		return -1
	default:
		return 25000
	}
}

func (t *Team) GetMinMonitorFrequencySeconds() uint64 {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 60
	case PaymentPlanPro, PaymentPlanTeam:
		return 30
	case PaymentPlanEnterprise:
		return 10
	default:
		return 60
	}
}

func (t *Team) GetMaxMonitorSteps() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 3
	case PaymentPlanPro, PaymentPlanTeam:
		return 10
	case PaymentPlanEnterprise:
		return 25
	default:
		return 3
	}
}

func (t *Team) GetMaxConcurrentInvestigations() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 1
	case PaymentPlanPro, PaymentPlanTeam:
		return 3
	case PaymentPlanEnterprise:
		return 10
	default:
		return 1
	}
}

func (t *Team) MaxEntitlementTier() EntitlementTier {
	switch t.PaymentPlan {
	case PaymentPlanEnterprise:
		return EntitlementTierEnterprise
	case PaymentPlanPro, PaymentPlanTeam:
		return EntitlementTierPro
	default:
		return EntitlementTierFree
	}
}

func EntitlementTierRank(tier EntitlementTier) int {
	switch tier {
	case EntitlementTierEnterprise:
		return 3
	case EntitlementTierPro:
		return 2
	default:
		return 1
	}
}

func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if ok := checkTeamNameFormat(t.Name); !ok {
		return ErrIllegalTeamNameFormat
	}

	return nil
}

func (t *Team) BeforeUpdate(tx *gorm.DB) (err error) {
	if t.Name == "" {
		return nil
	}

	if ok := checkTeamNameFormat(t.Name); !ok {
		return ErrIllegalTeamNameFormat
	}

	return nil
}

func checkTeamNameFormat(name string) bool {
	return NameFormatRegex.MatchString(name)
}

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "OWNER"
	TeamRoleAdmin  TeamRole = "ADMIN"
	TeamRoleMember TeamRole = "MEMBER"
)

func (r TeamRole) IsValid() bool {
	switch r {
	case TeamRoleOwner, TeamRoleAdmin, TeamRoleMember:
		return true
	default:
		return false
	}
}

func TeamRoleFrom(source any) (TeamRole, error) {
	s, ok := source.(string)
	if !ok {
		return "", errors.New("invalid team role type, must be string")
	}

	switch s {
	case "OWNER":
		return TeamRoleOwner, nil
	case "ADMIN":
		return TeamRoleAdmin, nil
	case "MEMBER":
		return TeamRoleMember, nil
	default:
		return "", errors.New("invalid team role")
	}
}

type TeamUser struct {
	UserID uint `gorm:"primaryKey;autoIncrement:false"`
	TeamID uint `gorm:"primaryKey;autoIncrement:false"`

	Role TeamRole `gorm:"index"`

	UpdatedAt time.Time
	CreatedAt time.Time
}

func (TeamUser) TableName() string {
	return "team_users"
}

func (tu *TeamUser) BeforeSave(tx *gorm.DB) error {
	if !tu.Role.IsValid() {
		return errors.New("invalid team role")
	}
	return nil
}

type TeamInvitation struct {
	ID        uint     `gorm:"primaryKey"`
	TeamID    uint     `gorm:"index;not null"`
	Email     string   `gorm:"index;not null"`
	Role      TeamRole `gorm:"not null"`
	Token     string   `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TeamInvitation) TableName() string {
	return "team_invitations"
}

func (ti *TeamInvitation) BeforeSave(tx *gorm.DB) error {
	if !ti.Role.IsValid() {
		return errors.New("invalid team role")
	}
	return nil
}
