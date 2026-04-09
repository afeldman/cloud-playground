package assistant

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"cpctl/internal/config"
	executil "cpctl/internal/exec"
)

const maxSectionBytes = 1800

func BuildSnapshot(ctx context.Context, topic string) string {
	repoRoot := config.RepoRoot()
	sections := []string{
		"Playground configuration:",
		fmt.Sprintf("- playground: %s", config.Cfg.Playground.Name),
		fmt.Sprintf("- stage: %s", config.Cfg.Development.Stage),
		fmt.Sprintf("- aws region: %s", config.Cfg.AWS.Region),
		fmt.Sprintf("- localstack endpoint: %s", config.Cfg.LocalStack.Endpoint),
		fmt.Sprintf("- focus topic: %s", topic),
	}

	sections = append(sections,
		capture(ctx, "kubectl nodes", "", "kubectl", "get", "nodes", "-o", "wide"),
		capture(ctx, "kubectl pods", "", "kubectl", "get", "pods", "-A"),
		capture(ctx, "docker containers", "", "docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}"),
	)

	tofuDir := filepath.Join(repoRoot, "tofu", "localstack")
	sections = append(sections,
		capture(ctx, "tofu output", tofuDir, "tofu", "output"),
	)

	switch strings.ToLower(topic) {
	case "lambda":
		sections = append(sections, capture(ctx, "local lambda functions", "", "aws", "--endpoint-url", config.Cfg.LocalStack.Endpoint, "lambda", "list-functions", "--output", "table"))
	case "batch":
		sections = append(sections, capture(ctx, "batch queues", "", "aws", "--endpoint-url", config.Cfg.LocalStack.Endpoint, "batch", "describe-job-queues", "--output", "table"))
	case "localstack":
		sections = append(sections, capture(ctx, "localstack containers", "", "docker", "ps", "--filter", "name=localstack", "--format", "table {{.Names}}\t{{.Status}}\t{{.Image}}"))
	case "tunnel":
		sections = append(sections, capture(ctx, "listening tcp ports", "", "lsof", "-iTCP", "-sTCP:LISTEN", "-nP"))
	}

	return strings.Join(filterEmpty(sections), "\n\n")
}

func DefaultSystemPrompt(custom string) string {
	if strings.TrimSpace(custom) != "" {
		return custom
	}
	return strings.Join([]string{
		"You are the cloud-playground operations assistant.",
		"Use the supplied environment snapshot only; do not invent cluster state.",
		"Respond with these sections in plain text:",
		"1. Assessment",
		"2. Likely causes",
		"3. Checks to run next",
		"4. Recommended commands",
	}, "\n")
}

func BuildPrompt(mode, input, snapshot string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"Mode: %s\n\nEnvironment snapshot:\n%s\n\nUser request:\n%s",
		mode,
		snapshot,
		input,
	))
}

func capture(ctx context.Context, title, dir, name string, args ...string) string {
	if !executil.Exists(name) {
		return fmt.Sprintf("%s:\n(unavailable: %s not installed)", title, name)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && text == "" {
		text = err.Error()
	}
	if len(text) > maxSectionBytes {
		text = text[:maxSectionBytes] + "\n...[truncated]"
	}
	if text == "" {
		text = "(no output)"
	}

	return fmt.Sprintf("%s:\n%s", title, text)
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}