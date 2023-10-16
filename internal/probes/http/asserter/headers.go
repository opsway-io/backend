package asserter

import (
	"fmt"
	"strings"

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

	// The target must be set for the following operators:
	//	- CONTAINS
	//	- NOT_CONTAINS
	if ok := rule.Operator == "CONTAINS" || rule.Operator == "NOT_CONTAINS"; ok {
		if ok := rule.Target != ""; !ok {
			return fmt.Errorf("invalid target: %s", rule.Target)
		}
	}

	// The target must be empty for the following operators:
	//	- EMPTY
	//	- NOT_EMPTY
	if ok := rule.Operator == "EMPTY" || rule.Operator == "NOT_EMPTY"; ok {
		if ok := rule.Target == ""; !ok {
			return fmt.Errorf("invalid target: %s", rule.Target)
		}
	}

	// The target must be an integer for the following operators:
	//	- GREATER_THAN
	//	- LESS_THAN
	if ok := rule.Operator == "GREATER_THAN" || rule.Operator == "LESS_THAN"; ok {
		if _, ok := toInt(rule.Target); !ok {
			return fmt.Errorf("invalid target: %s", rule.Target)
		}
	}

	return nil
}

func (a *HeadersAsserter) assert(result *http.Result, rule Rule) bool {
	switch rule.Operator {
	case "EQUAL":
		return a.assertEqual(result, rule)
	case "NOT_EQUAL":
		return a.assertNotEqual(result, rule)
	case "EMPTY":
		return a.assertEmpty(result, rule)
	case "NOT_EMPTY":
		return a.assertNotEmpty(result, rule)
	case "GREATER_THAN":
		return a.assertGreaterThan(result, rule)
	case "LESS_THAN":
		return a.assertLessThan(result, rule)
	case "CONTAINS":
		return a.assertContains(result, rule)
	case "NOT_CONTAINS":
		return a.assertNotContains(result, rule)
	default:
		return false
	}
}

func (a *HeadersAsserter) assertEqual(result *http.Result, rule Rule) bool {
	return result.Response.Header.Get(rule.Property) == rule.Target
}

func (a *HeadersAsserter) assertNotEqual(result *http.Result, rule Rule) bool {
	return result.Response.Header.Get(rule.Property) != rule.Target
}

func (a *HeadersAsserter) assertEmpty(result *http.Result, rule Rule) bool {
	return result.Response.Header.Get(rule.Property) == ""
}

func (a *HeadersAsserter) assertNotEmpty(result *http.Result, rule Rule) bool {
	return result.Response.Header.Get(rule.Property) != ""
}

func (a *HeadersAsserter) assertGreaterThan(result *http.Result, rule Rule) bool {
	intTarget, ok := toInt(rule.Target)
	if !ok {
		return false
	}

	intResult, ok := toInt(result.Response.Header.Get(rule.Property))
	if !ok {
		return false
	}

	return intResult > intTarget
}

func (a *HeadersAsserter) assertLessThan(result *http.Result, rule Rule) bool {
	intTarget, ok := toInt(rule.Target)
	if !ok {
		return false
	}

	intResult, ok := toInt(result.Response.Header.Get(rule.Property))
	if !ok {
		return false
	}

	return intResult < intTarget
}

func (a *HeadersAsserter) assertContains(result *http.Result, rule Rule) bool {
	return strings.Contains(result.Response.Header.Get(rule.Property), rule.Target)
}

func (a *HeadersAsserter) assertNotContains(result *http.Result, rule Rule) bool {
	return !strings.Contains(result.Response.Header.Get(rule.Property), rule.Target)
}
