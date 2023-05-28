package asserter

import (
	"fmt"

	"github.com/opsway-io/backend/internal/probes/http"
)

type Rule struct {
	// Which part of the response to assert on
	Source string

	// Input (if any) to the assertion
	Property any

	// The operator to use for the assertion
	Operator string

	// The target value  (if any) to assert against
	Target any
}

type Asserter interface {
	Assert(result *http.Result, rules []Rule) (ok []bool, err error)
	IsRuleValid(rule Rule) error
}

type HTTPResultAsserter struct {
	asserterMap map[string]Asserter
}

func NewHTTPResponseAsserter() *HTTPResultAsserter {
	return &HTTPResultAsserter{
		asserterMap: map[string]Asserter{
			"RESPONSE_TIME": NewResponseTimeAsserter(),
			"STATUS_CODE":   NewStatusCodeAsserter(),
			"HEADERS":       NewHeadersAsserter(),
			"TLS":           NewTLSAsserter(),
			"RAW_BODY":      NewRawBodyAsserter(),
			"JSON_BODY":     NewJSONBodyAsserter(),
		},
	}
}

func (a *HTTPResultAsserter) Assert(result *http.Result, rules []Rule) (ok []bool, err error) {
	if len(rules) == 0 {
		return []bool{}, nil
	}

	oks := make([]bool, len(rules))

	for i, rule := range rules {
		asserter, err := a.getAsserterForSource(rule.Source)
		if err != nil {
			return nil, err
		}

		ok, err = asserter.Assert(result, []Rule{rule})
		if err != nil {
			return nil, err
		}

		if len(ok) != 1 {
			return nil, fmt.Errorf("expected 1 result, got %d", len(ok))
		}

		oks[i] = ok[0]
	}

	return oks, nil
}

func (a *HTTPResultAsserter) IsRuleValid(rule Rule) error {
	asserter, err := a.getAsserterForSource(rule.Source)
	if err != nil {
		return err
	}

	return asserter.IsRuleValid(rule)
}

func (a *HTTPResultAsserter) getAsserterForSource(source string) (Asserter, error) {
	asserter, ok := a.asserterMap[source]
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", source)
	}

	return asserter, nil
}
