package asserter

import (
	"testing"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestJSONBodyAsserter_IsRuleValid(t *testing.T) {
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
			name:    "Valid rule EQUAL",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "$.name", Operator: "EQUAL", Target: "test"}},
			wantErr: false,
		},
		{
			name:    "Invalid source",
			args:    args{rule: Rule{Source: "RAW_BODY", Property: "$.name", Operator: "EQUAL", Target: "test"}},
			wantErr: true,
		},
		{
			name:    "Empty property",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "", Operator: "EQUAL", Target: "test"}},
			wantErr: true,
		},
		{
			name:    "Invalid operator",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "$.name", Operator: "INVALID", Target: "test"}},
			wantErr: true,
		},
		{
			name:    "EMPTY operator with non-empty target",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "$.name", Operator: "EMPTY", Target: "test"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewJSONBodyAsserter()
			err := a.IsRuleValid(tt.args.rule)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONBodyAsserter_assert(t *testing.T) {
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
			name: "Valid json path EQUAL",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"name": "test"}`)}},
				rules:  []Rule{{Source: "JSON_BODY", Property: "$.name", Operator: "EQUAL", Target: "test"}},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Valid json path NOT_EQUAL",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"name": "test2"}`)}},
				rules:  []Rule{{Source: "JSON_BODY", Property: "$.name", Operator: "EQUAL", Target: "test"}},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "Invalid json body",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`invalid`)}},
				rules:  []Rule{{Source: "JSON_BODY", Property: "$.name", Operator: "EQUAL", Target: "test"}},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewJSONBodyAsserter()
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
