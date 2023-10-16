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
		// Empty
		{
			name: "valid empty rule",
			args: args{
				rule: Rule{
					Source:   "HEADERS",
					Property: "Content-Type",
					Operator: "EMPTY",
					Target:   "",
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
					Target:   "foobar",
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
					Target:   "",
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
					Target:   "foobar",
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
					Target:   "100",
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
					Target:   "foobar",
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
					Target:   "100",
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
					Target:   "foobar",
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
					Target:   "",
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
					Target:   "",
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
		// General
		{
			name: "invalid source",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/json"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "INVALID",
						Property: "Content-Type",
						Operator: "EQUAL",
						Target:   "application/json",
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
		{
			name: "invalid operator",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/json"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "INVALID",
						Target:   "application/json",
					},
				},
			},
			wantOk:  nil,
			wantErr: true,
		},
		{
			name: "multiple valid rules",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type":   {"application/json"},
							"Content-Length": {"100"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EQUAL",
						Target:   "application/json",
					},
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "EQUAL",
						Target:   "99",
					},
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "GREATER_THAN",
						Target:   "50",
					},
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "LESS_THAN",
						Target:   "150",
					},
				},
			},
			wantOk:  []bool{true, false, true, true},
			wantErr: false,
		},
		// Equal
		{
			name: "valid equal rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/json"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EQUAL",
						Target:   "application/json",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid equal rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/xml"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EQUAL",
						Target:   "application/json",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not Equal
		{
			name: "valid not equal rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/json"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "NOT_EQUAL",
						Target:   "application/xml",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not equal rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/xml"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "NOT_EQUAL",
						Target:   "application/xml",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Empty
		{
			name: "valid empty rule true #1",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid empty rule true #2",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {""},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid empty rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/xml"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not Empty
		{
			name: "valid not empty rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {"application/xml"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "NOT_EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not empty rule false #1",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "NOT_EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "valid not empty rule false #2",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Type": {""},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Type",
						Operator: "NOT_EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Greater than
		{
			name: "valid greater than rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Length": {"100"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "GREATER_THAN",
						Target:   "50",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid greater than rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Length": {"100"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "GREATER_THAN",
						Target:   "150",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Less than
		{
			name: "valid less than rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Length": {"100"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "LESS_THAN",
						Target:   "150",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid less than rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Content-Length": {"100"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Content-Length",
						Operator: "LESS_THAN",
						Target:   "50",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Contains
		{
			name: "valid contains rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Server": {"nginx/1.19.0"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Server",
						Operator: "CONTAINS",
						Target:   "nginx",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid contains rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Server": {"nginx/1.19.0"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Server",
						Operator: "CONTAINS",
						Target:   "apache",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not contains
		{
			name: "valid not contains rule true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Server": {"nginx/1.19.0"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Server",
						Operator: "NOT_CONTAINS",
						Target:   "apache",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not contains rule false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Header: map[string][]string{
							"Server": {"nginx/1.19.0"},
						},
					},
				},
				rules: []Rule{
					{
						Source:   "HEADERS",
						Property: "Server",
						Operator: "NOT_CONTAINS",
						Target:   "nginx",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewHeadersAsserter()
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
