package helpers

import (
	"github.com/go-playground/validator"
	"github.com/pkg/errors"
)

type Validator struct {
	validator *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()

	return &Validator{
		validator: v,
	}
}

func (cv *Validator) Validate(i interface{}) error {
	return errors.Wrap(cv.validator.Struct(i), "validation failed")
}
