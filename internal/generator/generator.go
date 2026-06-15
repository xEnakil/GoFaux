package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"GoFaux/internal/assistant"
	"GoFaux/internal/config"
	"GoFaux/internal/mock"
	"GoFaux/internal/runner"
)

type Request struct {
	Intent          string                     `json:"intent"`
	Method          string                     `json:"method,omitempty"`
	Endpoint        string                     `json:"endpoint,omitempty"`
	Status          int                        `json:"status,omitempty"`
	Name            string                     `json:"name,omitempty"`
	Description     string                     `json:"description,omitempty"`
	DTOName         string                     `json:"dto_name,omitempty"`
	FieldsText      string                     `json:"fields_text,omitempty"`
	ConstraintsText string                     `json:"constraints_text,omitempty"`
	RequestDTO      string                     `json:"request_dto,omitempty"`
	ResponseSchema  string                     `json:"response_schema,omitempty"`
	Fields          []assistant.FieldSpec      `json:"fields,omitempty"`
	Constraints     []assistant.ConstraintSpec `json:"constraints,omitempty"`
}

type Result struct {
	Request      assistant.GenerateRequest `json:"request"`
	Body         json.RawMessage           `json:"body"`
	Provider     string                    `json:"provider"`
	Model        string                    `json:"model,omitempty"`
	Attempts     int                       `json:"attempts"`
	Messages     []string                  `json:"messages,omitempty"`
	UsedFallback bool                      `json:"used_fallback"`
}

func Generate(ctx context.Context, cfg config.Config, input Request) (Result, error) {
	req, err := BuildRequest(input)
	if err != nil {
		return Result{}, err
	}

	messages := []string{}
	providerFallback := false
	provider, cleanup, err := providerForGeneration(ctx, cfg)
	if err != nil {
		if !isManagedProvider(cfg) {
			return Result{}, err
		}
		messages = append(messages, fmt.Sprintf("Managed runner could not start: %v", err))
		messages = append(messages, "GoFaux used deterministic local generation so you can continue working.")
		provider = assistant.TemplateProvider{}
		cleanup = func() {}
		providerFallback = true
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	result, err := provider.GenerateMock(ctx, req)
	if err != nil && isManagedProvider(cfg) {
		messages = append(messages, fmt.Sprintf("Managed runner request failed: %v", err))
		messages = append(messages, "Restarting the managed runner and retrying once.")
		cleanup()
		cleanup = nil

		retryProvider, retryCleanup, retryStartErr := providerForGeneration(ctx, cfg)
		if retryStartErr == nil {
			provider = retryProvider
			cleanup = retryCleanup
			result, err = provider.GenerateMock(ctx, req)
			if err == nil {
				messages = append(messages, "Managed runner retry succeeded.")
			}
		} else {
			messages = append(messages, fmt.Sprintf("Managed runner retry could not start: %v", retryStartErr))
		}
	}
	if err != nil {
		if isManagedProvider(cfg) {
			messages = append(messages, fmt.Sprintf("Managed runner stayed unavailable after retry: %v", err))
			messages = append(messages, "GoFaux used deterministic local generation so you can continue working.")
		} else {
			messages = append(messages, fmt.Sprintf("Provider %s failed: %v", provider.Name(), err))
			messages = append(messages, "Used deterministic local fallback.")
		}
		result, err = assistant.TemplateProvider{}.GenerateMock(ctx, req)
		providerFallback = true
	}
	if err != nil {
		return Result{}, err
	}

	body, err := mock.ValidateJSON(string(result.Body))
	if err != nil {
		repaired, repairErr := provider.RepairJSON(ctx, result.Raw, err.Error(), req)
		if repairErr == nil {
			if repairedBody, validateErr := mock.ValidateJSON(string(repaired.Body)); validateErr == nil {
				body = repairedBody
				result = repaired
				result.Attempts++
				err = nil
				messages = append(messages, "Model repaired invalid JSON.")
			}
		}
	}
	if err != nil {
		return Result{}, fmt.Errorf("generated body was not valid JSON: %w", err)
	}

	result, body, qualityMessages, usedFallback, err := ensureQuality(ctx, provider, req, result, body)
	messages = append(messages, qualityMessages...)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Request:      req,
		Body:         body,
		Provider:     result.Provider,
		Model:        result.Model,
		Attempts:     result.Attempts,
		Messages:     messages,
		UsedFallback: usedFallback || providerFallback,
	}, nil
}

func BuildRequest(input Request) (assistant.GenerateRequest, error) {
	method, endpoint, description := InferMockRequest(input.Intent)
	if strings.TrimSpace(input.Method) != "" {
		method = input.Method
	}
	if strings.TrimSpace(input.Endpoint) != "" {
		endpoint = input.Endpoint
	}
	if strings.TrimSpace(input.Description) != "" {
		description = input.Description
	}
	if strings.TrimSpace(description) == "" {
		description = strings.TrimSpace(input.Intent)
	}
	if strings.TrimSpace(description) == "" {
		description = "A realistic JSON response for local development"
	}

	fields := append([]assistant.FieldSpec{}, input.Fields...)
	if len(fields) == 0 {
		fields = ParseFields(input.FieldsText)
	}
	if len(fields) == 0 && strings.TrimSpace(input.FieldsText) != "" {
		description = strings.TrimSpace(description + ". Field hints: " + input.FieldsText)
	}
	constraints := append([]assistant.ConstraintSpec{}, input.Constraints...)
	constraints = append(constraints, assistant.ParseConstraintText(input.ConstraintsText)...)

	requestDTO, err := optionalJSON(input.RequestDTO)
	if err != nil {
		return assistant.GenerateRequest{}, fmt.Errorf("request DTO is invalid JSON: %w", err)
	}
	responseSchema, err := optionalJSON(input.ResponseSchema)
	if err != nil {
		return assistant.GenerateRequest{}, fmt.Errorf("response schema is invalid JSON: %w", err)
	}

	status := input.Status
	if status == 0 {
		status = http.StatusOK
	}

	return assistant.GenerateRequest{
		Method:         method,
		Endpoint:       endpoint,
		Status:         status,
		Description:    description,
		DTOName:        input.DTOName,
		RequestDTO:     requestDTO,
		ResponseSchema: responseSchema,
		Fields:         fields,
		Constraints:    constraints,
	}, nil
}

func ensureQuality(ctx context.Context, provider assistant.Provider, req assistant.GenerateRequest, result assistant.GenerateResult, body json.RawMessage) (assistant.GenerateResult, json.RawMessage, []string, bool, error) {
	issues := assistant.ValidateMockQuality(body, req)
	if len(issues) == 0 {
		return result, body, nil, false, nil
	}

	messages := []string{"Generated JSON needed semantic correction."}
	messages = append(messages, assistant.QualityIssueMessages(issues)...)

	repaired, repairErr := provider.RepairJSON(ctx, result.Raw, strings.Join(assistant.QualityIssueMessages(issues), "; "), req)
	if repairErr == nil {
		repairedBody, validateErr := mock.ValidateJSON(string(repaired.Body))
		if validateErr == nil {
			repairedIssues := assistant.ValidateMockQuality(repairedBody, req)
			if len(repairedIssues) == 0 {
				messages = append(messages, "Model corrected the response after a repair prompt.")
				repaired.Attempts = result.Attempts + 1
				return repaired, repairedBody, messages, false, nil
			}
			issues = repairedIssues
			messages = append(messages, assistant.QualityIssueMessages(repairedIssues)...)
		}
	}

	fallback, fallbackErr := assistant.TemplateProvider{}.GenerateMock(ctx, req)
	if fallbackErr != nil {
		return result, body, messages, false, fmt.Errorf("could not build fallback mock: %w", fallbackErr)
	}
	fallbackBody, fallbackErr := mock.ValidateJSON(string(fallback.Body))
	if fallbackErr != nil {
		return result, body, messages, false, fmt.Errorf("fallback mock was not valid JSON: %w", fallbackErr)
	}
	if fallbackIssues := assistant.ValidateMockQuality(fallbackBody, req); len(fallbackIssues) > 0 {
		return result, body, messages, false, errors.New(strings.Join(assistant.QualityIssueMessages(fallbackIssues), "; "))
	}
	messages = append(messages, "Used deterministic local fallback after model repair failed.")
	return fallback, fallbackBody, messages, true, nil
}

func providerForGeneration(ctx context.Context, cfg config.Config) (assistant.Provider, func(), error) {
	if strings.EqualFold(strings.TrimSpace(cfg.Assistant.Provider), "managed") {
		modelPath := strings.TrimSpace(cfg.Assistant.Model)
		if modelPath == "" {
			return nil, func() {}, errors.New("no downloaded model selected")
		}

		startCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
		manager := runner.NewManager(cfg.RunnerDir)
		running, err := manager.Start(startCtx, modelPath)
		cancel()
		if err != nil {
			return nil, func() {}, err
		}
		provider := assistant.NewOpenAICompatibleProvider(running.Endpoint, running.Model, "")
		provider.UseResponseFormat = false
		return provider, running.Stop, nil
	}

	return assistant.NewProvider(cfg.Assistant), func() {}, nil
}

func isManagedProvider(cfg config.Config) bool {
	return strings.EqualFold(strings.TrimSpace(cfg.Assistant.Provider), "managed")
}

func optionalJSON(value string) (json.RawMessage, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	return mock.ValidateJSON(value)
}
