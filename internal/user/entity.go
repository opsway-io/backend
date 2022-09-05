package user

import (
	"time"

	"github.com/opsway-io/backend/internal/team"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	Name         string `gorm:"not null"`
	DisplayName  string
	Email        string `gorm:"uniqueIndex:idx_email"`
	PasswordHash string
	TeamID       int `gorm:"not null,index:idx_team_id"`
	Team         team.Team
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to generate password hash")
	}

	u.PasswordHash = string(hash)

	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))

	return err == nil
}
