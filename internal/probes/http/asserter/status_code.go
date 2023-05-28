package asserter

import (
	"errors"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the status code of a request.

	The following assertions are supported:
		- Equal
		- Not Equal
		- Greater than
		- Less than
*/

type StatusCodeAsserter struct{}

func NewStatusCodeAsserter() *StatusCodeAsserter {
	return &StatusCodeAsserter{}
}

func (a *StatusCodeAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (a *StatusCodeAsserter) IsRuleValid(rule Rule) error {
	// TODO: implement
	return errors.New("not implemented")
}
