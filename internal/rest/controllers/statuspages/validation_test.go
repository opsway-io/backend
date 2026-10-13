package statuspages

import (
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/statuspage"
)

func TestDomainValidation(t *testing.T) {
	v := helpers.NewValidator()

	cases := map[string]bool{
		"status.example.com":     true,
		"Status.Example.COM":     true,
		"a-b.example.co.uk":      true,
		"example":                false,
		"":                       false,
		"status example.com":     false,
		"-bad.example.com":       false,
		"status.example.com/foo": false,
		"https://example.com":    false,
	}

	for domain, ok := range cases {
		err := v.Validate(&PostStatusPageRequest{TeamID: 1, Name: "n", Domain: domain})
		if (err == nil) != ok {
			t.Errorf("domain %q: valid=%v, want %v (err=%v)", domain, err == nil, ok, err)
		}
	}
}

func TestNormalizeDomain(t *testing.T) {
	if got := statuspage.NormalizeDomain(" Status.Example.com. "); got != "status.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestURLChecks(t *testing.T) {
	if !isHTTPURL("https://example.com/logo.png") || isHTTPURL("javascript:alert(1)") {
		t.Fatal("isHTTPURL")
	}
	if !isValidSupportURL("mailto:support@example.com") || isValidSupportURL("javascript:alert(1)") {
		t.Fatal("isValidSupportURL")
	}
}

func TestValidateSubscription(t *testing.T) {
	cases := []struct {
		channel, email, dest string
		ok                   bool
	}{
		{"EMAIL", "a@example.com", "", true},
		{"EMAIL", "not-an-email", "", false},
		{"EMAIL", "Name <a@example.com>", "", false},
		{"WEBHOOK", "a@example.com", "https://example.com/hook", true},
		{"WEBHOOK", "a@example.com", "http://example.com/hook", false},
		{"SLACK", "a@example.com", "https://hooks.slack.com/services/T/B/X", true},
		{"SLACK", "a@example.com", "https://evil.example.com/services/T/B/X", false},
		{"SMS", "a@example.com", "", false},
	}

	for _, c := range cases {
		err := validateSubscription(entities.StatusPageSubscriberChannel(c.channel), c.email, c.dest)
		if (err == nil) != c.ok {
			t.Errorf("%s %q %q: valid=%v want %v (%v)", c.channel, c.email, c.dest, err == nil, c.ok, err)
		}
	}
}
