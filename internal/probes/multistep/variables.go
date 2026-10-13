package multistep

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/probes/http/asserter"
	"github.com/yalp/jsonpath"
)

// Variable sources a step can extract values from.
const (
	VariableSourceJSONBody   = "JSON_BODY"
	VariableSourceHeader     = "HEADER"
	VariableSourceHeaders    = "HEADERS" // alias of HEADER
	VariableSourceStatusCode = "STATUS_CODE"
	VariableSourceRawBody    = "RAW_BODY"
)

var (
	variableNameRegexp  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,63}$`)
	placeholderRegexp   = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.-]{0,63})\s*\}\}`)
	allowedVarSources   = []string{VariableSourceJSONBody, VariableSourceHeader, VariableSourceHeaders, VariableSourceStatusCode, VariableSourceRawBody}
	sourcesWithProperty = []string{VariableSourceJSONBody, VariableSourceHeader, VariableSourceHeaders}
)

// ValidateVariable checks a variable definition.
func ValidateVariable(v entities.MonitorStepVariable) error {
	if !variableNameRegexp.MatchString(v.Name) {
		return fmt.Errorf("invalid variable name %q (letters, digits, '_', '-', '.'; must start with a letter or '_')", v.Name)
	}

	if !contains(allowedVarSources, v.Source) {
		return fmt.Errorf("invalid variable source %q", v.Source)
	}

	if contains(sourcesWithProperty, v.Source) && strings.TrimSpace(v.Property) == "" {
		return fmt.Errorf("variable %q requires a property", v.Name)
	}

	if v.Source == VariableSourceJSONBody {
		if _, err := jsonpath.Prepare(asserter.NormalizeJSONPath(v.Property)); err != nil {
			return fmt.Errorf("variable %q has an invalid JSON path %q", v.Name, v.Property)
		}
	}

	return nil
}

// ExtractVariable reads the value of a variable from a step's response.
func ExtractVariable(res *http.Result, v entities.MonitorStepVariable) (string, error) {
	if res == nil {
		return "", fmt.Errorf("no response")
	}

	switch v.Source {
	case VariableSourceJSONBody:
		var data interface{}
		if err := json.Unmarshal(res.Response.Body, &data); err != nil {
			return "", fmt.Errorf("response body is not valid JSON: %w", err)
		}

		value, err := jsonpath.Read(data, asserter.NormalizeJSONPath(v.Property))
		if err != nil {
			return "", fmt.Errorf("JSON path %q not found: %w", v.Property, err)
		}

		return asserter.JSONValueToString(value), nil
	case VariableSourceHeader, VariableSourceHeaders:
		values := res.Response.Header.Values(strings.TrimSpace(v.Property))
		if len(values) == 0 {
			return "", fmt.Errorf("header %q not found", v.Property)
		}

		return values[0], nil
	case VariableSourceStatusCode:
		return strconv.Itoa(res.Response.StatusCode), nil
	case VariableSourceRawBody:
		return string(res.Response.Body), nil
	default:
		return "", fmt.Errorf("unknown variable source %q", v.Source)
	}
}

// Substitute replaces {{name}} placeholders with known variable values.
// Unknown placeholders are left untouched.
func Substitute(s string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(s, "{{") {
		return s
	}

	return placeholderRegexp.ReplaceAllStringFunc(s, func(match string) string {
		name := placeholderRegexp.FindStringSubmatch(match)[1]
		if value, ok := vars[name]; ok {
			return value
		}

		return match
	})
}

// StripPlaceholders replaces placeholders with a dummy value, e.g. to validate
// a templated URL.
func StripPlaceholders(s string) string {
	return placeholderRegexp.ReplaceAllString(s, "x")
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}

	return false
}
