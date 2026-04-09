package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ProviderType identifies the wire protocol / API dialect to use.
type ProviderType string

const (
	ProviderOpenAI           ProviderType = "openai"
	ProviderAnthropic        ProviderType = "anthropic"
	ProviderOpenAICompatible ProviderType = "openai-compatible"
	ProviderOllama           ProviderType = "ollama"
)

// ProviderConfig is the runtime representation of a single AI backend.
// APIKey is already expanded (os.ExpandEnv is applied by the caller).
type ProviderConfig struct {
	Type   ProviderType
	URL    string
	APIKey string
	Model  string
}

// provider is the internal interface every backend must implement.
type provider interface {
	name() string
	chat(ctx context.Context, messages []Message) (string, error)
}

// buildProvider constructs the concrete provider for a given config.
func buildProvider(cfg ProviderConfig, hc *http.Client) (provider, error) {
	switch cfg.Type {
	case ProviderOpenAI:
		base := cfg.URL
		if base == "" {
			base = "https://api.openai.com"
		}
		return &openAIProvider{
			url:    chatURL(base),
			apiKey: cfg.APIKey,
			model:  cfg.Model,
			http:   hc,
		}, nil

	case ProviderOpenAICompatible:
		if cfg.URL == "" {
			return nil, fmt.Errorf("openai-compatible provider requires a url")
		}
		return &openAIProvider{
			url:    chatURL(cfg.URL),
			apiKey: cfg.APIKey,
			model:  cfg.Model,
			http:   hc,
		}, nil

	case ProviderAnthropic:
		base := cfg.URL
		if base == "" {
			base = "https://api.anthropic.com"
		}
		return &anthropicProvider{
			url:    strings.TrimRight(base, "/") + "/v1/messages",
			apiKey: cfg.APIKey,
			model:  cfg.Model,
			http:   hc,
		}, nil

	case ProviderOllama:
		base := cfg.URL
		if base == "" {
			base = "http://localhost:11434"
		}
		return &ollamaProvider{
			url:   ollamaURL(base),
			model: cfg.Model,
			http:  hc,
		}, nil

	default:
		return nil, fmt.Errorf("unknown provider type %q", cfg.Type)
	}
}

// chatURL normalises a base URL to the OpenAI-compatible chat completions path.
func chatURL(base string) string {
	trimmed := strings.TrimRight(base, "/")
	if strings.HasSuffix(trimmed, "/v1/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}

// ollamaURL normalises a base URL to the Ollama native chat path.
func ollamaURL(base string) string {
	trimmed := strings.TrimRight(base, "/")
	if strings.HasSuffix(trimmed, "/api/chat") {
		return trimmed
	}
	return trimmed + "/api/chat"
}
