package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		primary  string
		fallback string
	}{
		{name: "ollama base", input: "http://localhost:11434", primary: "http://localhost:11434/v1/chat/completions", fallback: "http://localhost:11434/api/chat"},
		{name: "lmstudio v1", input: "http://localhost:1234/v1", primary: "http://localhost:1234/v1/chat/completions", fallback: "http://localhost:1234/api/chat"},
		{name: "ollama native", input: "http://localhost:11434/api/chat", primary: "http://localhost:11434/v1/chat/completions", fallback: "http://localhost:11434/api/chat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary, fallback := resolveEndpoints(tt.input)
			if primary != tt.primary || fallback != tt.fallback {
				t.Fatalf("resolveEndpoints(%q) = (%q, %q), want (%q, %q)", tt.input, primary, fallback, tt.primary, tt.fallback)
			}
		})
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"analysis ready"}}]}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "llama3.2")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if response != "analysis ready" {
		t.Fatalf("Chat() = %q, want %q", response, "analysis ready")
	}
}

func TestChatFallsBackToOllamaNative(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			http.Error(w, "not found", http.StatusNotFound)
		case "/api/chat":
			fmt.Fprint(w, `{"message":{"role":"assistant","content":"ollama ready"}}`)
		default:
			http.Error(w, "wrong path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "llama3.2")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	response, err := client.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if response != "ollama ready" {
		t.Fatalf("Chat() = %q, want %q", response, "ollama ready")
	}
}