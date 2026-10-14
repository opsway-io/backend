package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	Locations []string `validate:"required,dive,location"`
	Frequency uint64   `validate:"required,monitorFrequency"`
}

func TestValidators(t *testing.T) {
	v := NewValidator()

	t.Run("valid locations and frequency", func(t *testing.T) {
		s := testStruct{
			Locations: []string{"global", "us-east"},
			Frequency: 60,
		}
		err := v.Validate(s)
		assert.NoError(t, err)
	})

	t.Run("empty location invalid", func(t *testing.T) {
		s := testStruct{
			Locations: []string{""},
			Frequency: 60,
		}
		err := v.Validate(s)
		assert.Error(t, err)
	})

	t.Run("invalid frequency", func(t *testing.T) {
		s := testStruct{
			Locations: []string{"global"},
			Frequency: 45, // not in AllowedMonitorFrequencies
		}
		err := v.Validate(s)
		assert.Error(t, err)
	})
}
