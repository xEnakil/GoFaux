package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type TemplateProvider struct{}

func (TemplateProvider) Name() string {
	return "template"
}

func (TemplateProvider) ListModels(_ context.Context) ([]ModelInfo, error) {
	return []ModelInfo{
		{
			Provider: "template",
			Name:     "deterministic-template",
		},
	}, nil
}

func (TemplateProvider) GenerateMock(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	body, err := BuildTemplateJSON(req)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{
		Provider: "template",
		Model:    "deterministic",
		Body:     body,
		Raw:      string(body),
		Attempts: 1,
	}, nil
}

func (p TemplateProvider) RepairJSON(ctx context.Context, _ string, _ string, req GenerateRequest) (GenerateResult, error) {
	return p.GenerateMock(ctx, req)
}

func BuildTemplateJSON(req GenerateRequest) (json.RawMessage, error) {
	fields := inferResponseFields(req)
	rules := inferValueRules(req.Description)
	constraints := InferConstraints(req)
	if isListRequest(req) {
		return buildListTemplateJSON(req, fields, defaultItemCount(req), rules, constraints)
	}

	object := map[string]any{}
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		object[name] = sampleValueAtIndex(name, field.Type, 0, endpointResource(req.Endpoint), rules, constraints)
	}

	if len(object) == 0 {
		object["id"] = 1
		object["message"] = "Mock response for " + strings.ToUpper(req.Method) + " " + req.Endpoint
		object["description"] = req.Description
		object["success"] = req.Status < 400 || req.Status == 0
	}

	return marshalTemplate(object)
}

func buildListTemplateJSON(req GenerateRequest, fields []FieldSpec, count int, rules valueRules, constraints []ConstraintSpec) (json.RawMessage, error) {
	count = clampItemCount(count)
	resource := endpointResource(req.Endpoint)
	items := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		item := map[string]any{}
		for _, field := range fields {
			name := strings.TrimSpace(field.Name)
			if name == "" {
				continue
			}
			item[name] = sampleValueAtIndex(name, field.Type, i, resource, rules, constraints)
		}
		if len(item) == 0 {
			item["id"] = i + 1
			item["name"] = sampleResourceName(resource, i)
		}
		items = append(items, item)
	}

	return marshalTemplate(map[string]any{
		"items": items,
		"total": len(items),
	})
}

func marshalTemplate(value any) (json.RawMessage, error) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func inferResponseFields(req GenerateRequest) []FieldSpec {
	fields := normalizeFields(req.Fields)
	if len(fields) > 0 {
		return appendDescriptionFields(fields, req.Description)
	}

	fields = fieldsFromRequestDTO(req.RequestDTO)
	if len(fields) > 0 {
		return appendDescriptionFields(fields, req.Description)
	}

	resource := endpointResource(req.Endpoint)
	fields = defaultFieldsForResource(resource)
	fields = appendDescriptionFields(fields, req.Description)
	return fields
}

func normalizeFields(input []FieldSpec) []FieldSpec {
	out := make([]FieldSpec, 0, len(input))
	seen := map[string]bool{}
	for _, field := range input {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		field.Name = name
		out = append(out, field)
	}
	return out
}

func fieldsFromRequestDTO(raw json.RawMessage) []FieldSpec {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var dto map[string]any
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil
	}

	keys := make([]string, 0, len(dto))
	for key := range dto {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fields := make([]FieldSpec, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, FieldSpec{Name: key, Type: typeHintFromValue(dto[key]), Required: true})
	}
	return fields
}

func typeHintFromValue(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case float64, int, int64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "string"
	}
}

func defaultFieldsForResource(resource string) []FieldSpec {
	switch singularResource(resource) {
	case "person", "user", "customer", "employee", "author":
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string"},
		}
	case "product", "item":
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "price", Type: "number"},
			{Name: "inStock", Type: "bool"},
		}
	case "order", "invoice", "payment":
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "orderNumber", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "total", Type: "number"},
			{Name: "createdAt", Type: "string"},
		}
	case "book":
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "title", Type: "string", Required: true},
			{Name: "author", Type: "string"},
			{Name: "publishedYear", Type: "int"},
		}
	case "car", "vehicle":
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "make", Type: "string"},
			{Name: "model", Type: "string"},
			{Name: "year", Type: "int"},
			{Name: "color", Type: "string"},
		}
	default:
		return []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "status", Type: "string"},
		}
	}
}

func appendDescriptionFields(fields []FieldSpec, description string) []FieldSpec {
	lower := strings.ToLower(description)
	candidates := []struct {
		contains []string
		field    FieldSpec
	}{
		{[]string{"age"}, FieldSpec{Name: "age", Type: "int"}},
		{[]string{"gender", "sex"}, FieldSpec{Name: "gender", Type: "string"}},
		{[]string{"hair"}, FieldSpec{Name: "hairColor", Type: "string"}},
		{[]string{"email", "e-mail"}, FieldSpec{Name: "email", Type: "string"}},
		{[]string{"phone", "mobile"}, FieldSpec{Name: "phone", Type: "string"}},
		{[]string{"address"}, FieldSpec{Name: "address", Type: "string"}},
		{[]string{"nationality", "citizenship"}, FieldSpec{Name: "nationality", Type: "string"}},
		{[]string{"passport"}, FieldSpec{Name: "passportNumber", Type: "string"}},
		{[]string{"percentage", "percent"}, FieldSpec{Name: "percentage", Type: "number"}},
		{[]string{"status"}, FieldSpec{Name: "status", Type: "string"}},
		{[]string{"description", "details"}, FieldSpec{Name: "description", Type: "string"}},
		{[]string{"price", "cost"}, FieldSpec{Name: "price", Type: "number"}},
		{[]string{"amount", "total"}, FieldSpec{Name: "total", Type: "number"}},
		{[]string{"date", "created"}, FieldSpec{Name: "createdAt", Type: "string"}},
	}

	for _, candidate := range candidates {
		for _, token := range candidate.contains {
			if strings.Contains(lower, token) {
				fields = appendFieldIfMissing(fields, candidate.field)
				break
			}
		}
	}
	return fields
}

func appendFieldIfMissing(fields []FieldSpec, field FieldSpec) []FieldSpec {
	for _, existing := range fields {
		if strings.EqualFold(existing.Name, field.Name) {
			return fields
		}
	}
	return append(fields, field)
}

func looksLikePeopleList(description, endpoint string) bool {
	value := strings.ToLower(description + " " + endpoint)
	return strings.Contains(value, "people") || strings.Contains(value, "person") || strings.Contains(value, "user")
}

func BuildPeopleListJSON(count int) (json.RawMessage, error) {
	return buildListTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: fmt.Sprintf("list %d people with name, age, gender, hair color and email", count),
	}, []FieldSpec{
		{Name: "id", Type: "int", Required: true},
		{Name: "name", Type: "string", Required: true},
		{Name: "age", Type: "int"},
		{Name: "gender", Type: "string"},
		{Name: "hairColor", Type: "string"},
		{Name: "email", Type: "string"},
	}, count, inferValueRules("list people with name, age, gender, hair color and email"), nil)
}

func sampleValueAtIndex(name, typeHint string, index int, resource string, rules valueRules, constraints []ConstraintSpec) any {
	lower := strings.ToLower(name)
	if value, ok := constrainedSampleValue(name, typeHint, index, resource, constraints); ok {
		return value
	}
	switch {
	case strings.Contains(lower, "percentage"), strings.Contains(lower, "percent"):
		min, max := rules.PercentageMin, rules.PercentageMax
		if min == 0 && max == 0 {
			max = 100
		}
		if max < min {
			max = min
		}
		return min + (index*7)%(max-min+1)
	case strings.Contains(lower, "passport"):
		return passportValue(index, rules)
	case strings.Contains(lower, "nationality"):
		return []string{"American", "British", "French", "Canadian", "German", "Dutch", "Spanish", "Italian", "Polish", "Swedish"}[index%10]
	}
	switch strings.ToLower(typeHint) {
	case "int", "integer":
		if strings.Contains(lower, "age") {
			return 18 + (index*7)%47
		}
		if strings.Contains(lower, "year") {
			return 2015 + index%11
		}
		return index + 1
	case "float", "decimal", "number":
		return float64(999+index*137) / 100
	case "bool", "boolean":
		return index%2 == 0
	case "array", "list":
		return []any{}
	case "object":
		return map[string]any{}
	}

	switch {
	case lower == "id" || strings.HasSuffix(lower, "id"):
		return index + 1
	case strings.Contains(lower, "email"):
		return emailFromName(samplePersonName(index))
	case strings.Contains(lower, "firstname"):
		return strings.Fields(samplePersonName(index))[0]
	case strings.Contains(lower, "lastname"):
		parts := strings.Fields(samplePersonName(index))
		return parts[len(parts)-1]
	case strings.Contains(lower, "name"):
		return sampleResourceName(resource, index)
	case strings.Contains(lower, "title"):
		return sampleTitle(resource, index)
	case strings.Contains(lower, "author"):
		return samplePersonName(index + 4)
	case strings.Contains(lower, "age"):
		return 18 + (index*7)%47
	case strings.Contains(lower, "nationality"):
		return []string{"American", "British", "French", "Canadian", "German", "Dutch", "Spanish", "Italian", "Polish", "Swedish"}[index%10]
	case strings.Contains(lower, "passport"):
		return passportValue(index, rules)
	case strings.Contains(lower, "gender"):
		return []string{"female", "male", "non-binary"}[index%3]
	case strings.Contains(lower, "hair"):
		return []string{"brown", "black", "blonde", "auburn", "dark brown", "red", "gray", "chestnut"}[(index*3)%8]
	case strings.Contains(lower, "phone"):
		return fmt.Sprintf("+1-555-010%d", index%10)
	case strings.Contains(lower, "address"):
		return fmt.Sprintf("%d Example Street", 100+index)
	case strings.Contains(lower, "date"), strings.Contains(lower, "time"), strings.Contains(lower, "at"):
		return fmt.Sprintf("2026-05-%02dT12:00:00Z", 1+index%28)
	case strings.Contains(lower, "percentage"), strings.Contains(lower, "percent"):
		min, max := rules.PercentageMin, rules.PercentageMax
		if min == 0 && max == 0 {
			max = 100
		}
		if max < min {
			max = min
		}
		return min + (index*7)%(max-min+1)
	case strings.Contains(lower, "price"), strings.Contains(lower, "amount"), strings.Contains(lower, "total"):
		return float64(999+index*137) / 100
	case strings.HasPrefix(lower, "is"), strings.HasPrefix(lower, "has"), strings.Contains(lower, "stock"):
		return index%2 == 0
	case strings.Contains(lower, "status"):
		return []string{"active", "pending", "completed", "inactive"}[index%4]
	case strings.Contains(lower, "color"):
		return []string{"black", "white", "blue", "green", "silver"}[index%5]
	case strings.Contains(lower, "make"):
		return []string{"Toyota", "Ford", "Honda", "Volkswagen", "Hyundai"}[index%5]
	case strings.Contains(lower, "model"):
		return []string{"Astra", "Civic", "Focus", "Corolla", "Tucson"}[index%5]
	case strings.Contains(lower, "items"):
		return []any{}
	default:
		return sampleValue(name, typeHint)
	}
}

func constrainedSampleValue(name, typeHint string, index int, resource string, constraints []ConstraintSpec) (any, bool) {
	for _, constraint := range constraints {
		if !constraintMatchesField(constraint, name) {
			continue
		}
		switch constraint.Kind {
		case "required", "type":
			continue
		case "pattern":
			if constraint.Prefix != "" && constraint.Digits > 0 {
				return fmt.Sprintf("%s%0*d", constraint.Prefix, constraint.Digits, 1000000+index), true
			}
		case "range":
			min, max := 0.0, 100.0
			if constraint.Min != nil {
				min = *constraint.Min
			}
			if constraint.Max != nil {
				max = *constraint.Max
			}
			if max < min {
				max = min
			}
			span := int(max - min + 1)
			if span < 1 {
				span = 1
			}
			value := min + float64((index*7)%span)
			if normalizeConstraintType(typeHint) == "integer" || strings.Contains(strings.ToLower(name), "age") {
				return int(value), true
			}
			return value, true
		case "email":
			return emailFromName(samplePersonName(index)), true
		case "enum":
			if len(constraint.Values) > 0 {
				return constraint.Values[index%len(constraint.Values)], true
			}
		}
	}
	for _, constraint := range constraints {
		if !constraintMatchesField(constraint, name) {
			continue
		}
		switch constraint.Kind {
		case "type":
			switch constraint.Type {
			case "integer":
				return index + 1, true
			case "number":
				return float64(999+index*137) / 100, true
			case "boolean":
				return index%2 == 0, true
			case "array":
				return []any{}, true
			case "object":
				return map[string]any{}, true
			}
		}
	}
	return nil, false
}

func passportValue(index int, rules valueRules) string {
	prefix := rules.PassportPrefix
	if prefix == "" {
		prefix = "P"
	}
	digits := rules.PassportDigits
	if digits <= 0 {
		digits = 7
	}
	return fmt.Sprintf("%s%0*d", prefix, digits, 1000000+index)
}

func sampleResourceName(resource string, index int) string {
	switch singularResource(resource) {
	case "person", "user", "customer", "employee", "author":
		return samplePersonName(index)
	case "product", "item":
		return []string{"Wireless Mouse", "USB-C Hub", "Desk Lamp", "Notebook", "Travel Mug"}[index%5]
	case "book":
		return sampleTitle(resource, index)
	case "order":
		return fmt.Sprintf("Order %04d", index+1)
	default:
		name := strings.ReplaceAll(singularResource(resource), "_", " ")
		if name == "" {
			name = "item"
		}
		return fmt.Sprintf("%s %d", titleWords(name), index+1)
	}
}

func samplePersonName(index int) string {
	names := []string{
		"Ava Johnson", "Noah Smith", "Mia Brown", "Liam Davis", "Sophia Wilson",
		"Ethan Miller", "Isabella Moore", "Lucas Taylor", "Amelia Anderson", "Mason Thomas",
		"Harper Jackson", "Elijah White", "Evelyn Harris", "James Martin", "Charlotte Thompson",
		"Benjamin Garcia", "Luna Martinez", "Henry Robinson", "Aria Clark", "Daniel Lewis",
	}
	return names[index%len(names)]
}

func sampleTitle(resource string, index int) string {
	titles := []string{
		"The Local Service Handbook",
		"Designing Reliable APIs",
		"Mock Data in Practice",
		"Offline Development Patterns",
		"Applied Backend Testing",
	}
	return titles[index%len(titles)]
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func emailFromName(name string) string {
	value := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	return value + "@example.com"
}

func sampleValue(name, typeHint string) any {
	lower := strings.ToLower(name)
	switch strings.ToLower(typeHint) {
	case "int", "integer", "number":
		return 1
	case "float", "decimal":
		return 49.99
	case "bool", "boolean":
		return true
	case "array", "list":
		return []any{}
	case "object":
		return map[string]any{}
	}

	switch {
	case strings.Contains(lower, "id"):
		return 1
	case strings.Contains(lower, "email"):
		return "user@example.com"
	case strings.Contains(lower, "name"):
		return "Example User"
	case strings.Contains(lower, "date"), strings.Contains(lower, "time"), strings.Contains(lower, "at"):
		return "2026-05-13T12:00:00Z"
	case strings.Contains(lower, "price"), strings.Contains(lower, "amount"), strings.Contains(lower, "total"):
		return 49.99
	case strings.HasPrefix(lower, "is_"), strings.HasPrefix(lower, "has_"):
		return true
	case strings.Contains(lower, "status"):
		return "active"
	case strings.Contains(lower, "items"):
		return []any{}
	default:
		return "example"
	}
}
