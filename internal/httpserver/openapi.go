package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"GoFaux/internal/generator"
	"GoFaux/internal/mock"
	openapiimport "GoFaux/internal/openapi"
)

type openAPIPreviewRequest struct {
	Content string `json:"content"`
}

type openAPIImportRequest struct {
	Content      string   `json:"content"`
	OperationIDs []string `json:"operation_ids,omitempty"`
	UseAI        bool     `json:"use_ai,omitempty"`
}

type openAPIImportResult struct {
	Preview  openapiimport.Preview `json:"preview"`
	Imported []mock.Definition     `json:"imported"`
	Messages []string              `json:"messages,omitempty"`
}

func (s *Server) handleAPIOpenAPIPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var input openAPIPreviewRequest
	if err := readJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid OpenAPI request: " + err.Error()})
		return
	}
	preview, err := openapiimport.Parse(input.Content)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleAPIOpenAPIImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var input openAPIImportRequest
	if err := readJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid OpenAPI request: " + err.Error()})
		return
	}
	preview, err := openapiimport.Parse(input.Content)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	selected := selectedOperationIDs(input.OperationIDs)
	cfg := s.configSnapshot()
	if !input.UseAI {
		cfg.Assistant.Provider = "template"
		cfg.Assistant.Model = ""
		cfg.Assistant.Endpoint = ""
	}

	var imported []mock.Definition
	var messages []string
	for _, operation := range preview.Operations {
		if len(selected) > 0 && !selected[operation.ID] {
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		result, err := generator.Generate(ctx, cfg, generatorRequestFromOperation(operation))
		cancel()
		if err != nil {
			messages = append(messages, operation.ID+": "+err.Error())
			continue
		}
		def, err := s.store.Add(mock.Definition{
			Name:        mockNameFromOperation(operation),
			Description: operation.Intent,
			Method:      operation.Method,
			Endpoint:    operation.Path,
			Status:      operation.Status,
			Headers:     map[string]string{"Content-Type": "application/json"},
			Body:        result.Body,
			Tags:        []string{"openapi", safeTag(preview.Title)},
		})
		if err != nil {
			messages = append(messages, operation.ID+": "+err.Error())
			continue
		}
		imported = append(imported, def)
		if result.UsedFallback {
			messages = append(messages, operation.ID+": deterministic fallback was used")
		}
		messages = append(messages, result.Messages...)
	}
	writeJSON(w, http.StatusOK, openAPIImportResult{Preview: preview, Imported: imported, Messages: compactMessages(messages)})
}

func generatorRequestFromOperation(operation openapiimport.Operation) generator.Request {
	return generator.Request{
		Intent:         operation.Intent,
		Method:         operation.Method,
		Endpoint:       operation.Path,
		Status:         operation.Status,
		Name:           mockNameFromOperation(operation),
		Description:    operation.Intent,
		DTOName:        operation.DTOName,
		ResponseSchema: string(operation.ResponseSchema),
		Fields:         operation.Fields,
		Constraints:    operation.Constraints,
	}
}

func selectedOperationIDs(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	selected := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			selected[id] = true
		}
	}
	return selected
}

func mockNameFromOperation(operation openapiimport.Operation) string {
	if strings.TrimSpace(operation.Summary) != "" {
		return operation.Summary
	}
	if strings.TrimSpace(operation.ID) != "" {
		return operation.ID
	}
	return operation.Method + " " + operation.Path
}

func safeTag(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "contract"
	}
	return value
}

func compactMessages(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
