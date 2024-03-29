package asserter

import (
	"errors"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/probes/http"
)

/*
	Assertions about the TLS certificate of HTTP result.

	The following operators are supported:
		- Expired
		- Not Expired
		- Expires less than
		- Expires greater than
*/

var allowedTLSOperators = []string{
	"EXPIRED",
	"NOT_EXPIRED",
	"EXPIRES_LESS_THAN",
	"EXPIRES_GREATER_THAN",
}

type TLSAsserter struct{}

func NewTLSAsserter() *TLSAsserter {
	return &TLSAsserter{}
}

func (a *TLSAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
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

func (a *TLSAsserter) IsRuleValid(rule Rule) error {
	// Source must be "TLS"
	if ok := rule.Source == "TLS"; !ok {
		return fmt.Errorf("invalid source: %s", rule.Source)
	}

	// The operator must be one of the allowed operators
	if ok := isStringInSlice(rule.Operator, allowedTLSOperators); !ok {
		return fmt.Errorf("unknown operator: %v", rule.Operator)
	}

	// The property must be empty
	if ok := rule.Property == ""; !ok {
		return fmt.Errorf("property must be empty: %s", rule.Property)
	}

	// The target must be empty for the following operators:
	//	- EXPIRED
	//	- NOT_EXPIRED
	if ok := rule.Operator == "EXPIRED" || rule.Operator == "NOT_EXPIRED"; ok {
		if ok := rule.Target == ""; !ok {
			return fmt.Errorf("target must be empty: %s", rule.Target)
		}
	}

	// The target must be an integer for the following operators:
	//	- EXPIRES_LESS_THAN
	//	- EXPIRES_GREATER_THAN
	if ok := rule.Operator == "EXPIRES_LESS_THAN" || rule.Operator == "EXPIRES_GREATER_THAN"; ok {
		if ok := isInt(rule.Target); !ok {
			return errors.New("invalid target")
		}
	}

	return nil
}

func (a *TLSAsserter) assert(result *http.Result, rule Rule) bool {
	switch rule.Operator {
	case "EXPIRED":
		return a.assertExpired(result)
	case "NOT_EXPIRED":
		return a.assertNotExpired(result)
	case "EXPIRES_LESS_THAN":
		return a.assertExpiresLessThan(result, rule)
	case "EXPIRES_GREATER_THAN":
		return a.assertExpiresGreaterThan(result, rule)
	default:
		return false
	}
}

func (a *TLSAsserter) assertExpired(result *http.Result) bool {
	return result.TLS.Certificate.NotAfter.Before(time.Now())
}

func (a *TLSAsserter) assertNotExpired(result *http.Result) bool {
	return result.TLS.Certificate.NotAfter.After(time.Now())
}

func (a *TLSAsserter) assertExpiresLessThan(result *http.Result, rule Rule) bool {
	target, ok := a.getTargetDeltaAsTime(rule)
	if !ok {
		return false
	}

	return result.TLS.Certificate.NotAfter.Before(target)
}

func (a *TLSAsserter) assertExpiresGreaterThan(result *http.Result, rule Rule) bool {
	target, ok := a.getTargetDeltaAsTime(rule)
	if !ok {
		return false
	}

	return result.TLS.Certificate.NotAfter.After(target)
}

func (a *TLSAsserter) getTargetDeltaAsTime(rule Rule) (time.Time, bool) {
	target, ok := toInt(rule.Target)
	if !ok {
		return time.Time{}, false
	}

	return time.Now().Add(time.Duration(target) * time.Second), true
}
