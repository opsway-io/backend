package asserter

import (
	"errors"
	"fmt"

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

var allowedStatusCodeOperators = []string{
	"EQUAL",
	"NOT_EQUAL",
	"GREATER_THAN",
	"LESS_THAN",
}

type StatusCodeAsserter struct{}

func NewStatusCodeAsserter() *StatusCodeAsserter {
	return &StatusCodeAsserter{}
}

func (a *StatusCodeAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
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

func (a *StatusCodeAsserter) IsRuleValid(rule Rule) error {
	// Source must be "STATUS_CODE"
	if ok := rule.Source == "STATUS_CODE"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The property must be empty
	if ok := rule.Property == ""; !ok {
		return fmt.Errorf("property must be empty: %s", rule.Property)
	}

	// The operator must be one of the allowed operators
	if ok := isStringInSlice(rule.Operator, allowedStatusCodeOperators); !ok {
		return fmt.Errorf("unknown operator: %v", rule.Operator)
	}

	// The target must be an integer
	if ok := isInt(rule.Target); !ok {
		return errors.New("invalid target")
	}

	return nil
}

func (a *StatusCodeAsserter) assert(result *http.Result, rule Rule) bool {
	targetInt, ok := toInt(rule.Target)
	if !ok {
		return false
	}

	switch rule.Operator {
	case "EQUAL":
		return result.Response.StatusCode == targetInt
	case "NOT_EQUAL":
		return result.Response.StatusCode != targetInt
	case "GREATER_THAN":
		return result.Response.StatusCode > targetInt
	case "LESS_THAN":
		return result.Response.StatusCode < targetInt
	default:
		return false
	}
}
