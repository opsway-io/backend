package asserter

import (
	"errors"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the status code of HTTP result.

	The following assertions are supported:
		- Equal
		- Not Equal
		- Empty
		- Not Empty
		- Greater than
		- Less than
		- Contains
		- Not contains
*/

type HeadersAsserter struct{}

func NewHeadersAsserter() *HeadersAsserter {
	return &HeadersAsserter{}
}

func (a *HeadersAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (a *HeadersAsserter) IsRuleValid(rule Rule) error {
	// TODO: implement
	return errors.New("not implemented")
}
