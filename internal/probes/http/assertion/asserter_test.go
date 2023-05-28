package assertion

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
						Target:   100,
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
						Target:   100,
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
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
			a := NewHTTPResponseAsserter()
			gotOk, err := a.Assert(tt.args.result, tt.args.rules)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}
