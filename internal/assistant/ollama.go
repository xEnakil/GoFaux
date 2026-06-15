package assistant

import (
	"context"
	"errors"
	"strings"
)

type OllamaProvider struct {
	Endpoint string
	Model    string
	Client   httpClient
}

func NewOllamaProvider(endpoint, model string) *OllamaProvider {
	return &OllamaProvider{
		Endpoint: normalizeOllamaEndpoint(endpoint),
		Model:    model,
		Client:   newDefaultHTTPClient(),
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if p.Client == nil {
		p.Client = newDefaultHTTPClient()
	}

	var response struct {
		Models []struct {
			Name       string `json:"name"`
			Model      string `json:"model"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
		} `json:"models"`
	}
	if err := getJSON(ctx, p.Client, ollamaTagsEndpoint(p.Endpoint), "", &response); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(response.Models))
	for _, item := range response.Models {
		name := item.Name
		if name == "" {
			name = item.Model
		}
		if name == "" {
			continue
		}
		models = append(models, ModelInfo{
			Provider:   p.Name(),
			Name:       name,
			Size:       item.Size,
			ModifiedAt: item.ModifiedAt,
		})
	}
	return models, nil
}

func (p *OllamaProvider) GenerateMock(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	prompt, err := buildGenerationPrompt(req)
	if err != nil {
		return GenerateResult{}, err
	}

	raw, err := p.call(ctx, prompt)
	if err != nil {
		return GenerateResult{}, err
	}

	body, err := ExtractJSON(raw)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{Provider: p.Name(), Model: p.Model, Body: body, Raw: raw, Attempts: 1}, nil
}

func (p *OllamaProvider) RepairJSON(ctx context.Context, invalid string, validationError string, req GenerateRequest) (GenerateResult, error) {
	prompt, err := buildRepairPrompt(req, invalid, validationError)
	if err != nil {
		return GenerateResult{}, err
	}

	raw, err := p.call(ctx, prompt)
	if err != nil {
		return GenerateResult{}, err
	}

	body, err := ExtractJSON(raw)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{Provider: p.Name(), Model: p.Model, Body: body, Raw: raw, Attempts: 1}, nil
}

func (p *OllamaProvider) call(ctx context.Context, prompt string) (string, error) {
	if p.Client == nil {
		p.Client = newDefaultHTTPClient()
	}

	payload := map[string]any{
		"model":  p.Model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
		"options": map[string]any{
			"temperature": 0.2,
		},
	}
	var response struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := postJSON(ctx, p.Client, p.Endpoint, "", payload, &response); err != nil {
		return "", err
	}
	if response.Error != "" {
		return "", errors.New(response.Error)
	}
	return response.Response, nil
}

func normalizeOllamaEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if strings.HasSuffix(endpoint, "/api/generate") {
		return endpoint
	}
	return endpoint + "/api/generate"
}

func ollamaTagsEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(endpoint, "/api/generate") {
		return strings.TrimSuffix(endpoint, "/api/generate") + "/api/tags"
	}
	if strings.HasSuffix(endpoint, "/api/tags") {
		return endpoint
	}
	return strings.TrimRight(endpoint, "/") + "/api/tags"
}
