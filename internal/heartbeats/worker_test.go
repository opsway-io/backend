package heartbeats

import (
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestFormatHeartbeatFailureDescription(t *testing.T) {
	lastPing := time.Now().Add(-2 * time.Hour)
	hb := entities.Heartbeat{
		ID:       42,
		TeamID:   1,
		Name:     "Database Daily Backup",
		Interval: 1 * time.Hour,
		Grace:    15 * time.Minute,
		LastPing: &lastPing,
	}

	desc := FormatHeartbeatFailureDescription(hb)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, `Heartbeat "Database Daily Backup" (ID: 42) missed its expected check-in window.`)
	assert.Contains(t, desc, "Configured Interval: 1h0m0s")
	assert.Contains(t, desc, "Grace Period:        15m0s")
	assert.Contains(t, desc, "Allowed Window:      1h15m0s")
	assert.Contains(t, desc, "curl -fsS -m 10 --retry 3 https://api.opsway.eu/v1/heartbeats/42/ping")
	assert.Contains(t, desc, "Investigation Checklist")
	assert.Contains(t, desc, "Last Successful Ping:")
}

func TestFormatHeartbeatFailureDescription_NoPing(t *testing.T) {
	hb := entities.Heartbeat{
		ID:       99,
		TeamID:   2,
		Name:     "Analytics Exporter",
		Interval: 30 * time.Minute,
		Grace:    5 * time.Minute,
		LastPing: nil,
	}

	desc := FormatHeartbeatFailureDescription(hb)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, `Heartbeat "Analytics Exporter" (ID: 99) missed its expected check-in window.`)
	assert.Contains(t, desc, "Last Successful Ping: None (no check-in received since creation)")
	assert.Contains(t, desc, "https://api.opsway.eu/v1/heartbeats/99/ping")
}
