package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

type DoctorReport struct {
	ConfiguredEndpoint string
	PrimaryEndpoint    string
	FallbackEndpoint   string
	Backend            string
	Model              string
	Reachable          bool
	ModelAvailable     bool
	AvailableModels    []string
	Details            []string
}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
		Model string `json:"model"`
	} `json:"models"`
}

func (c *Client) Diagnose(ctx context.Context) (*DoctorReport, error) {
	primaryURL, fallbackURL := resolveEndpoints(c.endpoint)
	report := &DoctorReport{
		ConfiguredEndpoint: c.endpoint,
		PrimaryEndpoint:    primaryURL,
		FallbackEndpoint:   fallbackURL,
		Model:              c.model,
	}

	openAIModels, openAIErr := c.fetchOpenAIModels(ctx, primaryURL)
	if openAIErr == nil {
		report.Backend = "openai-compatible"
		report.Reachable = true
		report.AvailableModels = openAIModels
		report.ModelAvailable = slices.Contains(openAIModels, c.model)
		if report.ModelAvailable {
			report.Details = append(report.Details, "Configured model is available via the OpenAI-compatible API.")
		} else {
			report.Details = append(report.Details, "Configured model was not returned by the OpenAI-compatible models endpoint.")
		}
		return report, nil
	}

	ollamaModels, ollamaErr := c.fetchOllamaModels(ctx, fallbackURL)
	if ollamaErr == nil {
		report.Backend = "ollama"
		report.Reachable = true
		report.AvailableModels = ollamaModels
		report.ModelAvailable = slices.Contains(ollamaModels, c.model)
		if report.ModelAvailable {
			report.Details = append(report.Details, "Configured model is available via the native Ollama API.")
		} else {
			report.Details = append(report.Details, "Configured model was not returned by the native Ollama tags endpoint.")
		}
		if openAIErr != nil {
			report.Details = append(report.Details, "OpenAI-compatible probe failed: "+openAIErr.Error())
		}
		return report, nil
	}

	report.Backend = "unreachable"
	report.Reachable = false
	report.Details = append(report.Details,
		"OpenAI-compatible probe failed: "+openAIErr.Error(),
		"Ollama probe failed: "+ollamaErr.Error(),
	)
	return report, fmt.Errorf("ai endpoint is not reachable")
}

func (c *Client) fetchOpenAIModels(ctx context.Context, chatEndpoint string) ([]string, error) {
	modelsURL := replacePath(chatEndpoint, "/v1/models")
	body, err := c.get(ctx, modelsURL)
	if err != nil {
		return nil, err
	}

	var parsed openAIModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	models := make([]string, 0, len(parsed.Data))
	for _, entry := range parsed.Data {
		if strings.TrimSpace(entry.ID) == "" {
			continue
		}
		models = append(models, entry.ID)
	}
	return models, nil
}

func (c *Client) fetchOllamaModels(ctx context.Context, endpoint string) ([]string, error) {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = replacePath(c.endpoint, "/api/tags")
	}
	if !strings.HasSuffix(endpoint, "/api/tags") {
		endpoint = replacePath(endpoint, "/api/tags")
	}

	body, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var parsed ollamaTagsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode ollama tags response: %w", err)
	}

	models := make([]string, 0, len(parsed.Models))
	for _, entry := range parsed.Models {
		name := strings.TrimSpace(entry.Name)
		model := strings.TrimSpace(entry.Model)
		switch {
		case name != "":
			models = append(models, name)
		case model != "":
			models = append(models, model)
		}
	}
	return models, nil
}

func (c *Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%s returned %s: %s", endpoint, resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func baseURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}