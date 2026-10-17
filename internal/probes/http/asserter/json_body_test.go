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
		{
			name:    "Valid rule HAS_KEY with empty target",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "$.name", Operator: "HAS_KEY", Target: ""}},
			wantErr: false,
		},
		{
			name:    "Valid rule NOT_HAS_KEY with empty target",
			args:    args{rule: Rule{Source: "JSON_BODY", Property: "$.name", Operator: "NOT_HAS_KEY", Target: ""}},
			wantErr: false,
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
		{
			name: "Valid json path without leading $ EQUAL",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"name": "test"}`)}},
				rules:  []Rule{{Source: "JSON_BODY", Property: "name", Operator: "EQUAL", Target: "test"}},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Valid json path HAS_KEY with empty target",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"name": "test", "nested": {"id": 123}}`)}},
				rules: []Rule{
					{Source: "JSON_BODY", Property: "$.name", Operator: "HAS_KEY", Target: ""},
					{Source: "JSON_BODY", Property: "$.nested.id", Operator: "HAS_KEY", Target: ""},
					{Source: "JSON_BODY", Property: "$.missing", Operator: "HAS_KEY", Target: ""},
				},
			},
			wantOk:  []bool{true, true, false},
			wantErr: false,
		},
		{
			name: "Valid json path NOT_HAS_KEY with empty target",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"name": "test"}`)}},
				rules: []Rule{
					{Source: "JSON_BODY", Property: "$.missing", Operator: "NOT_HAS_KEY", Target: ""},
					{Source: "JSON_BODY", Property: "$.name", Operator: "NOT_HAS_KEY", Target: ""},
				},
			},
			wantOk:  []bool{true, false},
			wantErr: false,
		},
		{
			name: "Valid json path array indexing",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`[{"name": "test", "id": 1}]`)}},
				rules: []Rule{
					{Source: "JSON_BODY", Property: "$", Operator: "NOT_NULL", Target: ""},
					{Source: "JSON_BODY", Property: "$[0].name", Operator: "HAS_KEY", Target: ""},
					{Source: "JSON_BODY", Property: "$[0].id", Operator: "NOT_NULL", Target: ""},
					{Source: "JSON_BODY", Property: "$[0].missing", Operator: "HAS_KEY", Target: ""},
				},
			},
			wantOk:  []bool{true, true, true, false},
			wantErr: false,
		},
		{
			name: "Valid json path nested array indexing",
			args: args{
				result: &http.Result{Response: http.Response{Body: []byte(`{"items": [{"title": "hello"}]}`)}},
				rules: []Rule{
					{Source: "JSON_BODY", Property: "$.items", Operator: "NOT_NULL", Target: ""},
					{Source: "JSON_BODY", Property: "$.items[0].title", Operator: "HAS_KEY", Target: ""},
					{Source: "JSON_BODY", Property: "$.items[0].missing", Operator: "HAS_KEY", Target: ""},
				},
			},
			wantOk:  []bool{true, true, false},
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

func TestNormalizeJSONPath(t *testing.T) {
	assert.Equal(t, "$.name", NormalizeJSONPath("name"))
	assert.Equal(t, "$.user.profile.id", NormalizeJSONPath("user.profile.id"))
	assert.Equal(t, "$[0].item", NormalizeJSONPath("[0].item"))
	assert.Equal(t, "$.name", NormalizeJSONPath("$.name"))
	assert.Equal(t, "$..item", NormalizeJSONPath("$..item"))
	assert.Equal(t, "@.item", NormalizeJSONPath("@.item"))
	assert.Equal(t, "", NormalizeJSONPath(""))
}
