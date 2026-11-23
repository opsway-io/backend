package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatWithTools_OpenAINativeToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req ChatCompletionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "qwen2.5-3b-instruct", req.Model)
		assert.Len(t, req.Tools, 1)

		resp := ChatCompletionResponse{
			ID:    "chatcmpl-1",
			Model: req.Model,
			Choices: []Choice{
				{
					Index:        0,
					FinishReason: "tool_calls",
					Message: Message{
						Role: "assistant",
						ToolCalls: []ToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: ToolCallFunction{
									Name:      "logs.get_error_summary",
									Arguments: `{"service_name":"payments-api"}`,
								},
							},
						},
					},
				},
			},
			Usage: Usage{PromptTokens: 120, CompletionTokens: 25, TotalTokens: 145},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-key", "qwen2.5-3b-instruct")
	resp, err := client.Chat(context.Background(), ChatCompletionRequest{
		Messages: []Message{{Role: "user", Content: "Check payments-api errors"}},
		Tools: []ToolDefinition{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "logs.get_error_summary",
					Description: "Get error summary",
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 145, resp.Usage.TotalTokens)
	require.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	assert.Equal(t, "logs.get_error_summary", resp.Choices[0].Message.ToolCalls[0].Function.Name)
}

func TestChatWithTools_XMLFallbackForSmallGGUFModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:    "chatcmpl-xml",
			Model: "qwen2.5-3b-instruct",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "<tool_call>\n{\"name\": \"monitoring.get_monitor_check_timings\", \"arguments\": {\"monitor_id\": 42, \"limit\": 5}}\n</tool_call>",
					},
				},
			},
			Usage: Usage{TotalTokens: 90},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", "qwen2.5-3b-instruct")
	resp, err := client.Chat(context.Background(), ChatCompletionRequest{
		Messages: []Message{{Role: "user", Content: "Investigate monitor 42"}},
		Tools: []ToolDefinition{
			{Type: "function", Function: FunctionDefinition{Name: "monitoring.get_monitor_check_timings"}},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	tc := resp.Choices[0].Message.ToolCalls[0]
	assert.Equal(t, "monitoring.get_monitor_check_timings", tc.Function.Name)
	assert.JSONEq(t, `{"monitor_id": 42, "limit": 5}`, tc.Function.Arguments)
}

func TestNewClientForConfig_BYOLLM(t *testing.T) {
	fallback := NewClient("http://llama-server:8000/v1", "", "qwen2.5-3b-instruct")

	// MANAGED mode returns fallback
	managedCfg := &entities.AgentConfig{ExecutionMode: entities.AgentModeManaged}
	assert.Equal(t, fallback, NewClientForConfig(managedCfg, fallback))

	// BYO_LLM mode uses custom provider/model/key
	provider := "openai"
	model := "gpt-4o-mini"
	key := "sk-custom"
	byoCfg := &entities.AgentConfig{
		ExecutionMode:     entities.AgentModeByoLLM,
		CustomLLMProvider: &provider,
		CustomLLMModel:    &model,
		CustomLLMAPIKey:   &key,
	}
	resolved := NewClientForConfig(byoCfg, fallback)
	assert.NotEqual(t, fallback, resolved)
	assert.Equal(t, "gpt-4o-mini", resolved.Model())
}
