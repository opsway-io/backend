package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client interface {
	GenerateRCA(ctx context.Context, prompt string) (string, error)
}

type clientImpl struct {
	baseURL string
	apiKey  string
	model   string
}

func NewClient(baseURL, apiKey, model string) Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1" // Default to local ollama openai api
	}
	if model == "" {
		model = "llama3" // Default ollama model
	}
	return &clientImpl{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
	}
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (c *clientImpl) GenerateRCA(ctx context.Context, prompt string) (string, error) {
	reqBody := ChatCompletionRequest{
		Model: c.model,
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
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/chat/completions", c.baseURL), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from LLM API")
	}

	return chatResp.Choices[0].Message.Content, nil
}
