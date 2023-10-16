package asserter

import (
	"errors"
	"fmt"

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

var allowedHeadersOperators = []string{
	"EQUAL",
	"NOT_EQUAL",
	"EMPTY",
	"NOT_EMPTY",
	"GREATER_THAN",
	"LESS_THAN",
	"CONTAINS",
	"NOT_CONTAINS",
}

type HeadersAsserter struct{}

func NewHeadersAsserter() *HeadersAsserter {
	return &HeadersAsserter{}
}

func (a *HeadersAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	if len(rules) == 0 {
		return []bool{}, nil
	}

	errs := isRulesValid(a, rules)
	if !allErrorsNil(errs) {
		return nil, fmt.Errorf("invalid rules: %v", errs)
	}

	ok = make([]bool, len(rules))

	for i, rule := range rules {
		ok[i] = a.assert(result, rule)
	}

	return ok, nil
}

func (a *HeadersAsserter) IsRuleValid(rule Rule) error {
	// Source must be "HEADERS"
	if ok := rule.Source == "HEADERS"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The operator must be one of the allowed operators
	if ok := isStringInSlice(rule.Operator, allowedHeadersOperators); !ok {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	// The property must be a string
	if _, ok := rule.Property.(string); !ok {
		return fmt.Errorf("property must be a string: %v", rule.Property)
	}

	// If the operator is "GREATER_THAN" or "LESS_THAN"
	// the target must be an integer
	if rule.Operator == "GREATER_THAN" || rule.Operator == "LESS_THAN" {
		if _, ok := rule.Target.(int); !ok {
			return errors.New("target must be an int64")
		}
	}

	// If the operator is "EMPTY" or "NOT_EMPTY" the target must be empty
	if rule.Operator == "EMPTY" || rule.Operator == "NOT_EMPTY" {
		if ok := rule.Target == "" || rule.Target == nil; !ok {
			return fmt.Errorf("target must be empty: %s", rule.Target)
		}
	}

	// If the operator is "EQUAL" or "NOT_EQUAL" the target must be a string
	if rule.Operator == "EQUAL" || rule.Operator == "NOT_EQUAL" {
		if _, ok := rule.Target.(string); !ok {
			return errors.New("target must be a string")
		}
	}

	// If the operator is "CONTAINS" or "NOT_CONTAINS" the target must be a string
	if rule.Operator == "CONTAINS" || rule.Operator == "NOT_CONTAINS" {
		if _, ok := rule.Target.(string); !ok {
			return errors.New("target must be a string")
		}
	}

	return nil
}

func (a *HeadersAsserter) assert(result *http.Result, rule Rule) bool {
	return false // TODO: implement
}
