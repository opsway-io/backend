package asserter

import (
	"encoding/json"
	"fmt"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/yalp/jsonpath"
)

/*
	Assertions about the JSON body of HTTP result.

	The property is a JSON path to the value to be asserted.

	The following operators are supported:
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

var allowedJSONBodyOperators = []string{
	"EQUALS",
	"NOT_EQUALS",
	"HAS_KEY",
	"NOT_HAS_KEY",
	"HAS_VALUE",
	"NOT_HAS_VALUE",
	"IS_EMPTY",
	"IS_NOT_EMPTY",
	"GREATER_THAN",
	"LESS_THAN",
	"CONTAINS",
	"NOT_CONTAINS",
	"IS_NULL",
	"IS_NOT_NULL",
}

type JSONBodyAsserter struct{}

func NewJSONBodyAsserter() *JSONBodyAsserter {
	return &JSONBodyAsserter{}
}

func (a *JSONBodyAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
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

func (a *JSONBodyAsserter) IsRuleValid(rule Rule) error {
	// Source must be "JSON_BODY"
	if ok := rule.Source == "JSON_BODY"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The property must not be empty
	if ok := rule.Property != ""; !ok {
		return fmt.Errorf("empty property")
	}

	// The property must be a valid JSON path
	if ok := a.isJSONPath(rule.Property); !ok {
		return fmt.Errorf("invalid property: %s", rule.Property)
	}

	// The operator must be one of the allowed operators
	if ok := isStringInSlice(rule.Operator, allowedJSONBodyOperators); !ok {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	// The target must be set for the following operators:
	// - CONTAINS
	// - NOT_CONTAINS
	// - HAS_KEY
	// - NOT_HAS_KEY
	// Not for EQUAL, NOT_EQUAL, HAS_VALUE, NOT_HAS_VALUE because the target can be empty
	if ok := rule.Operator == "CONTAINS" || rule.Operator == "NOT_CONTAINS" || rule.Operator == "HAS_KEY" || rule.Operator == "NOT_HAS_KEY"; ok {
		if ok := rule.Target != ""; !ok {
			return fmt.Errorf("target must be set for operator: %s", rule.Operator)
		}
	}

	// The target must be empty for the following operators:
	// - IS_EMPTY
	// - IS_NOT_EMPTY
	// - IS_NULL
	// - IS_NOT_NULL
	if ok := rule.Operator == "IS_EMPTY" || rule.Operator == "IS_NOT_EMPTY" || rule.Operator == "IS_NULL" || rule.Operator == "IS_NOT_NULL"; ok {
		if ok := rule.Target == ""; !ok {
			return fmt.Errorf("target must be empty for operator: %s", rule.Operator)
		}
	}

	// The target must be a number for the following operators:
	// - GREATER_THAN
	// - LESS_THAN
	if ok := rule.Operator == "GREATER_THAN" || rule.Operator == "LESS_THAN"; ok {
		if ok := isInt(rule.Target); !ok {
			return fmt.Errorf("target must be a number for operator: %s", rule.Operator)
		}
	}

	return nil
}

func (a *JSONBodyAsserter) isJSONPath(path string) bool {
	_, err := jsonpath.Prepare(path)

	return err == nil
}

func (a *JSONBodyAsserter) assert(result *http.Result, rule Rule) bool {
	var unmarshalData interface{}
	err := json.Unmarshal(result.Response.Body, &unmarshalData)
	if err != nil {
		return false
	}

	value, err := jsonpath.Read(unmarshalData, rule.Property)
	if err != nil {
		return false
	}

	switch rule.Operator {
	case "EQUALS":
		return a.assertEquals(value, rule.Target)
	case "NOT_EQUALS":
		return a.assertNotEquals(value, rule.Target)
	case "HAS_KEY":
		return a.assertHasKey(value, rule.Target)
	case "NOT_HAS_KEY":
		return a.assertNotHasKey(value, rule.Target)
	case "HAS_VALUE":
		return a.assertHasValue(value, rule.Target)
	case "NOT_HAS_VALUE":
		return a.assertNotHasValue(value, rule.Target)
	case "IS_EMPTY":
		return a.assertIsEmpty(value)
	case "IS_NOT_EMPTY":
		return a.assertIsNotEmpty(value)
	case "GREATER_THAN":
		return a.assertGreaterThan(value, rule.Target)
	case "LESS_THAN":
		return a.assertLessThan(value, rule.Target)
	case "CONTAINS":
		return a.assertContains(value, rule.Target)
	case "NOT_CONTAINS":
		return a.assertNotContains(value, rule.Target)
	case "IS_NULL":
		return a.assertIsNull(value)
	case "IS_NOT_NULL":
		return a.assertIsNotNull(value)
	default:
		return false
	}
}

func (a *JSONBodyAsserter) assertEquals(value interface{}, target string) bool {
	return fmt.Sprintf("%v", value) == target
}

func (a *JSONBodyAsserter) assertNotEquals(value interface{}, target string) bool {
	return fmt.Sprintf("%v", value) != target
}

func (a *JSONBodyAsserter) assertHasKey(value interface{}, target string) bool {
	return value != nil
}

func (a *JSONBodyAsserter) assertNotHasKey(value interface{}, target string) bool {
	return value == nil
}

func (a *JSONBodyAsserter) assertHasValue(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertNotHasValue(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertIsEmpty(value interface{}) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertIsNotEmpty(value interface{}) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertGreaterThan(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertLessThan(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertContains(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertNotContains(value interface{}, target string) bool {
	return false // TODO: Implement
}

func (a *JSONBodyAsserter) assertIsNull(value interface{}) bool {
	return value == nil
}

func (a *JSONBodyAsserter) assertIsNotNull(value interface{}) bool {
	return value != nil
}
