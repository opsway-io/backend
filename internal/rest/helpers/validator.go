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

	v.RegisterValidation("teamRole", TeamRoleValidator)
	v.RegisterValidation("monitorFrequency", MonitorFrequencyValidator)

	return &Validator{
		validator: v,
	}
}

func (cv *Validator) Validate(i interface{}) error {
	return errors.Wrap(cv.validator.Struct(i), "validation failed")
}

var AllowedRoles = []string{"admin", "member", "owner"}

func TeamRoleValidator(fl validator.FieldLevel) bool {
	for _, role := range AllowedRoles {
		if role == fl.Field().String() {
			return true
		}
	}

	return false
}

var AllowedMonitorFrequencies = []uint64{
	30000,    // 30 seconds
	60000,    // 1 minute
	300000,   // 5 minutes
	600000,   // 10 minutes
	900000,   // 15 minutes
	1800000,  // 30 minutes
	3600000,  // 1 hour
	43200000, // 12 hours
	86400000, // 1 day
}

func MonitorFrequencyValidator(fl validator.FieldLevel) bool {
	for _, frequency := range AllowedMonitorFrequencies {
		if frequency == fl.Field().Uint() {
			return true
		}
	}

	return false
}
