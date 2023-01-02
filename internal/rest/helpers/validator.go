package helpers

import (
	"github.com/go-playground/validator"
	"github.com/pkg/errors"
)

var AllowedRoles = []string{"admin", "member", "owner"}

type Validator struct {
	validator *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()

	v.RegisterValidation("teamRole", func(fl validator.FieldLevel) bool {
		for _, role := range AllowedRoles {
			if role == fl.Field().String() {
				return true
			}
		}

		return false
	})

	return &Validator{
		validator: v,
	}
}

func (cv *Validator) Validate(i interface{}) error {
	return errors.Wrap(cv.validator.Struct(i), "validation failed")
}
