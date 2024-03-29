package asserter

import (
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestHTTPResultAsserter_Assert(t *testing.T) {
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
		{
			name: "No rules",
			args: args{
				result: &http.Result{},
				rules:  []Rule{},
			},
			wantOk:  []bool{},
			wantErr: false,
		},
		{
			name: "Nil result fails",
			args: args{
				result: nil,
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "EQUAL",
						Target:   "100",
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
		{
			name: "Known source RESPONSE_TIME and valid rule passes",
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: 100 * time.Millisecond,
						},
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "EQUAL",
						Target:   "100",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Known source RESPONSE_TIME and invalid rule fails",
			args: args{
				result: &http.Result{},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "INVALID",
						Target:   "100",
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
		{
			name: "Response time and status code rules pass",
			args: args{
				result: &http.Result{
					Timing: http.Timing{
						Phases: http.TimingPhases{
							Total: 100 * time.Millisecond,
						},
					},
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "RESPONSE_TIME",
						Property: "TOTAL",
						Operator: "LESS_THAN",
						Target:   "500",
					},
					{
						Source:   "STATUS_CODE",
						Operator: "EQUAL",
						Target:   "200",
					},
				},
			},
			wantOk:  []bool{true, true},
			wantErr: false,
		},
		{
			name: "Unknown source fails",
			args: args{
				result: &http.Result{},
				rules: []Rule{
					{
						Source: "UNKNOWN",
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New()
			gotOk, err := a.Assert(tt.args.result, tt.args.rules)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
