package asserter

import (
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestResponseTimeAssertion_IsRuleValid(t *testing.T) {
	t.Parallel()

	type args struct {
		rule Rule
	}
	tests := []struct {
		name    string
		a       *ResponseTimeAssertion
		args    args
		wantErr bool
	}{
		{
			name: "valid rule",
			a:    &ResponseTimeAssertion{},
			args: args{
				rule: Rule{
					Source:   "RESPONSE_TIME",
					Property: "TOTAL",
					Operator: "EQUAL",
					Target:   100,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid source",
			a:    &ResponseTimeAssertion{},
			args: args{
				rule: Rule{
					Source:   "INVALID",
					Property: "TOTAL",
					Operator: "EQUAL",
					Target:   100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid property",
			a:    &ResponseTimeAssertion{},
			args: args{
				rule: Rule{
					Source:   "RESPONSE_TIME",
					Property: "INVALID",
					Operator: "EQUAL",
					Target:   100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			a:    &ResponseTimeAssertion{},
			args: args{
				rule: Rule{
					Source:   "RESPONSE_TIME",
					Property: "TOTAL",
					Operator: "INVALID",
					Target:   100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid target",
			a:    &ResponseTimeAssertion{},
			args: args{
				rule: Rule{
					Source:   "RESPONSE_TIME",
					Property: "TOTAL",
					Operator: "EQUAL",
					Target:   "INVALID",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ResponseTimeAssertion{}
			err := a.IsRuleValid(tt.args.rule)

			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestResponseTimeAssertion_Assert(t *testing.T) {
	t.Parallel()

	type args struct {
		result *http.Result
		rules  []Rule
	}
	tests := []struct {
		name    string
		a       *ResponseTimeAssertion
		args    args
		wantOk  []bool
		wantErr bool
	}{
		{
			name: "No rules success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{},
			},
			wantOk:  []bool{},
			wantErr: false,
		},
		{
			name: "Equal success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "EQUAL",
						Target:   100,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Equal failure",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "EQUAL",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "NotEqual success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "NOT_EQUAL",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "NotEqual failure",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "NOT_EQUAL",
						Target:   100,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "GreaterThan success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "GREATER_THAN",
						Target:   50,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "GreaterThan failure",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "GREATER_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "LessThan success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "LESS_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "LessThan failure",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "LESS_THAN",
						Target:   50,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "Invalid rule",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{},
				rules: []Rule{
					{
						Source:   "INVALID",
						Property: "TOTAL",
						Operator: "EQUAL",
						Target:   100,
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
		{
			name: "Multiple rules success",
			a:    &ResponseTimeAssertion{},
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							TCPConnection: time.Millisecond * 50,
							DNSLookup:     time.Millisecond * 20,
							TLSHandshake:  time.Millisecond * 30,
							Total:         time.Millisecond * 100,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TCP_CONNECTION",
						Operator: "GREATER_THAN",
						Target:   0,
					},
					{
						Source:   "RESPONSE_TIME",
						Property: "DNS_LOOKUP",
						Operator: "EQUAL",
						Target:   20,
					},
					{
						Source:   "RESPONSE_TIME",
						Property: "TLS_HANDSHAKE",
						Operator: "LESS_THAN",
						Target:   300,
					},
				},
			},
			wantOk:  []bool{true, true, true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ResponseTimeAssertion{}
			gotOk, err := a.Assert(tt.args.result, tt.args.rules)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
