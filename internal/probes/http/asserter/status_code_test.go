package asserter

import (
	"testing"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestStatusCodeAsserter_IsRuleValid(t *testing.T) {
	t.Parallel()

	type args struct {
		rule Rule
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid rule",
			args: args{
				rule: Rule{
					Source:   "STATUS_CODE",
					Property: "",
					Operator: "EQUAL",
					Target:   200,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid source",
			args: args{
				rule: Rule{
					Source:   "INVALID",
					Property: "",
					Operator: "EQUAL",
					Target:   200,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewStatusCodeAsserter()
			err := a.IsRuleValid(tt.args.rule)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStatusCodeAsserter_Assert(t *testing.T) {
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
		{
			name: "EQUAL passes",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "EQUAL",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "EQUAL fails",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "EQUAL",
						Target:   201,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "NOT_EQUAL passes",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   201,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "NOT_EQUAL fails",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "GREATER_THAN passes",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 201,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "GREATER_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "GREATER_THAN fails",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "GREATER_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "LESS_THAN passes",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 199,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "LESS_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "LESS_THAN fails",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "LESS_THAN",
						Target:   200,
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "multiple rules",
			args: args{
				result: &http.Result{
					Response: http.Response{
						StatusCode: 200,
					},
				},
				rules: []Rule{
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "EQUAL",
						Target:   200,
					},
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   201,
					},
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "GREATER_THAN",
						Target:   199,
					},
					{
						Source:   "STATUS_CODE",
						Property: "",
						Operator: "LESS_THAN",
						Target:   201,
					},
				},
			},
			wantOk:  []bool{true, true, true, true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewStatusCodeAsserter()
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
