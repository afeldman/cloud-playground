package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openAIProvider handles OpenAI, mammoth.ai, LM Studio, or any service that
// speaks the OpenAI chat-completions wire format (POST /v1/chat/completions).
type openAIProvider struct {
	url    string
	apiKey string
	model  string
	http   *http.Client
}

func (p *openAIProvider) name() string { return "openai-compatible(" + p.url + ")" }

func (p *openAIProvider) chat(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("openai: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("openai: server returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("openai: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai: response contained no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("openai: response was empty")
	}
	return content, nil
}
