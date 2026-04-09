package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiagnoseOpenAICompatible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			fmt.Fprint(w, `{"data":[{"id":"llama3.2"},{"id":"qwen2.5-coder"}]}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/v1", "llama3.2")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if report.Backend != "openai-compatible" {
		t.Fatalf("Backend = %q, want %q", report.Backend, "openai-compatible")
	}
	if !report.ModelAvailable {
		t.Fatalf("expected model to be available")
	}
}

func TestDiagnoseOllamaFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			http.Error(w, "missing", http.StatusNotFound)
		case "/api/tags":
			fmt.Fprint(w, `{"models":[{"name":"llama3.2"},{"name":"mistral"}]}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "llama3.2")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.Diagnose(context.Background())
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
	if report.Backend != "ollama" {
		t.Fatalf("Backend = %q, want %q", report.Backend, "ollama")
	}
	if !report.ModelAvailable {
		t.Fatalf("expected model to be available")
	}
}