package generator

import (
	"net/http"
	"strings"

	"GoFaux/internal/assistant"
)

func InferMockRequest(input string) (method, endpoint, description string) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "" {
		return http.MethodGet, "/items", "A realistic JSON response for local development"
	}

	lower := strings.ToLower(cleaned)
	method = http.MethodGet
	switch {
	case strings.HasPrefix(lower, "create "), strings.HasPrefix(lower, "add "), strings.HasPrefix(lower, "post "):
		method = http.MethodPost
	case strings.HasPrefix(lower, "update "), strings.HasPrefix(lower, "put "):
		method = http.MethodPut
	case strings.HasPrefix(lower, "delete "), strings.HasPrefix(lower, "remove "):
		method = http.MethodDelete
	}

	resource := "items"
	words := strings.Fields(lower)
	for i, word := range words {
		word = strings.Trim(word, ".,:;!?")
		if word == "list" && i+1 < len(words) {
			for j := i + 1; j < len(words); j++ {
				candidate := strings.Trim(words[j], ".,:;!?")
				if isResourceFillerWord(candidate) {
					continue
				}
				resource = sanitizePathWord(candidate)
				break
			}
			if resource != "items" {
				break
			}
		}
		switch word {
		case "people", "persons":
			resource = "people"
		case "users", "user":
			resource = "users"
		case "products", "product":
			resource = "products"
		case "orders", "order":
			resource = "orders"
		case "books", "book":
			resource = "books"
		case "cars", "car", "vehicles", "vehicle":
			resource = "cars"
		}
		if resource != "items" {
			break
		}
	}

	return method, "/" + resource, cleaned
}

func ParseFields(line string) []assistant.FieldSpec {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if !strings.Contains(line, ":") {
		return parseFieldNamesOnly(line)
	}

	var fields []assistant.FieldSpec
	for _, part := range strings.Split(line, ",") {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		name := piece
		typeHint := ""
		if left, right, ok := strings.Cut(piece, ":"); ok {
			name = strings.TrimSpace(left)
			typeHint = strings.TrimSpace(right)
		}
		if name != "" {
			fields = append(fields, assistant.FieldSpec{Name: name, Type: typeHint, Required: true})
		}
	}
	return fields
}

func parseFieldNamesOnly(line string) []assistant.FieldSpec {
	var fields []assistant.FieldSpec
	for _, part := range strings.Split(line, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if !isSimpleFieldName(name) {
			return nil
		}
		fields = append(fields, assistant.FieldSpec{Name: name, Required: true})
	}
	return fields
}

func isResourceFillerWord(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "of", "the", "a", "an", "all", "some", "many", "maybe", "with", "and", "like":
		return true
	default:
		return false
	}
}

func sanitizePathWord(value string) string {
	value = strings.ToLower(strings.Trim(value, " /.,:;!?"))
	value = strings.ReplaceAll(value, "_", "-")
	if value == "person" || value == "persons" {
		return "people"
	}
	if value == "" {
		return "items"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "items"
	}
	return b.String()
}

func isSimpleFieldName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}
