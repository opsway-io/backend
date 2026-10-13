package incident

import (
	"fmt"
	"strings"

	"github.com/opsway-io/backend/internal/entities"
)

var assertionTitles = map[string]string{
	"STATUS_CODE":   "Unexpected status code",
	"HEADERS":       "Response header check failed",
	"JSON_BODY":     "Response body check failed",
	"RESPONSE_TIME": "Response time too slow",
	"TLS":           "TLS certificate check failed",
	"TLS_HANDSHAKE": "TLS handshake failed",
}

var assertionSubjects = map[string]string{
	"STATUS_CODE":   "status code",
	"HEADERS":       "header",
	"JSON_BODY":     "body",
	"RESPONSE_TIME": "response time",
	"TLS":           "certificate",
	"TLS_HANDSHAKE": "TLS handshake",
}

var operatorPhrases = map[string]string{
	"EQUAL":                "to equal",
	"NOT_EQUAL":            "not to equal",
	"GREATER_THAN":         "to be greater than",
	"LESS_THAN":            "to be less than",
	"CONTAINS":             "to contain",
	"NOT_CONTAINS":         "not to contain",
	"EMPTY":                "to be empty",
	"NOT_EMPTY":            "not to be empty",
	"EXPIRES_GREATER_THAN": "to expire in more than",
	"EXPIRES_LESS_THAN":    "to expire in less than",
}

// DescribeAssertion returns a human readable incident title and description
// for a failed monitor assertion, e.g. "Unexpected status code" and
// "Expected status code to equal 200".
func DescribeAssertion(a entities.MonitorAssertion) (title, description string) {
	title, ok := assertionTitles[a.Source]
	if !ok {
		title = "Assertion failed"
	}

	subject, ok := assertionSubjects[a.Source]
	if !ok {
		subject = strings.ToLower(strings.ReplaceAll(a.Source, "_", " "))
	}

	if a.Property != "" {
		subject = fmt.Sprintf("%s %s", subject, a.Property)
	}

	phrase, ok := operatorPhrases[a.Operator]
	if !ok {
		phrase = strings.ToLower(strings.ReplaceAll(a.Operator, "_", " "))
	}

	description = fmt.Sprintf("Expected %s %s", subject, phrase)
	if a.Target != "" {
		description += " " + a.Target
	}

	return title, description
}
