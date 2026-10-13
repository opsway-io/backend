package alerting

import (
	"encoding/json"
	"strings"

	"github.com/opsway-io/backend/internal/entities"
)

// ConditionAny matches every incident.
const ConditionAny = "*"

// Matches reports whether an alert rule condition matches an incident.
// "*" (or an empty condition) matches everything. Otherwise the condition is compared case-insensitively
// with the incident title (which holds the failed assertion source, e.g. STATUS_CODE) and, as a fallback,
// searched for in the title and description.
func Matches(condition string, incident *entities.Incident) bool {
	cond := strings.TrimSpace(condition)
	if cond == "" || cond == ConditionAny {
		return true
	}

	cond = strings.ToLower(cond)
	title := strings.ToLower(incident.Title)

	if title == cond || strings.Contains(title, cond) {
		return true
	}

	if incident.Description != nil && strings.Contains(strings.ToLower(*incident.Description), cond) {
		return true
	}

	return false
}

// ParseChannels decodes the channels of a rule. Channels are stored as a JSON array, but a comma
// separated list is accepted for robustness.
func ParseChannels(channels string) []string {
	channels = strings.TrimSpace(channels)
	if channels == "" {
		return []string{}
	}

	var parsed []string
	if err := json.Unmarshal([]byte(channels), &parsed); err == nil {
		return normalizeChannels(parsed)
	}

	return normalizeChannels(strings.Split(channels, ","))
}

// EncodeChannels encodes channels as a JSON array.
func EncodeChannels(channels []string) string {
	b, _ := json.Marshal(normalizeChannels(channels))

	return string(b)
}

func normalizeChannels(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}

	for _, c := range in {
		c = strings.ToLower(strings.TrimSpace(c))
		if c == "" || seen[c] {
			continue
		}

		seen[c] = true
		out = append(out, c)
	}

	return out
}
