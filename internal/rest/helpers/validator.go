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
	v.RegisterValidation("monitorMethod", MonitorMethodValidator)
	v.RegisterValidation("monitorBodyType", BodyTypeValidator)
	v.RegisterValidation("monitorState", MonitorStateValidator)

	return &Validator{
		validator: v,
	}
}

func (cv *Validator) Validate(i interface{}) error {
	return errors.Wrap(cv.validator.Struct(i), "validation failed")
}

var AllowedRoles = []string{"MEMBER", "ADMIN", "OWNER"}

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

var AllowedMonitorMethods = []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}

func MonitorMethodValidator(fl validator.FieldLevel) bool {
	for _, method := range AllowedMonitorMethods {
		if method == fl.Field().String() {
			return true
		}
	}

	return false
}

var AllowedBodyTypes = []string{"NONE", "RAW", "JSON", "GRAPHQL", "XML"}

func BodyTypeValidator(fl validator.FieldLevel) bool {
	for _, bodyType := range AllowedBodyTypes {
		if bodyType == fl.Field().String() {
			return true
		}
	}

	return false
}

var AllowedMonitorStates = []string{"ACTIVE", "INACTIVE"}

func MonitorStateValidator(fl validator.FieldLevel) bool {
	for _, state := range AllowedMonitorStates {
		if state == fl.Field().String() {
			return true
		}
	}

	return false
}
