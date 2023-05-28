package asserter

import (
	"errors"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the raw body of HTTP result.

	The following assertions are supported:
		- Equal
		- Not Equal
		- Empty
		- Not empty
		- Greater than
		- Less than
		- Contains
		- Not contains
*/

type RawBodyAsserter struct{}

func NewRawBodyAsserter() *RawBodyAsserter {
	return &RawBodyAsserter{}
}

func (a *RawBodyAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (a *RawBodyAsserter) IsRuleValid(rule Rule) error {
	// TODO: implement
	return errors.New("not implemented")
}
