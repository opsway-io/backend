package asserter

import (
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
)

func TestTLSAsserter_IsRuleValid(t *testing.T) {
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
					Source:   "TLS",
					Operator: "EXPIRED",
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
			a := NewTLSAsserter()
			err := a.IsRuleValid(tt.args.rule)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTLSAsserter_Assert(t *testing.T) {
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
			name: "Certificate has not expired passes",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Hour),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "NOT_EXPIRED",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Certificate has not expired fails",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(-time.Hour),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "NOT_EXPIRED",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "Certificate has expired passes",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(-time.Hour),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRED",
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Certificate has expired fails",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Hour),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRED",
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "Certificate expires less than passes",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Minute),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRES_LESS_THAN",
						Target:   int64(time.Second * 300), // 5 minutes
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Certificate expires less than fails",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Minute),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRES_LESS_THAN",
						Target:   int64(time.Second * 30), // 30 seconds
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
		{
			name: "Certificate expires greater than passes",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Minute * 5),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRES_GREATER_THAN",
						Target:   int64(time.Second * 30), // 30 seconds
					},
				},
			},
			wantOk:  []bool{true},
			wantErr: false,
		},
		{
			name: "Certificate expires greater than fails",
			args: args{
				result: &http.Result{
					TLS: &http.TLS{
						Certificate: http.Certificate{
							NotAfter: time.Now().Add(time.Minute * 5),
						},
					},
				},
				rules: []Rule{
					{
						Source:   "TLS",
						Operator: "EXPIRES_GREATER_THAN",
						Target:   int64(time.Second * 300), // 5 minutes
					},
				},
			},
			wantOk:  []bool{false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewTLSAsserter()
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
