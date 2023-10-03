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

type Team struct {
	ID          uint
	Name        string  `gorm:"uniqueIndex;not null"`
	DisplayName *string `gorm:"index"`
	PaymentPlan string  `gorm:"default:FREE"`
	StripeKey   *string
	HasAvatar   bool
	Users       []User        `gorm:"many2many:team_users"`
	Monitors    []Monitor     `gorm:"constraint:OnDelete:CASCADE"`
	Maintenance []Maintenance `gorm:"constraint:OnDelete:CASCADE"`
	Incidents   []Incident    `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time     `gorm:"index"`
	UpdatedAt   time.Time     `gorm:"index"`
}

func (Team) TableName() string {
	return "teams"
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
	UserID    uint     `gorm:"primaryKey;autoIncrement:false"`
	TeamID    uint     `gorm:"primaryKey;autoIncrement:false"`
	Role      TeamRole `gorm:"index"`
	CreatedAt time.Time
}

func (TeamUser) TableName() string {
	return "team_users"
}
