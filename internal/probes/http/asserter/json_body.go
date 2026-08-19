package asserter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/yalp/jsonpath"
)

/*
	Assertions about the JSON body of HTTP result.

	The property is a JSON path to the value to be asserted.

	The following operators are supported:
		- Equals
		- Not equals
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
	"EQUAL",
	"NOT_EQUAL",
	"EMPTY",
	"NOT_EMPTY",
	"GREATER_THAN",
	"LESS_THAN",
	"CONTAINS",
	"NOT_CONTAINS",
	"HAS_KEY",
	"NOT_HAS_KEY",
	"NULL",
	"NOT_NULL",
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
	// - EMPTY
	// - NOT_EMPTY
	// - NULL
	// - NOT_NULL
	if ok := rule.Operator == "EMPTY" || rule.Operator == "NOT_EMPTY" || rule.Operator == "NULL" || rule.Operator == "NOT_NULL"; ok {
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
	case "EQUAL":
		return a.assertEquals(value, rule.Target)
	case "NOT_EQUAL":
		return a.assertNotEquals(value, rule.Target)
	case "HAS_KEY":
		return a.assertHasKey(value, rule.Target)
	case "NOT_HAS_KEY":
		return a.assertNotHasKey(value, rule.Target)
	case "EMPTY":
		return a.assertIsEmpty(value)
	case "NOT_EMPTY":
		return a.assertIsNotEmpty(value)
	case "GREATER_THAN":
		return a.assertGreaterThan(value, rule.Target)
	case "LESS_THAN":
		return a.assertLessThan(value, rule.Target)
	case "CONTAINS":
		return a.assertContains(value, rule.Target)
	case "NOT_CONTAINS":
		return a.assertNotContains(value, rule.Target)
	case "NULL":
		return a.assertIsNull(value)
	case "NOT_NULL":
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
	return a.assertEquals(value, target)
}

func (a *JSONBodyAsserter) assertNotHasValue(value interface{}, target string) bool {
	return a.assertNotEquals(value, target)
}

func (a *JSONBodyAsserter) assertIsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return len(v) == 0
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (a *JSONBodyAsserter) assertIsNotEmpty(value interface{}) bool {
	return !a.assertIsEmpty(value)
}

func (a *JSONBodyAsserter) assertGreaterThan(value interface{}, target string) bool {
	targetFloat, err := strconv.ParseFloat(target, 64)
	if err != nil {
		return false
	}
	valueFloat, ok := value.(float64)
	if !ok {
		return false
	}
	return valueFloat > targetFloat
}

func (a *JSONBodyAsserter) assertLessThan(value interface{}, target string) bool {
	targetFloat, err := strconv.ParseFloat(target, 64)
	if err != nil {
		return false
	}
	valueFloat, ok := value.(float64)
	if !ok {
		return false
	}
	return valueFloat < targetFloat
}

func (a *JSONBodyAsserter) assertContains(value interface{}, target string) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return strings.Contains(v, target)
	case []interface{}:
		for _, item := range v {
			if fmt.Sprintf("%v", item) == target {
				return true
			}
		}
		return false
	case map[string]interface{}:
		for _, item := range v {
			if fmt.Sprintf("%v", item) == target {
				return true
			}
		}
		return false
	default:
		return strings.Contains(fmt.Sprintf("%v", value), target)
	}
}

func (a *JSONBodyAsserter) assertNotContains(value interface{}, target string) bool {
	return !a.assertContains(value, target)
}

func (a *JSONBodyAsserter) assertIsNull(value interface{}) bool {
	return value == nil
}

func (a *JSONBodyAsserter) assertIsNotNull(value interface{}) bool {
	return value != nil
}
