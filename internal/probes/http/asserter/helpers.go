package asserter

import (
	"strconv"
	"time"
)

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

func isInt(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

func toInt(str string) (int, bool) {
	i, err := strconv.Atoi(str)
	return i, err == nil
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

func UnixMillisecondsToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}
