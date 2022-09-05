package team

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

	ErrIllegalNameFormat = errors.New("illegal name format")
)

type Team struct {
	ID          int
	Name        string `gorm:"uniqueIndex:idx_name"`
	DisplayName string
	Logo        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if ok := checkNameFormat(t.Name); !ok {
		return ErrIllegalNameFormat
	}

	return nil
}

func (t *Team) BeforeUpdate(tx *gorm.DB) (err error) {
	if ok := checkNameFormat(t.Name); !ok {
		return ErrIllegalNameFormat
	}

	return nil
}

func checkNameFormat(name string) bool {
	return NameFormatRegex.MatchString(name)
}
