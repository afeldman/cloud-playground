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

// anthropicProvider calls Anthropic's Messages API
// (POST /v1/messages, auth via x-api-key header).
type anthropicProvider struct {
	url    string
	apiKey string
	model  string
	http   *http.Client
}

func (p *anthropicProvider) name() string { return "anthropic(" + p.url + ")" }

func (p *anthropicProvider) chat(ctx context.Context, messages []Message) (string, error) {
	// Anthropic separates the system prompt from the user/assistant turns.
	var system string
	var turns []anthropicMessage
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		turns = append(turns, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	reqBody := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Messages:  turns,
	}
	if system != "" {
		reqBody.System = system
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("anthropic: server returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: decode response: %w", err)
	}

	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("anthropic: response contained no text content")
}

// ---- wire types ----

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
