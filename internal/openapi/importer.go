package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"GoFaux/internal/assistant"

	"gopkg.in/yaml.v3"
)

type Preview struct {
	Title      string      `json:"title,omitempty"`
	Version    string      `json:"version,omitempty"`
	OpenAPI    string      `json:"openapi,omitempty"`
	Operations []Operation `json:"operations"`
}

type Operation struct {
	ID             string                     `json:"id"`
	Method         string                     `json:"method"`
	Path           string                     `json:"path"`
	Summary        string                     `json:"summary,omitempty"`
	Description    string                     `json:"description,omitempty"`
	Status         int                        `json:"status"`
	DTOName        string                     `json:"dto_name,omitempty"`
	RequestSchema  json.RawMessage            `json:"request_schema,omitempty"`
	ResponseSchema json.RawMessage            `json:"response_schema,omitempty"`
	Fields         []assistant.FieldSpec      `json:"fields,omitempty"`
	Constraints    []assistant.ConstraintSpec `json:"constraints,omitempty"`
	Intent         string                     `json:"intent"`
}

func Parse(content string) (Preview, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Preview{}, errors.New("OpenAPI content is empty")
	}

	var raw any
	if strings.HasPrefix(content, "{") {
		if err := json.Unmarshal([]byte(content), &raw); err != nil {
			return Preview{}, fmt.Errorf("parse OpenAPI JSON: %w", err)
		}
	} else if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return Preview{}, fmt.Errorf("parse OpenAPI YAML: %w", err)
	}
	doc, ok := normalizeValue(raw).(map[string]any)
	if !ok {
		return Preview{}, errors.New("OpenAPI document must be an object")
	}

	preview := Preview{
		Title:   stringAtMap(doc, "info", "title"),
		Version: stringAtMap(doc, "info", "version"),
		OpenAPI: firstString(doc["openapi"], doc["swagger"]),
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		return Preview{}, errors.New("OpenAPI document has no paths")
	}

	pathNames := sortedKeys(paths)
	for _, path := range pathNames {
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete", "options", "head"} {
			rawOperation, ok := pathItem[method].(map[string]any)
			if !ok {
				continue
			}
			operation := buildOperation(doc, strings.ToUpper(method), path, rawOperation)
			preview.Operations = append(preview.Operations, operation)
		}
	}
	return preview, nil
}

func buildOperation(doc map[string]any, method, path string, operation map[string]any) Operation {
	status, responseSchema := responseSchemaForOperation(doc, operation)
	requestSchema := requestSchemaForOperation(doc, operation)
	fields := fieldsFromSchema(responseSchema)
	constraints := assistant.ConstraintsFromSchema(responseSchema)
	summary, _ := operation["summary"].(string)
	description, _ := operation["description"].(string)
	id, _ := operation["operationId"].(string)
	if strings.TrimSpace(id) == "" {
		id = operationID(method, path)
	}
	intent := intentForOperation(method, path, summary, description, responseSchema)
	dtoName := dtoNameForOperation(id, path)
	return Operation{
		ID:             id,
		Method:         method,
		Path:           normalizeOpenAPIPath(path),
		Summary:        summary,
		Description:    description,
		Status:         status,
		DTOName:        dtoName,
		RequestSchema:  requestSchema,
		ResponseSchema: responseSchema,
		Fields:         fields,
		Constraints:    constraints,
		Intent:         intent,
	}
}

func responseSchemaForOperation(doc map[string]any, operation map[string]any) (int, json.RawMessage) {
	responses, _ := operation["responses"].(map[string]any)
	statusKeys := sortedKeys(responses)
	chosenStatus := 200
	var chosen any
	for _, key := range statusKeys {
		if len(key) == 3 && key[0] == '2' {
			chosenStatus = statusCode(key, http.StatusOK)
			chosen = responses[key]
			break
		}
	}
	if chosen == nil {
		for _, key := range statusKeys {
			chosenStatus = statusCode(key, http.StatusOK)
			chosen = responses[key]
			break
		}
	}
	schema := schemaFromContent(chosen)
	schema = resolveSchema(doc, schema, map[string]bool{})
	return chosenStatus, rawJSON(schema)
}

func requestSchemaForOperation(doc map[string]any, operation map[string]any) json.RawMessage {
	requestBody, _ := operation["requestBody"].(map[string]any)
	schema := schemaFromContent(requestBody)
	schema = resolveSchema(doc, schema, map[string]bool{})
	return rawJSON(schema)
}

func schemaFromContent(value any) any {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if schema, ok := object["schema"]; ok {
		return schema
	}
	content, _ := object["content"].(map[string]any)
	if len(content) == 0 {
		return nil
	}
	for _, mediaType := range []string{"application/json", "application/problem+json"} {
		if media, ok := content[mediaType].(map[string]any); ok {
			return media["schema"]
		}
	}
	for _, media := range content {
		if mediaObject, ok := media.(map[string]any); ok {
			return mediaObject["schema"]
		}
	}
	return nil
}

func resolveSchema(doc map[string]any, schema any, seen map[string]bool) any {
	switch typed := schema.(type) {
	case map[string]any:
		if ref, _ := typed["$ref"].(string); ref != "" {
			if seen[ref] {
				return typed
			}
			seen[ref] = true
			if resolved, ok := resolveRef(doc, ref); ok {
				return resolveSchema(doc, resolved, seen)
			}
		}
		out := map[string]any{}
		for key, value := range typed {
			out[key] = resolveSchema(doc, value, seen)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = resolveSchema(doc, item, seen)
		}
		return out
	default:
		return schema
	}
}

func resolveRef(doc map[string]any, ref string) (any, bool) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, false
	}
	var current any = doc
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func fieldsFromSchema(raw json.RawMessage) []assistant.FieldSpec {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var schema any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil
	}
	return fieldsFromSchemaValue(schema)
}

func fieldsFromSchemaValue(schema any) []assistant.FieldSpec {
	object, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	if typ, _ := object["type"].(string); typ == "array" {
		return fieldsFromSchemaValue(object["items"])
	}
	properties, _ := object["properties"].(map[string]any)
	if itemSchema, ok := arrayItemsProperty(properties, "items"); ok {
		return fieldsFromSchemaValue(itemSchema)
	}
	required := stringSet(object["required"])
	keys := sortedKeys(properties)
	fields := make([]assistant.FieldSpec, 0, len(keys))
	for _, key := range keys {
		prop, _ := properties[key].(map[string]any)
		fields = append(fields, assistant.FieldSpec{
			Name:        key,
			Type:        schemaType(prop),
			Description: firstString(prop["description"], prop["format"]),
			Required:    required[key],
		})
	}
	return fields
}

func arrayItemsProperty(properties map[string]any, key string) (any, bool) {
	if len(properties) == 0 {
		return nil, false
	}
	prop, ok := properties[key].(map[string]any)
	if !ok {
		return nil, false
	}
	if typ, _ := prop["type"].(string); typ != "array" {
		return nil, false
	}
	return prop["items"], prop["items"] != nil
}

func schemaType(prop map[string]any) string {
	if prop == nil {
		return ""
	}
	if typ, _ := prop["type"].(string); typ != "" {
		return typ
	}
	if _, ok := prop["properties"].(map[string]any); ok {
		return "object"
	}
	return ""
}

func intentForOperation(method, path, summary, description string, schema json.RawMessage) string {
	parts := []string{}
	if summary != "" {
		parts = append(parts, summary)
	}
	if description != "" {
		parts = append(parts, description)
	}
	if len(parts) == 0 {
		resource := strings.Trim(strings.ReplaceAll(path, "{", ""), "/")
		if method == http.MethodGet && !strings.Contains(path, "{") {
			parts = append(parts, "list "+resource)
		} else {
			parts = append(parts, method+" "+path+" response")
		}
	}
	if looksLikeArraySchema(schema) && !strings.Contains(strings.ToLower(strings.Join(parts, " ")), "list") {
		parts = append([]string{"list response"}, parts...)
	}
	return strings.Join(parts, ". ")
}

func looksLikeArraySchema(raw json.RawMessage) bool {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return false
	}
	typ, _ := object["type"].(string)
	return typ == "array"
}

func operationID(method, path string) string {
	clean := strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_").Replace(strings.Trim(path, "/"))
	clean = strings.Trim(clean, "_")
	if clean == "" {
		clean = "root"
	}
	return strings.ToLower(method) + "_" + clean
}

func dtoNameForOperation(id, path string) string {
	if id != "" {
		parts := strings.FieldsFunc(id, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
		if len(parts) > 0 {
			return strings.ToLower(parts[len(parts)-1])
		}
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return "response"
	}
	parts := strings.Split(path, "/")
	return strings.Trim(parts[len(parts)-1], "{}")
}

func normalizeOpenAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func rawJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	content, err := json.Marshal(value)
	if err != nil || string(content) == "null" {
		return nil
	}
	return json.RawMessage(content)
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			out[key] = normalizeValue(item)
		}
		return out
	case map[any]any:
		out := map[string]any{}
		for key, item := range typed {
			out[fmt.Sprint(key)] = normalizeValue(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeValue(item)
		}
		return out
	default:
		return value
	}
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstString(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func stringAtMap(object map[string]any, path ...string) string {
	var current any = object
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	text, _ := current.(string)
	return text
}

func statusCode(value string, fallback int) int {
	var code int
	if _, err := fmt.Sscanf(value, "%d", &code); err != nil || code < 100 || code > 599 {
		return fallback
	}
	return code
}

func stringSet(value any) map[string]bool {
	set := map[string]bool{}
	array, _ := value.([]any)
	for _, item := range array {
		if text, ok := item.(string); ok {
			set[text] = true
		}
	}
	return set
}
