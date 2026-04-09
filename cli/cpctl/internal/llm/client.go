package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	endpoint   string
	model      string
	httpClient *http.Client
	providers  []provider
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ollamaChatResponse struct {
	Message Message `json:"message"`
}

func NewClient(endpoint, model string) (*Client, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("ai endpoint is not configured; examples: http://localhost:1234/v1 for LM Studio or http://localhost:11434 for Ollama")
	}
	if model == "" {
		return nil, fmt.Errorf("ai model is not configured; examples: llama3.2, qwen2.5-coder, mistral")
	}

	return &Client{
		endpoint: endpoint,
		model:    model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// NewClientFromProviders creates a Client that iterates through the given
// provider list in order, returning the first successful response.
// APIKey values that start with "${" are expanded via os.ExpandEnv.
func NewClientFromProviders(cfgs []ProviderConfig) (*Client, error) {
	if len(cfgs) == 0 {
		return nil, fmt.Errorf("at least one AI provider must be configured")
	}

	hc := &http.Client{Timeout: 60 * time.Second}
	providers := make([]provider, 0, len(cfgs))
	for i, cfg := range cfgs {
		cfg.APIKey = os.ExpandEnv(cfg.APIKey)
		if cfg.Model == "" {
			return nil, fmt.Errorf("provider[%d] (%s): model must not be empty", i, cfg.Type)
		}
		p, err := buildProvider(cfg, hc)
		if err != nil {
			return nil, fmt.Errorf("provider[%d]: %w", i, err)
		}
		providers = append(providers, p)
	}

	return &Client{
		httpClient: hc,
		providers:  providers,
	}, nil
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	// Multi-provider path.
	if len(c.providers) > 0 {
		var lastErr error
		for _, p := range c.providers {
			content, err := p.chat(ctx, messages)
			if err == nil {
				return content, nil
			}
			slog.Debug("ai provider failed, trying next", "provider", p.name(), "error", err)
			lastErr = err
		}
		return "", fmt.Errorf("all ai providers failed; last error: %w", lastErr)
	}

	// Legacy single-endpoint path.
	primaryURL, fallbackURL := resolveEndpoints(c.endpoint)
	content, err := c.chatOpenAICompatible(ctx, primaryURL, messages)
	if err == nil {
		return content, nil
	}

	if fallbackURL != "" && fallbackURL != primaryURL {
		fallbackContent, fallbackErr := c.chatOllamaNative(ctx, fallbackURL, messages)
		if fallbackErr == nil {
			return fallbackContent, nil
		}
		return "", fmt.Errorf("ai request failed via openai-compatible endpoint (%v) and ollama endpoint (%v)", err, fallbackErr)
	}

	if looksLikeOllamaBase(c.endpoint) || strings.Contains(primaryURL, "/api/chat") {
		content, nativeErr := c.chatOllamaNative(ctx, primaryURL, messages)
		if nativeErr == nil {
			return content, nil
		}
		return "", fmt.Errorf("ai request failed via ollama endpoint: %w", nativeErr)
	}

	return "", fmt.Errorf("ai request failed via openai-compatible endpoint: %w", err)
}

func (c *Client) chatOpenAICompatible(ctx context.Context, endpoint string, messages []Message) (string, error) {
	requestBody, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode ai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create ai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach ai endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ai response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("ai endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to decode ai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai response did not contain any choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("ai response was empty")
	}

	return content, nil
}

func (c *Client) chatOllamaNative(ctx context.Context, endpoint string, messages []Message) (string, error) {
	requestBody, err := json.Marshal(ollamaChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach ollama endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ollama response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("ollama endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	content := strings.TrimSpace(parsed.Message.Content)
	if content == "" {
		return "", fmt.Errorf("ollama response was empty")
	}

	return content, nil
}

func resolveEndpoints(endpoint string) (string, string) {
	trimmed := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if trimmed == "" {
		return "", ""
	}

	if strings.HasSuffix(trimmed, "/v1/chat/completions") {
		return trimmed, replacePath(trimmed, "/api/chat")
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions", replacePath(trimmed, "/api/chat")
	}
	if strings.HasSuffix(trimmed, "/api/chat") {
		return replacePath(trimmed, "/v1/chat/completions"), trimmed
	}
	if looksLikeOllamaBase(trimmed) {
		return trimmed + "/v1/chat/completions", trimmed + "/api/chat"
	}
	return trimmed + "/v1/chat/completions", replacePath(trimmed+"/v1/chat/completions", "/api/chat")
}

func looksLikeOllamaBase(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return strings.Contains(parsed.Host, "11434")
}

func replacePath(rawURL, path string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parsed.Path = path
	parsed.RawPath = path
	return strings.TrimRight(parsed.String(), "/")
}