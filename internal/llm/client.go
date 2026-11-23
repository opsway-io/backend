package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
)

type Client interface {
	GenerateRCA(ctx context.Context, prompt string) (string, error)
	Chat(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)
	Model() string
}

type clientImpl struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string) Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "qwen2.5-3b-instruct"
	}
	return &clientImpl{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// NewClientForConfig resolves a per-team BYO_LLM client when configured,
// falling back to the platform managed LLM client.
func NewClientForConfig(cfg *entities.AgentConfig, fallback Client) Client {
	if cfg == nil || cfg.ExecutionMode != entities.AgentModeByoLLM {
		return fallback
	}

	baseURL := ""
	if cfg.CustomLLMBaseURL != nil {
		baseURL = strings.TrimSpace(*cfg.CustomLLMBaseURL)
	}
	provider := ""
	if cfg.CustomLLMProvider != nil {
		provider = strings.ToLower(strings.TrimSpace(*cfg.CustomLLMProvider))
	}
	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "ollama":
			baseURL = "http://localhost:11434/v1"
		case "openrouter":
			baseURL = "https://openrouter.ai/api/v1"
		case "groq":
			baseURL = "https://api.groq.com/openai/v1"
		case "together":
			baseURL = "https://api.together.xyz/v1"
		}
	}

	model := ""
	if cfg.CustomLLMModel != nil {
		model = strings.TrimSpace(*cfg.CustomLLMModel)
	}
	apiKey := ""
	if cfg.CustomLLMAPIKey != nil {
		apiKey = strings.TrimSpace(*cfg.CustomLLMAPIKey)
	}

	if baseURL == "" && apiKey == "" && model == "" {
		return fallback
	}
	return NewClient(baseURL, apiKey, model)
}

func (c *clientImpl) Model() string {
	return c.model
}

type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type ToolDefinition struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ChatCompletionRequest struct {
	Model       string           `json:"model,omitempty"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  interface{}      `json:"tool_choice,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

var xmlToolCallPattern = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)

func (c *clientImpl) Chat(ctx context.Context, reqBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if strings.TrimSpace(reqBody.Model) == "" {
		reqBody.Model = c.model
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed (%s): %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read LLM response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from LLM API")
	}

	// Fallback parser for small GGUF models (e.g. Qwen2.5/Qwen3) that emit
	// <tool_call>{"name":"...","arguments":{...}}</tool_call> inside content.
	if len(chatResp.Choices[0].Message.ToolCalls) == 0 && len(reqBody.Tools) > 0 && strings.Contains(chatResp.Choices[0].Message.Content, "<tool_call>") {
		parsed := extractXMLToolCalls(chatResp.Choices[0].Message.Content)
		if len(parsed) > 0 {
			chatResp.Choices[0].Message.ToolCalls = parsed
		}
	}

	return &chatResp, nil
}

func extractXMLToolCalls(content string) []ToolCall {
	matches := xmlToolCallPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	calls := make([]ToolCall, 0, len(matches))
	for i, m := range matches {
		if len(m) < 2 {
			continue
		}
		var raw struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(m[1]), &raw); err != nil || strings.TrimSpace(raw.Name) == "" {
			continue
		}
		argStr := "{}"
		if len(raw.Arguments) > 0 {
			// Arguments may be a JSON object or an escaped JSON string
			var asStr string
			if err := json.Unmarshal(raw.Arguments, &asStr); err == nil {
				argStr = asStr
			} else {
				argStr = string(raw.Arguments)
			}
		}
		calls = append(calls, ToolCall{
			ID:   fmt.Sprintf("call_xml_%d", i+1),
			Type: "function",
			Function: ToolCallFunction{
				Name:      raw.Name,
				Arguments: argStr,
			},
		})
	}
	return calls
}

func (c *clientImpl) GenerateRCA(ctx context.Context, prompt string) (string, error) {
	temp := 0.2
	resp, err := c.Chat(ctx, ChatCompletionRequest{
		Model:       c.model,
		Temperature: &temp,
		MaxTokens:   256,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert DevOps AI assistant. Your task is to analyze the provided monitoring metrics and incident context to generate a short, concise Root Cause Analysis (RCA). Keep the RCA to 2-3 sentences max.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
