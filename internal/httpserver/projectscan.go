package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"GoFaux/internal/generator"
	"GoFaux/internal/mock"
	"GoFaux/internal/projectscan"
)

type projectScanPreviewRequest struct {
	Path string `json:"path"`
}

type projectScanImportRequest struct {
	Path           string   `json:"path"`
	IntegrationIDs []string `json:"integration_ids,omitempty"`
	UseAI          bool     `json:"use_ai,omitempty"`
}

type projectScanImportResult struct {
	Preview  projectscan.Preview `json:"preview"`
	Imported []mock.Definition   `json:"imported"`
	Messages []string            `json:"messages,omitempty"`
}

func (s *Server) handleAPIProjectScanPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var input projectScanPreviewRequest
	if err := readJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid project scan request: " + err.Error()})
		return
	}
	preview, err := projectscan.Scan(input.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleAPIProjectScanImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var input projectScanImportRequest
	if err := readJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid project scan request: " + err.Error()})
		return
	}
	preview, err := projectscan.Scan(input.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	selected := selectedProjectIntegrationIDs(input.IntegrationIDs, preview.Integrations)
	cfg := s.configSnapshot()
	if !input.UseAI {
		cfg.Assistant.Provider = "template"
		cfg.Assistant.Model = ""
		cfg.Assistant.Endpoint = ""
	}

	var imported []mock.Definition
	var messages []string
	for _, integration := range preview.Integrations {
		if !selected[integration.ID] {
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		result, err := generator.Generate(ctx, cfg, generatorRequestFromIntegration(integration))
		cancel()
		if err != nil {
			messages = append(messages, integration.ID+": "+err.Error())
			continue
		}
		def, err := s.store.Add(mock.Definition{
			Name:        mockNameFromIntegration(integration),
			Description: mockDescriptionFromIntegration(integration),
			Method:      integration.Method,
			Endpoint:    integration.Endpoint,
			Status:      http.StatusOK,
			Headers:     map[string]string{"Content-Type": "application/json"},
			Body:        result.Body,
			Tags:        mockTagsFromIntegration(integration),
		})
		if err != nil {
			messages = append(messages, integration.ID+": "+err.Error())
			continue
		}
		imported = append(imported, def)
		if result.UsedFallback {
			messages = append(messages, integration.ID+": deterministic fallback was used")
		}
		messages = append(messages, result.Messages...)
	}
	if len(imported) == 0 && len(messages) == 0 {
		messages = append(messages, "No integrations were selected for import.")
	}
	writeJSON(w, http.StatusOK, projectScanImportResult{Preview: preview, Imported: imported, Messages: compactMessages(messages)})
}

func selectedProjectIntegrationIDs(ids []string, integrations []projectscan.Integration) map[string]bool {
	selected := map[string]bool{}
	if ids != nil {
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				selected[id] = true
			}
		}
		return selected
	}
	for _, integration := range integrations {
		if integration.External {
			selected[integration.ID] = true
		}
	}
	return selected
}

func generatorRequestFromIntegration(integration projectscan.Integration) generator.Request {
	dtoName := firstNonEmptyString(integration.ResponseDTO, resourceNameFromEndpoint(integration.Endpoint))
	description := fmt.Sprintf(
		"Generate a realistic JSON response body for a discovered %s API integration. The client calls %s %s. Source kind: %s. Base URL: %s. If this is a list endpoint, return multiple realistic items. Return only the response JSON body and do not include request metadata.",
		integration.Direction,
		integration.Method,
		integration.Endpoint,
		integration.Kind,
		firstNonEmptyString(integration.BaseURL, "not detected"),
	)
	if integration.ResponseDTO != "" {
		description += " Expected response DTO: " + integration.ResponseDTO + "."
	}
	if integration.RequestDTO != "" {
		description += " Request DTO seen in code: " + integration.RequestDTO + "."
	}
	return generator.Request{
		Intent:      description,
		Method:      integration.Method,
		Endpoint:    integration.Endpoint,
		Status:      http.StatusOK,
		Name:        mockNameFromIntegration(integration),
		Description: description,
		DTOName:     dtoName,
	}
}

func mockNameFromIntegration(integration projectscan.Integration) string {
	if strings.TrimSpace(integration.Name) != "" {
		return integration.Name
	}
	return integration.Method + " " + integration.Endpoint
}

func mockDescriptionFromIntegration(integration projectscan.Integration) string {
	parts := []string{
		"Auto-discovered by project scan.",
		fmt.Sprintf("Source: %s:%d.", integration.SourceFile, integration.Line),
		fmt.Sprintf("Kind: %s.", integration.Kind),
	}
	if integration.BaseURL != "" {
		parts = append(parts, "Base URL: "+integration.BaseURL+".")
	}
	if integration.RequestDTO != "" {
		parts = append(parts, "Request DTO: "+integration.RequestDTO+".")
	}
	if integration.ResponseDTO != "" {
		parts = append(parts, "Response DTO: "+integration.ResponseDTO+".")
	}
	return strings.Join(parts, " ")
}

func mockTagsFromIntegration(integration projectscan.Integration) []string {
	tags := []string{"project-scan", safeTag(integration.Kind), safeTag(integration.Direction)}
	if integration.External {
		tags = append(tags, "external")
	} else {
		tags = append(tags, "internal")
	}
	tags = append(tags, integration.Tags...)
	return compactMessages(tags)
}

func resourceNameFromEndpoint(endpoint string) string {
	endpoint = strings.Trim(endpoint, "/")
	if endpoint == "" {
		return "item"
	}
	parts := strings.Split(endpoint, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || strings.HasPrefix(part, "{") || strings.HasPrefix(part, ":") {
			continue
		}
		return part
	}
	return "item"
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
