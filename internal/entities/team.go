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
	PaymentPlanTeam       PaymentPlan = "TEAM"
	PaymentPlanEnterprise PaymentPlan = "ENTERPRISE"
)

type Team struct {
	ID               uint
	Name             string      `gorm:"uniqueIndex;not null"`
	DisplayName      *string     `gorm:"index"`
	PaymentPlan      PaymentPlan `gorm:"default:FREE;not null"`
	StripeCustomerID *string     `gorm:"index"`
	SlackWebhookURL    *string     `gorm:"type:text"`
	DiscordWebhookURL  *string     `gorm:"type:text"`
	TelegramChatID     *string     `gorm:"type:text"`
	DatadogWebhookURL  *string     `gorm:"type:text"`
	NewRelicWebhookURL *string     `gorm:"type:text"`
	HasAvatar          bool

	Users       []User        `gorm:"many2many:team_users;constraint:OnDelete:CASCADE;"`
	Monitors    []Monitor     `gorm:"constraint:OnDelete:CASCADE"`
	Maintenance []Maintenance `gorm:"constraint:OnDelete:CASCADE"`
	Incidents   []Incident    `gorm:"constraint:OnDelete:CASCADE"`
	Changelogs  []Changelog   `gorm:"constraint:OnDelete:CASCADE"`
	Invitations []TeamInvitation `gorm:"constraint:OnDelete:CASCADE"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (Team) TableName() string {
	return "teams"
}

func (t *Team) GetMonitorLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 5
	case PaymentPlanTeam:
		return 50
	case PaymentPlanEnterprise:
		return -1
	default:
		return 5
	}
}

func (t *Team) GetTeamMemberLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 3
	case PaymentPlanTeam:
		return 5
	case PaymentPlanEnterprise:
		return -1
	default:
		return 1
	}
}

func (t *Team) GetStatusPageLimit() int {
	switch t.PaymentPlan {
	case PaymentPlanFree:
		return 1
	case PaymentPlanTeam:
		return 5
	case PaymentPlanEnterprise:
		return -1
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
	ID        uint      `gorm:"primaryKey"`
	TeamID    uint      `gorm:"index;not null"`
	Email     string    `gorm:"index;not null"`
	Role      TeamRole  `gorm:"not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
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
