package prober

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCountryCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"global", "random"},
		{"opsway-dk", "dk"},
		{"opsway-de", "de"},
		{"dk-cph-1", "dk"},
		{"us-east-1", "us"},
		{"fr-par", "fr"},
		{"dk", "dk"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, ExtractCountryCode(tc.input))
		})
	}
}

func TestNilRedisFallback(t *testing.T) {
	fallback := []LocationInfo{
		{
			ID:          "opsway-dk",
			Name:        "Denmark (opsway-dk)",
			CountryCode: "dk",
			Status:      "online",
		},
	}

	svc := NewService(nil, fallback)
	ctx := context.Background()

	locs, err := svc.GetActiveLocations(ctx)
	assert.NoError(t, err)
	assert.Equal(t, fallback, locs)

	err = svc.RegisterHeartbeat(ctx, LocationInfo{ID: "test"})
	assert.NoError(t, err)

	err = svc.Unregister(ctx, "test")
	assert.NoError(t, err)
}
