package llm

import "strings"

const SystemPrompt = `You are a precise, pragmatic senior software engineer.
Be concise. Prefer actionable solutions. No fluff.`

// injectSystemPrompt ensures that a system prompt is always present.
// If the user already provided one, it will not be overridden.
func injectSystemPrompt(messages []Message, model string) []Message {
	for _, m := range messages {
		if m.Role == "system" {
			return messages
		}
	}

	prompt := SystemPrompt

	// Optional: leichte Optimierung für lokale Modelle
	if strings.Contains(strings.ToLower(model), "llama") {
		prompt += "\nKeep answers short and structured."
	}

	return append([]Message{
		{Role: "system", Content: prompt},
	}, messages...)
}