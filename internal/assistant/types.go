package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Config struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

type FieldSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ConstraintSpec struct {
	Field       string   `json:"field,omitempty"`
	Kind        string   `json:"kind"`
	Type        string   `json:"type,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Digits      int      `json:"digits,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Values      []string `json:"values,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Source      string   `json:"source,omitempty"`
	Description string   `json:"description,omitempty"`
}

type GenerateRequest struct {
	Method         string           `json:"method"`
	Endpoint       string           `json:"endpoint"`
	Status         int              `json:"status"`
	Description    string           `json:"description"`
	DTOName        string           `json:"dto_name,omitempty"`
	RequestDTO     json.RawMessage  `json:"request_dto,omitempty"`
	ResponseSchema json.RawMessage  `json:"response_schema,omitempty"`
	Fields         []FieldSpec      `json:"fields,omitempty"`
	Constraints    []ConstraintSpec `json:"constraints,omitempty"`
}

type GenerateResult struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model,omitempty"`
	Body     json.RawMessage `json:"body"`
	Raw      string          `json:"raw,omitempty"`
	Attempts int             `json:"attempts"`
}

type Provider interface {
	Name() string
	GenerateMock(ctx context.Context, req GenerateRequest) (GenerateResult, error)
	RepairJSON(ctx context.Context, invalid string, validationError string, req GenerateRequest) (GenerateResult, error)
}

type ModelInfo struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	Size       int64  `json:"size,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

type ModelLister interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

func NewProvider(cfg Config) Provider {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case "ollama":
		return NewOllamaProvider(withDefault(cfg.Endpoint, "http://localhost:11434"), withDefault(cfg.Model, "llama3.2:3b"))
	case "openai", "lmstudio", "llamacpp", "localai":
		return NewOpenAICompatibleProvider(
			withDefault(cfg.Endpoint, "http://localhost:1234/v1/chat/completions"),
			withDefault(cfg.Model, "local-model"),
			cfg.APIKey,
		)
	default:
		return TemplateProvider{}
	}
}

func ListModels(ctx context.Context, cfg Config) ([]ModelInfo, error) {
	provider := NewProvider(cfg)
	lister, ok := provider.(ModelLister)
	if !ok {
		return []ModelInfo{}, nil
	}
	return lister.ListModels(ctx)
}

func (c Config) Description() string {
	provider := withDefault(c.Provider, "template")
	model := withDefault(c.Model, "default")
	endpoint := withDefault(c.Endpoint, "provider default")
	return fmt.Sprintf("provider=%s, model=%s, endpoint=%s", provider, model, endpoint)
}

func withDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
