package assistant

import (
	"context"
	"errors"
	"strings"
)

type OpenAICompatibleProvider struct {
	Endpoint          string
	Model             string
	APIKey            string
	Client            httpClient
	UseResponseFormat bool
	MaxTokens         int
}

func NewOpenAICompatibleProvider(endpoint, model, apiKey string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		Endpoint:          normalizeOpenAIEndpoint(endpoint),
		Model:             model,
		APIKey:            apiKey,
		Client:            newDefaultHTTPClient(),
		UseResponseFormat: true,
		MaxTokens:         1200,
	}
}

func (p *OpenAICompatibleProvider) Name() string {
	return "openai-compatible"
}

func (p *OpenAICompatibleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if p.Client == nil {
		p.Client = newDefaultHTTPClient()
	}

	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := getJSON(ctx, p.Client, openAIModelsEndpoint(p.Endpoint), p.APIKey, &response); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(response.Data))
	for _, item := range response.Data {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		models = append(models, ModelInfo{
			Provider: p.Name(),
			Name:     item.ID,
		})
	}
	return models, nil
}

func (p *OpenAICompatibleProvider) GenerateMock(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
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
		compactPrompt, promptErr := buildCompactGenerationPrompt(req)
		if promptErr != nil {
			return GenerateResult{}, err
		}
		compactRaw, retryErr := p.call(ctx, compactPrompt)
		if retryErr != nil {
			return GenerateResult{}, err
		}
		compactBody, compactErr := ExtractJSON(compactRaw)
		if compactErr != nil {
			return GenerateResult{}, err
		}
		return GenerateResult{Provider: p.Name(), Model: p.Model, Body: compactBody, Raw: compactRaw, Attempts: 2}, nil
	}
	return GenerateResult{Provider: p.Name(), Model: p.Model, Body: body, Raw: raw, Attempts: 1}, nil
}

func (p *OpenAICompatibleProvider) RepairJSON(ctx context.Context, invalid string, validationError string, req GenerateRequest) (GenerateResult, error) {
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

func (p *OpenAICompatibleProvider) call(ctx context.Context, prompt string) (string, error) {
	if p.Client == nil {
		p.Client = newDefaultHTTPClient()
	}

	payload := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You generate valid JSON mock API responses. Return raw JSON only. The first character must be { or [. Do not explain."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	if p.MaxTokens > 0 {
		payload["max_tokens"] = p.MaxTokens
	}
	if p.UseResponseFormat {
		payload["response_format"] = map[string]string{
			"type": "json_object",
		}
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error any `json:"error"`
	}
	if err := postJSON(ctx, p.Client, p.Endpoint, p.APIKey, payload, &response); err != nil {
		delete(payload, "response_format")
		if retryErr := postJSON(ctx, p.Client, p.Endpoint, p.APIKey, payload, &response); retryErr != nil {
			return "", err
		}
	}
	if len(response.Choices) == 0 {
		return "", errors.New("local model returned no choices")
	}
	return response.Choices[0].Message.Content, nil
}

func normalizeOpenAIEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "http://localhost:1234/v1/chat/completions"
	}
	if strings.HasSuffix(endpoint, "/chat/completions") {
		return endpoint
	}
	if strings.HasSuffix(endpoint, "/v1") {
		return endpoint + "/chat/completions"
	}
	return endpoint
}

func openAIModelsEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "http://localhost:1234/v1/models"
	}
	if strings.HasSuffix(endpoint, "/chat/completions") {
		return strings.TrimSuffix(endpoint, "/chat/completions") + "/models"
	}
	if strings.HasSuffix(endpoint, "/v1") {
		return endpoint + "/models"
	}
	if strings.HasSuffix(endpoint, "/models") {
		return endpoint
	}
	return endpoint
}
