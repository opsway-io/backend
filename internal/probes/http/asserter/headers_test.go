package asserter

import (
	"testing"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestHeadersAsserter_IsRuleValid(t *testing.T) {
	t.Parallel()

	type args struct {
		rule Rule
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// General
		{
			name: "property is not a string",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: 100,
					Operator: "EQUAL",
					Target:   "application/json",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid source",
			args: args{
				rule: Rule{
					Source:   "INVALID",
					Property: "Content-Type",
					Operator: "EQUAL",
					Target:   "application/json",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "INVALID",
					Target:   "application/json",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid target",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EQUAL",
					Target:   nil,
				},
			},
			wantErr: true,
		},
		// Equal
		{
			name: "valid equal rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EQUAL",
					Target:   "application/json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid equal rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EQUAL",
					Target:   100,
				},
			},
			wantErr: true,
		},
		// Not Equal
		{
			name: "valid not equal rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_EQUAL",
					Target:   "application/json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not equal rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_EQUAL",
					Target:   100,
				},
			},
			wantErr: true,
		},
		// Empty
		{
			name: "valid empty rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EMPTY",
					Target:   nil,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid empty rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EMPTY",
					Target:   "application/json",
				},
			},
			wantErr: true,
		},
		// Not Empty
		{
			name: "valid not empty rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_EMPTY",
					Target:   nil,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not empty rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_EMPTY",
					Target:   "application/json",
				},
			},
			wantErr: true,
		},
		// Greater than
		{
			name: "valid greater than rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Length",
					Operator: "GREATER_THAN",
					Target:   100,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid greater than rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Length",
					Operator: "GREATER_THAN",
					Target:   "100",
				},
			},
			wantErr: true,
		},
		// Less than
		{
			name: "valid less than rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Length",
					Operator: "LESS_THAN",
					Target:   100,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid less than rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Length",
					Operator: "LESS_THAN",
					Target:   "100",
				},
			},
			wantErr: true,
		},
		// Contains
		{
			name: "valid contains rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "CONTAINS",
					Target:   "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid contains rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "CONTAINS",
					Target:   100,
				},
			},
			wantErr: true,
		},
		// Not contains
		{
			name: "valid not contains rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_CONTAINS",
					Target:   "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not contains rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "NOT_CONTAINS",
					Target:   100,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewHeadersAsserter()
			err := a.IsRuleValid(tt.args.rule)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHeadersAsserter_assert(t *testing.T) {
	t.Parallel()

	type args struct {
		result *http.Result
		rules  []Rule
	}
	tests := []struct {
		name    string
		args    args
		wantOk  []bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewResponseTimeAsserter()
			gotOk, err := a.Assert(tt.args.result, tt.args.rules)

			assert.Equal(t, tt.wantOk, gotOk)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
