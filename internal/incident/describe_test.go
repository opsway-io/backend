package incident

import (
	"testing"

	"github.com/opsway-io/backend/internal/entities"
)

func TestDescribeAssertion(t *testing.T) {
	tests := []struct {
		name        string
		assertion   entities.MonitorAssertion
		title       string
		description string
	}{
		{
			name:        "status code",
			assertion:   entities.MonitorAssertion{Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"},
			title:       "Unexpected status code",
			description: "Expected status code to equal 200",
		},
		{
			name:        "json body with property",
			assertion:   entities.MonitorAssertion{Source: "JSON_BODY", Property: "$.items.length", Operator: "GREATER_THAN", Target: "0"},
			title:       "Response body check failed",
			description: "Expected body $.items.length to be greater than 0",
		},
		{
			name:        "no target",
			assertion:   entities.MonitorAssertion{Source: "HEADERS", Property: "content-type", Operator: "NOT_EMPTY"},
			title:       "Response header check failed",
			description: "Expected header content-type not to be empty",
		},
		{
			name:        "unknown source and operator",
			assertion:   entities.MonitorAssertion{Source: "SOMETHING_NEW", Operator: "MATCHES_REGEX", Target: "^ok$"},
			title:       "Assertion failed",
			description: "Expected something new matches regex ^ok$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, description := DescribeAssertion(tt.assertion)
			if title != tt.title {
				t.Errorf("title = %q, want %q", title, tt.title)
			}
			if description != tt.description {
				t.Errorf("description = %q, want %q", description, tt.description)
			}
		})
	}
}
