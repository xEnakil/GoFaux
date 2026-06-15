package assistant

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

func buildGenerationPrompt(req GenerateRequest) (string, error) {
	dto, err := promptDTO(req)
	if err != nil {
		return "", err
	}
	return "You are GoFaux's local mock response generator.\n" +
		"Read the request details and create the HTTP response body JSON.\n" +
		"If the intent describes a list or collection, return a JSON object with an items array and a total number.\n" +
		"When an exact item count is provided, the items array must contain exactly that many objects.\n" +
		"If the intent describes one resource or action result, return one realistic object.\n" +
		"Infer field names and field values from the intent, endpoint, field hints, DTO sample, schema, and value constraints.\n" +
		"Return only valid JSON. Do not include markdown, comments, explanations, HTTP headers, status code, endpoint, method, user intent, or request details.\n\n" +
		"REQUEST DETAILS:\n" + string(dto), nil
}

func buildRepairPrompt(req GenerateRequest, invalid string, validationError string) (string, error) {
	dto, err := promptDTO(req)
	if err != nil {
		return "", err
	}
	return "Create a corrected mock API response body JSON from the request details below.\n" +
		"The previous answer was not acceptable: " + validationError + "\n" +
		"If the previous answer copied request details or violated constraints, ignore it and generate the actual response body instead.\n" +
		"Return only one valid JSON object or JSON array. Do not include markdown, comments, explanations, HTTP headers, status code, endpoint, method, user intent, or request details.\n\n" +
		"REQUEST DETAILS:\n" + string(dto) + "\n\n" +
		"PREVIOUS ANSWER:\n" + invalid, nil
}

func buildCompactGenerationPrompt(req GenerateRequest) (string, error) {
	var b strings.Builder
	b.WriteString("Return only raw valid JSON for a mock API response body.\n")
	b.WriteString("No markdown. No code fences. No explanation. Do not copy request metadata.\n")
	b.WriteString("Intent: ")
	b.WriteString(req.Description)
	b.WriteString("\nHTTP: ")
	b.WriteString(strings.ToUpper(req.Method))
	b.WriteString(" ")
	b.WriteString(req.Endpoint)
	b.WriteString("\n")

	if isListRequest(req) {
		count := defaultItemCount(req)
		b.WriteString("Shape: JSON object with keys items and total. items must contain ")
		b.WriteString(strconv.Itoa(count))
		b.WriteString(" objects. total must equal ")
		b.WriteString(strconv.Itoa(count))
		b.WriteString(".\n")
	} else {
		b.WriteString("Shape: one JSON object.\n")
	}

	fields := inferResponseFields(req)
	if len(fields) > 0 {
		b.WriteString("Fields: ")
		for i, field := range fields {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(field.Name)
			if strings.TrimSpace(field.Type) != "" {
				b.WriteString(" ")
				b.WriteString(field.Type)
			}
		}
		b.WriteString(".\n")
	}

	if constraints := promptValueConstraints(req.Description); len(constraints) > 0 {
		b.WriteString("Value constraints: ")
		b.WriteString(strings.Join(constraints, "; "))
		b.WriteString(".\n")
	}
	if constraints := ConstraintSummaries(req); len(constraints) > 0 {
		b.WriteString("Constraint rules: ")
		b.WriteString(strings.Join(constraints, "; "))
		b.WriteString(".\n")
	}
	if len(req.RequestDTO) > 0 && json.Valid(req.RequestDTO) {
		b.WriteString("Request DTO context: ")
		b.WriteString(compactPromptJSON(req.RequestDTO))
		b.WriteString("\n")
	}
	if len(req.ResponseSchema) > 0 && json.Valid(req.ResponseSchema) {
		b.WriteString("Response schema: ")
		b.WriteString(compactPromptJSON(req.ResponseSchema))
		b.WriteString("\n")
	}
	b.WriteString("First character must be { or [. Last character must be } or ].")
	return b.String(), nil
}

func promptDTO(req GenerateRequest) ([]byte, error) {
	dto := map[string]any{
		"http_method":   strings.ToUpper(req.Method),
		"endpoint":      req.Endpoint,
		"status_code":   req.Status,
		"user_intent":   req.Description,
		"response_kind": responseKind(req),
	}
	if isListRequest(req) {
		dto["response_contract"] = map[string]any{
			"top_level": "object",
			"items":     "array of response DTO objects",
			"total":     "number equal to length of items",
		}
		if count := requestedItemCount(req); count > 0 {
			dto["exact_item_count"] = count
		} else {
			dto["suggested_item_count"] = defaultItemCount(req)
			dto["minimum_item_count"] = 3
			dto["item_count_guidance"] = "choose a natural small sample size with multiple objects"
		}
	} else {
		dto["response_contract"] = "single JSON object"
	}
	if constraints := promptValueConstraints(req.Description); len(constraints) > 0 {
		dto["value_constraints"] = constraints
	}
	if constraints := ConstraintSummaries(req); len(constraints) > 0 {
		dto["constraint_engine_rules"] = constraints
	}
	if req.DTOName != "" {
		dto["dto_name"] = req.DTOName
	}
	if len(req.Fields) > 0 {
		dto["fields"] = req.Fields
	}
	if len(req.RequestDTO) > 0 && json.Valid(req.RequestDTO) {
		var requestDTO any
		_ = json.Unmarshal(req.RequestDTO, &requestDTO)
		dto["request_dto"] = requestDTO
	}
	if len(req.ResponseSchema) > 0 && json.Valid(req.ResponseSchema) {
		var schema any
		_ = json.Unmarshal(req.ResponseSchema, &schema)
		dto["response_schema"] = schema
		dto["schema_requirement"] = "The response must conform to response_schema."
	}
	return json.MarshalIndent(dto, "", "  ")
}

func compactPromptJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buf.String()
}

func promptValueConstraints(description string) []string {
	rules := inferValueRules(description)
	var constraints []string
	if rules.PercentageMin != 0 || rules.PercentageMax != 0 {
		constraints = append(constraints, "percentage fields must be numeric values between "+strconv.Itoa(rules.PercentageMin)+" and "+strconv.Itoa(rules.PercentageMax))
	}
	if rules.PassportPrefix != "" {
		if rules.PassportDigits > 0 {
			constraints = append(constraints, "passport number fields must match "+rules.PassportPrefix+" followed by "+strconv.Itoa(rules.PassportDigits)+" digits")
		} else {
			constraints = append(constraints, "passport number fields must be strings starting with "+rules.PassportPrefix)
		}
	}
	return constraints
}

func ExtractJSON(text string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, errors.New("empty model response")
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed), nil
	}

	starts := []int{}
	for i, r := range trimmed {
		if r == '{' || r == '[' {
			starts = append(starts, i)
		}
	}
	for _, start := range starts {
		for end := len(trimmed); end > start; end-- {
			candidate := strings.TrimSpace(trimmed[start:end])
			if json.Valid([]byte(candidate)) {
				return json.RawMessage(candidate), nil
			}
		}
	}
	return nil, errors.New("model response did not contain valid JSON")
}
