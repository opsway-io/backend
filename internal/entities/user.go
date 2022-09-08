package entities

import (
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/utils/pointer"
)

type User struct {
	ID                  uint
	Name                string `gorm:"not null"`
	DisplayName         *string
	Email               string `gorm:"uniqueIndex"`
	PasswordHash        *string
	Teams               []Team `gorm:"many2many:team_users;constraint:OnDelete:CASCADE"`
	MaintenanceComments []MaintenanceComment
	IncidentComments    []IncidentComment
	CreatedAt           time.Time `gorm:"index"`
	UpdatedAt           time.Time `gorm:"index"`
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
	err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password))

	return err == nil
}
