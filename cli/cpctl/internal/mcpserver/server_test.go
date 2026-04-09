package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestMCPProfileFromEnv_DefaultReadOnly(t *testing.T) {
	t.Setenv("CPCTL_MCP_PROFILE", "")

	got := mcpProfileFromEnv()
	if got != "read-only" {
		t.Fatalf("expected read-only default profile, got %q", got)
	}
}

func TestMCPProfileFromEnv_Normalizes(t *testing.T) {
	t.Setenv("CPCTL_MCP_PROFILE", "  Operator  ")

	got := mcpProfileFromEnv()
	if got != "operator" {
		t.Fatalf("expected normalized operator profile, got %q", got)
	}
}

func TestProfileAllowsMutating(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		want    bool
	}{
		{name: "read-only denied", profile: "read-only", want: false},
		{name: "operator allowed", profile: "operator", want: true},
		{name: "mutating allowed", profile: "mutating", want: true},
		{name: "full allowed", profile: "full", want: true},
		{name: "unknown denied", profile: "unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profileAllowsMutating(tt.profile)
			if got != tt.want {
				t.Fatalf("profile %q: expected %t, got %t", tt.profile, tt.want, got)
			}
		})
	}
}

func TestRegisterLambdaBatchTools_RegistersExpectedReadOnlyTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerLambdaBatchTools(s, t.TempDir())

	tools := s.ListTools()
	for _, name := range []string{"lambda_logs", "batch_watch", "batch_logs"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}

func TestRegisterLambdaBatchMutatingTools_RegistersExpectedTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0")
	registerLambdaBatchMutatingTools(s, t.TempDir())

	tools := s.ListTools()
	for _, name := range []string{"lambda_invoke", "lambda_deploy", "batch_submit"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}

func TestLambdaLogsTool_ExecutesCpctlCommand(t *testing.T) {
	installFakeCpctl(t)

	s := server.NewMCPServer("test", "1.0.0")
	registerLambdaBatchTools(s, t.TempDir())

	tool := s.GetTool("lambda_logs")
	if tool == nil {
		t.Fatalf("lambda_logs tool is not registered")
	}

	res, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"function_name": "demo-fn",
			"stage":         "mirror",
			"lines":         77,
		}},
	})
	if err != nil {
		t.Fatalf("lambda_logs handler returned error: %v", err)
	}

	out := toolResultText(t, res)
	want := "cpctl lambda logs demo-fn --stage mirror --lines 77"
	if !strings.Contains(out, want) {
		t.Fatalf("expected output to contain %q, got %q", want, out)
	}
}

func TestBatchSubmitTool_ExecutesCpctlCommand(t *testing.T) {
	installFakeCpctl(t)

	s := server.NewMCPServer("test", "1.0.0")
	registerLambdaBatchMutatingTools(s, t.TempDir())

	tool := s.GetTool("batch_submit")
	if tool == nil {
		t.Fatalf("batch_submit tool is not registered")
	}

	res, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"job_name":       "example-job",
			"job_definition": "example-def",
			"job_queue":      "example-queue",
			"stage":          "localstack",
		}},
	})
	if err != nil {
		t.Fatalf("batch_submit handler returned error: %v", err)
	}

	out := toolResultText(t, res)
	want := "cpctl batch submit example-job example-def example-queue --stage localstack"
	if !strings.Contains(out, want) {
		t.Fatalf("expected output to contain %q, got %q", want, out)
	}
}

func installFakeCpctl(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	fake := filepath.Join(tmp, "cpctl")
	content := "#!/bin/sh\n" +
		"printf 'cpctl %s\\n' \"$*\"\n"
	if err := os.WriteFile(fake, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write fake cpctl: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+oldPath)
}

func toolResultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()

	if res == nil {
		t.Fatalf("tool result is nil")
	}
	if len(res.Content) == 0 {
		t.Fatalf("tool result content is empty")
	}

	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", res.Content[0])
	}
	return text.Text
}
