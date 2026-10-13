package browser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserScriptParsing(t *testing.T) {
	script := `[
		{"action": "navigate", "value": "https://example.com"},
		{"action": "wait_visible", "selector": "h1"},
		{"action": "assert_text", "selector": "h1", "value": "Example Domain"},
		{"action": "fill", "selector": "input[name='q']", "value": "search query"},
		{"action": "click", "selector": "button[type='submit']"},
		{"action": "sleep", "value": "500"},
		{"action": "screenshot"}
	]`

	var actions []Action
	err := json.Unmarshal([]byte(script), &actions)
	assert.NoError(t, err)
	assert.Len(t, actions, 7)

	assert.Equal(t, "navigate", actions[0].Action)
	assert.Equal(t, "https://example.com", actions[0].Value)

	assert.Equal(t, "wait_visible", actions[1].Action)
	assert.Equal(t, "h1", actions[1].Selector)

	assert.Equal(t, "assert_text", actions[2].Action)
	assert.Equal(t, "h1", actions[2].Selector)
	assert.Equal(t, "Example Domain", actions[2].Value)

	assert.Equal(t, "fill", actions[3].Action)
	assert.Equal(t, "input[name='q']", actions[3].Selector)
	assert.Equal(t, "search query", actions[3].Value)

	assert.Equal(t, "click", actions[4].Action)
	assert.Equal(t, "button[type='submit']", actions[4].Selector)

	assert.Equal(t, "sleep", actions[5].Action)
	assert.Equal(t, "500", actions[5].Value)

	assert.Equal(t, "screenshot", actions[6].Action)
}
