package asserter

import (
	"errors"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the JSON body of HTTP result.

	The following assertions are supported:
		- Equals
		- Not equals
		- Has key
		- Not has key
		- Has value
		- Not has value
		- Is empty
		- Is not empty
		- Greater than
		- Less than
		- Contains
		- Not contains
		- Is null
		- Is not null
*/

type JSONBodyAsserter struct{}

func NewJSONBodyAsserter() *JSONBodyAsserter {
	return &JSONBodyAsserter{}
}

func (a *JSONBodyAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (a *JSONBodyAsserter) IsRuleValid(rule Rule) error {
	// TODO: implement
	return errors.New("not implemented")
}
