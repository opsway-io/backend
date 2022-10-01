package entities

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/utils/pointer"
)

type User struct {
	ID                  uint
	Name                string  `gorm:"index;not null"`
	DisplayName         *string `gorm:"index"`
	Email               string  `gorm:"uniqueIndex"`
	Avatar              *string
	PasswordHash        *string
	Roles               []TeamRole           `gorm:"constraint:OnDelete:CASCADE"`
	Teams               []Team               `gorm:"many2many:team_users;constraint:OnDelete:CASCADE"`
	MaintenanceComments []MaintenanceComment `gorm:"constraint:OnDelete:CASCADE"`
	IncidentComments    []IncidentComment    `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt           time.Time            `gorm:"index"`
	UpdatedAt           time.Time            `gorm:"index"`
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
	err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password))

	return err == nil
}
