package entities

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/utils/pointer"
)

type User struct {
	ID           uint
	Name         string  `gorm:"index;not null"`
	DisplayName  *string `gorm:"index"`
	Email        string  `gorm:"uniqueIndex"`
	HasAvatar    bool
	PasswordHash *string

	Teams               []Team               `gorm:"many2many:team_users"`
	MaintenanceComments []MaintenanceComment `gorm:"constraint:OnDelete:CASCADE"`
	IncidentComments    []IncidentComment    `gorm:"constraint:OnDelete:CASCADE"`
	ChangelogEntries    []ChangelogEntry     `gorm:"many2many:changelog_entry_authors"`

	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) SetEmail(email string) {
	u.Email = strings.ToLower(email)
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to generate password hash")
	}

	u.PasswordHash = pointer.StringPtr(string(hash))

	return nil
}

func (u *User) CheckPassword(password string) bool {
	if u.PasswordHash == nil {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password))

	return err == nil
}
