package asserter

import (
	"testing"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestRawBodyAsserter_IsRuleValid(t *testing.T) {
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
					Property: "",
					Operator: "",
					Target:   "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "INVALID",
					Target:   "",
				},
			},
			wantErr: true,
		},
		{
			name: "property must be empty",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "Content-Type",
					Operator: "",
					Target:   "",
				},
			},
			wantErr: true,
		},
		// Equal
		{
			name: "valid equal #1",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "EQUAL",
					Target:   "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid equal #2",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "EQUAL",
					Target:   "",
				},
			},
			wantErr: false,
		},
		// Not Equal
		{
			name: "valid not equal #1",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_EQUAL",
					Target:   "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not equal #2",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_EQUAL",
					Target:   "",
				},
			},
			wantErr: false,
		},
		// Empty
		{
			name: "valid empty",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "EMPTY",
					Target:   "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid empty",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "EMPTY",
					Target:   "foobar",
				},
			},
			wantErr: true,
		},
		// Not empty
		{
			name: "valid not empty",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_EMPTY",
					Target:   "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not empty",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_EMPTY",
					Target:   "foobar",
				},
			},
			wantErr: true,
		},
		// Greater than
		{
			name: "valid greater than #1",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "GREATER_THAN",
					Target:   "1",
				},
			},
			wantErr: false,
		},
		{
			name: "valid greater than #2",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "GREATER_THAN",
					Target:   "0",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid greater than",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "GREATER_THAN",
					Target:   "foobar",
				},
			},
			wantErr: true,
		},
		// Less than
		{
			name: "valid less than #1",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "LESS_THAN",
					Target:   "1",
				},
			},
			wantErr: false,
		},
		{
			name: "valid less than #2",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "LESS_THAN",
					Target:   "0",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid less than",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "LESS_THAN",
					Target:   "foobar",
				},
			},
			wantErr: true,
		},
		// Contains
		{
			name: "valid contains",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "CONTAINS",
					Target:   "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid contains",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "CONTAINS",
					Target:   "",
				},
			},
			wantErr: true,
		},
		// Not contains
		{
			name: "valid not contains",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_CONTAINS",
					Target:   "foobar",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid not contains",
			args: args{
				rule: Rule{
					Source:   "RAW_BODY",
					Property: "",
					Operator: "NOT_CONTAINS",
					Target:   "",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewRawBodyAsserter()
			err := a.IsRuleValid(tt.args.rule)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRawBodyAsserter_assert(t *testing.T) {
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
			name: "Multiple rules",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EQUAL",
						Target:   "foobar",
					},
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   "barfoo",
					},
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true, true, false},
			wantErr: false,
		},
		// Equal
		{
			name: "valid equal true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EQUAL",
						Target:   "foobar",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid equal false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EQUAL",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not Equal
		{
			name: "valid not equal true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   "barfoo",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not equal false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_EQUAL",
						Target:   "foobar",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Empty
		{
			name: "valid empty true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte(""),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid empty false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not empty
		{
			name: "valid not empty true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_EMPTY",
						Target:   "",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not empty false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte(""),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
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
			name: "valid greater than true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("2"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "GREATER_THAN",
						Target:   "1",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid greater than false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("0"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "GREATER_THAN",
						Target:   "1",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Less than
		{
			name: "valid less than true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("0"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "LESS_THAN",
						Target:   "1",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid less than false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("2"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "LESS_THAN",
						Target:   "1",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Contains
		{
			name: "valid contains true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "CONTAINS",
						Target:   "foo",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid contains false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "CONTAINS",
						Target:   "biz",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		// Not contains
		{
			name: "valid not contains true",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_CONTAINS",
						Target:   "biz",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "valid not contains false",
			args: args{
				result: &http.Result{
					Response: http.Response{
						Body: []byte("foobar"),
					},
				},
				rules: []Rule{
					{
						Source:   "RAW_BODY",
						Property: "",
						Operator: "NOT_CONTAINS",
						Target:   "foo",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewRawBodyAsserter()
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
