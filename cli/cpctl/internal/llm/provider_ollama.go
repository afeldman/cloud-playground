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

// ollamaProvider calls Ollama's native chat API (POST /api/chat).
type ollamaProvider struct {
	url   string
	model string
	http  *http.Client
}

func (p *ollamaProvider) name() string { return "ollama(" + p.url + ")" }

func (p *ollamaProvider) chat(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(ollamaChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("ollama: server returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}

	content := strings.TrimSpace(parsed.Message.Content)
	if content == "" {
		return "", fmt.Errorf("ollama: response was empty")
	}
	return content, nil
}
