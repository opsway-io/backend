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

	The property is a JSON path to the value to be asserted (e.g. "$.data.id";
	a missing "$." prefix is added automatically).

	The following operators are supported:
		- EQUAL / NOT_EQUAL
		- EMPTY / NOT_EMPTY
		- GREATER_THAN / LESS_THAN
		- CONTAINS / NOT_CONTAINS
		- HAS_KEY / NOT_HAS_KEY (the property path exists / does not exist)
		- NULL / NOT_NULL

	Legacy spellings (EQUALS, NOT_EQUALS, IS_EMPTY, IS_NOT_EMPTY, IS_NULL,
	IS_NOT_NULL) are accepted as aliases.
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

var jsonBodyOperatorAliases = map[string]string{
	"EQUALS":       "EQUAL",
	"NOT_EQUALS":   "NOT_EQUAL",
	"IS_EMPTY":     "EMPTY",
	"IS_NOT_EMPTY": "NOT_EMPTY",
	"IS_NULL":      "NULL",
	"IS_NOT_NULL":  "NOT_NULL",
}

// JSONBodyOperatorsWithoutTarget lists the operators that take no target value.
var JSONBodyOperatorsWithoutTarget = []string{"EMPTY", "NOT_EMPTY", "NULL", "NOT_NULL", "HAS_KEY", "NOT_HAS_KEY"}

func normalizeJSONBodyOperator(op string) string {
	if alias, ok := jsonBodyOperatorAliases[op]; ok {
		return alias
	}

	return op
}

// NormalizeJSONPath prefixes a plain property path ("data.id") with "$.".
func NormalizeJSONPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "$") {
		return path
	}

	if strings.HasPrefix(path, "[") {
		return "$" + path
	}

	return "$." + path
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
	if rule.Source != "JSON_BODY" {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	if rule.Property == "" {
		return fmt.Errorf("empty property")
	}

	if _, err := jsonpath.Prepare(NormalizeJSONPath(rule.Property)); err != nil {
		return fmt.Errorf("invalid property: %s", rule.Property)
	}

	op := normalizeJSONBodyOperator(rule.Operator)
	if !isStringInSlice(op, allowedJSONBodyOperators) {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	if (op == "CONTAINS" || op == "NOT_CONTAINS") && rule.Target == "" {
		return fmt.Errorf("target must be set for operator: %s", rule.Operator)
	}

	if isStringInSlice(op, JSONBodyOperatorsWithoutTarget) && rule.Target != "" {
		return fmt.Errorf("target must be empty for operator: %s", rule.Operator)
	}

	if op == "GREATER_THAN" || op == "LESS_THAN" {
		if _, err := strconv.ParseFloat(rule.Target, 64); err != nil {
			return fmt.Errorf("target must be a number for operator: %s", rule.Operator)
		}
	}

	return nil
}

func (a *JSONBodyAsserter) assert(result *http.Result, rule Rule) bool {
	var data interface{}
	if err := json.Unmarshal(result.Response.Body, &data); err != nil {
		return false
	}

	op := normalizeJSONBodyOperator(rule.Operator)

	value, err := jsonpath.Read(data, NormalizeJSONPath(rule.Property))
	found := err == nil

	switch op {
	case "HAS_KEY":
		return found
	case "NOT_HAS_KEY":
		return !found
	}

	if !found {
		return false
	}

	switch op {
	case "EQUAL":
		return JSONValueToString(value) == rule.Target
	case "NOT_EQUAL":
		return JSONValueToString(value) != rule.Target
	case "EMPTY":
		return isEmptyJSONValue(value)
	case "NOT_EMPTY":
		return !isEmptyJSONValue(value)
	case "GREATER_THAN", "LESS_THAN":
		n, ok := jsonNumber(value)
		if !ok {
			return false
		}

		target, err := strconv.ParseFloat(rule.Target, 64)
		if err != nil {
			return false
		}

		if op == "GREATER_THAN" {
			return n > target
		}

		return n < target
	case "CONTAINS":
		return jsonContains(value, rule.Target)
	case "NOT_CONTAINS":
		return !jsonContains(value, rule.Target)
	case "NULL":
		return value == nil
	case "NOT_NULL":
		return value != nil
	default:
		return false
	}
}

// JSONValueToString renders a decoded JSON value the way a user would type it:
// strings unquoted, numbers without exponent, objects/arrays as JSON.
func JSONValueToString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}

		return string(b)
	}
}

func isEmptyJSONValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	}

	return false
}

func jsonNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)

		return f, err == nil
	}

	return 0, false
}

func jsonContains(value interface{}, target string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(v, target)
	case []interface{}:
		for _, item := range v {
			if JSONValueToString(item) == target {
				return true
			}
		}

		return false
	case map[string]interface{}:
		_, ok := v[target]

		return ok
	case nil:
		return false
	default:
		return strings.Contains(JSONValueToString(v), target)
	}
}
