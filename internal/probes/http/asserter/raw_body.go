package asserter

import (
	"fmt"
	"strings"

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

var allowedRawBodyOperators = []string{
	"EQUAL",
	"NOT_EQUAL",
	"EMPTY",
	"NOT_EMPTY",
	"GREATER_THAN",
	"LESS_THAN",
	"CONTAINS",
	"NOT_CONTAINS",
}

type RawBodyAsserter struct{}

func NewRawBodyAsserter() *RawBodyAsserter {
	return &RawBodyAsserter{}
}

func (a *RawBodyAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
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

func (a *RawBodyAsserter) IsRuleValid(rule Rule) error {
	// Source must be "RAW_BODY"
	if ok := rule.Source == "RAW_BODY"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The property must be empty
	if ok := rule.Property == ""; !ok {
		return fmt.Errorf("property must be empty: %s", rule.Property)
	}

	// The operator must be one of the allowed operators
	if ok := isStringInSlice(rule.Operator, allowedRawBodyOperators); !ok {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	// The target must be set for the following operators:
	// - CONTAINS
	// - NOT_CONTAINS
	// Not for EQUAL and NOT_EQUAL because the target can be empty
	if ok := rule.Operator == "CONTAINS" || rule.Operator == "NOT_CONTAINS"; ok {
		if ok := rule.Target != ""; !ok {
			return fmt.Errorf("target must be set for operator: %s", rule.Operator)
		}
	}

	// The target must be empty for the following operators:
	// - EMPTY
	// - NOT_EMPTY
	if ok := rule.Operator == "EMPTY" || rule.Operator == "NOT_EMPTY"; ok {
		if ok := rule.Target == ""; !ok {
			return fmt.Errorf("target must be empty for operator: %s", rule.Operator)
		}
	}

	// The target must be an integer for the following operators:
	// - GREATER_THAN
	// - LESS_THAN
	if ok := rule.Operator == "GREATER_THAN" || rule.Operator == "LESS_THAN"; ok {
		if ok := isInt(rule.Target); !ok {
			return fmt.Errorf("target must be an integer for operator: %s", rule.Operator)
		}
	}

	return nil
}

func (a *RawBodyAsserter) assert(result *http.Result, rule Rule) bool {
	bodyStr := string(result.Response.Body)

	switch rule.Operator {
	case "EQUAL":
		return a.assertEqual(bodyStr, rule)
	case "NOT_EQUAL":
		return a.assertNotEqual(bodyStr, rule)
	case "EMPTY":
		return a.assertEmpty(bodyStr, rule)
	case "NOT_EMPTY":
		return a.assertNotEmpty(bodyStr, rule)
	case "GREATER_THAN":
		return a.assertGreaterThan(bodyStr, rule)
	case "LESS_THAN":
		return a.assertLessThan(bodyStr, rule)
	case "CONTAINS":
		return a.assertContains(bodyStr, rule)
	case "NOT_CONTAINS":
		return a.assertNotContains(bodyStr, rule)
	default:
		return false
	}
}

func (a *RawBodyAsserter) assertEqual(body string, rule Rule) bool {
	return body == rule.Target
}

func (a *RawBodyAsserter) assertNotEqual(body string, rule Rule) bool {
	return body != rule.Target
}

func (a *RawBodyAsserter) assertEmpty(body string, rule Rule) bool {
	return body == ""
}

func (a *RawBodyAsserter) assertNotEmpty(body string, rule Rule) bool {
	return body != ""
}

func (a *RawBodyAsserter) assertGreaterThan(body string, rule Rule) bool {
	intTarget, ok := toInt(rule.Target)
	if !ok {
		return false
	}

	intResult, ok := toInt(body)
	if !ok {
		return false
	}

	return intResult > intTarget
}

func (a *RawBodyAsserter) assertLessThan(body string, rule Rule) bool {
	intTarget, ok := toInt(rule.Target)
	if !ok {
		return false
	}

	intResult, ok := toInt(body)
	if !ok {
		return false
	}

	return intResult < intTarget
}

func (a *RawBodyAsserter) assertContains(body string, rule Rule) bool {
	return strings.Contains(body, rule.Target)
}

func (a *RawBodyAsserter) assertNotContains(body string, rule Rule) bool {
	return !strings.Contains(body, rule.Target)
}
