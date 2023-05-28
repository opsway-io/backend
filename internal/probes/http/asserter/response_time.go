package asserter

import (
	"errors"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the response time of a request.
	The following assertions are supported:
		- Equal
		- Not Equal
		- Greater than
		- Less than
	on the following response time metrics:
		- DNS lookup
		- TCP connection
		- TLS handshake
		- Server processing
		- Content transfer
		- Total
*/

var (
	allowedResponseTimeProperties = []string{
		"DNS_LOOKUP",
		"TCP_CONNECTION",
		"TLS_HANDSHAKE",
		"SERVER_PROCESSING",
		"CONTENT_TRANSFER",
		"TOTAL",
	}

	allowedResponseTimeOperators = []string{
		"EQUAL",
		"NOT_EQUAL",
		"GREATER_THAN",
		"LESS_THAN",
	}
)

type ResponseTimeAssertion struct{}

func NewResponseTimeAsserter() *ResponseTimeAssertion {
	return &ResponseTimeAssertion{}
}

func (a *ResponseTimeAssertion) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
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

func (a *ResponseTimeAssertion) IsRuleValid(rule Rule) error {
	// Source must be "RESPONSE_TIME"
	if ok := rule.Source == "RESPONSE_TIME"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The property must be one of the response time metrics
	properyStr, ok := rule.Property.(string)
	if !ok {
		return errors.New("property must be a string")
	}
	if ok := isStringInSlice(properyStr, allowedResponseTimeProperties); !ok {
		return fmt.Errorf("unknown property: %s", rule.Property)
	}

	// The operator must be one of the response time assertions
	if ok := isStringInSlice(rule.Operator, allowedResponseTimeOperators); !ok {
		return fmt.Errorf("unknown operator: %v", rule.Operator)
	}

	// The target must be an integer
	if _, ok := rule.Target.(int); !ok {
		return errors.New("target must be an integer")
	}

	return nil
}

func (a *ResponseTimeAssertion) assert(result *http.Result, rule Rule) bool {
	propertyStr, ok := rule.Property.(string)
	if !ok {
		return false
	}

	var resultValue time.Duration
	switch propertyStr {
	case "DNS_LOOKUP":
		resultValue = result.Timing.Phases.DNSLookup
	case "TCP_CONNECTION":
		resultValue = result.Timing.Phases.TCPConnection
	case "TLS_HANDSHAKE":
		resultValue = result.Timing.Phases.TLSHandshake
	case "SERVER_PROCESSING":
		resultValue = result.Timing.Phases.ServerProcessing
	case "CONTENT_TRANSFER":
		resultValue = result.Timing.Phases.ContentTransfer
	case "TOTAL":
		resultValue = result.Timing.Phases.Total
	default:
		return false
	}

	resultInt := durationToMilliseconds(resultValue)

	targetInt, ok := rule.Target.(int)
	if !ok {
		return false
	}

	switch rule.Operator {
	case "EQUAL":
		return resultInt == targetInt
	case "NOT_EQUAL":
		return resultInt != targetInt
	case "GREATER_THAN":
		return resultInt > targetInt
	case "LESS_THAN":
		return resultInt < targetInt
	}

	return false
}
