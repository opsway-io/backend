package asserter

import "time"

func isRulesValid(asserter Asserter, rules []Rule) []error {
	errs := make([]error, len(rules))

	for i, rule := range rules {
		errs[i] = asserter.IsRuleValid(rule)
	}

	return errs
}

func isStringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

func allErrorsNil(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			return false
		}
	}

	return true
}

func durationToMilliseconds(d time.Duration) int {
	return int(d / time.Millisecond)
}
