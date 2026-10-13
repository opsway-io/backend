package alerting

import (
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestMatches(t *testing.T) {
	desc := "Target Down: connection refused"
	in := &entities.Incident{Title: "STATUS_CODE", Description: &desc}

	assert.True(t, Matches("*", in))
	assert.True(t, Matches("", in))
	assert.True(t, Matches("status_code", in))
	assert.True(t, Matches("Target Down", in))
	assert.False(t, Matches("RESPONSE_TIME", in))
	assert.False(t, Matches("TLS", &entities.Incident{Title: "HEADERS"}))
}

func TestParseChannels(t *testing.T) {
	assert.Equal(t, []string{"email", "slack"}, ParseChannels(`["email","Slack","email"]`))
	assert.Equal(t, []string{"email", "discord"}, ParseChannels("email, discord"))
	assert.Equal(t, []string{}, ParseChannels(""))
	assert.Equal(t, `["webhook"]`, EncodeChannels([]string{" Webhook "}))
}
