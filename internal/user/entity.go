package user

import (
	"time"

	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/maintenance"
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
	TeamID              *uint `gorm:"index;not null"` // TODO: support multiple teams
	MaintenanceComments []maintenance.Comment
	IncidentComments    []incident.Comment
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
