package asserter

import (
	"errors"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the TLS certificate of HTTP result.

	The following assertions are supported:
		- Is valid
		- Is not valid
		- Expires within
		- Not expiring within
*/

type TLSAsserter struct{}

func NewTLSAsserter() *TLSAsserter {
	return &TLSAsserter{}
}

func (a *TLSAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (a *TLSAsserter) IsRuleValid(rule Rule) error {
	// TODO: implement
	return errors.New("not implemented")
}
