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
	Avatar      *string
	Users       []User        `gorm:"many2many:team_users"`
	Roles       []TeamRole    `gorm:"constraint:OnDelete:CASCADE"`
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
	if ok := checkTeamNameFormat(t.Name); !ok {
		return ErrIllegalTeamNameFormat
	}

	return nil
}

func checkTeamNameFormat(name string) bool {
	return NameFormatRegex.MatchString(name)
}

type Role string

const (
	TeamRoleAdmin  Role = "admin"
	TeamRoleMember Role = "member"
)

type TeamRole struct {
	UserID uint `gorm:"primaryKey"`
	TeamID uint `gorm:"primaryKey"`
	Role   Role
}

func (TeamRole) TableName() string {
	return "team_roles"
}
