package assistant

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ParseConstraintText(text string) []ConstraintSpec {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var constraints []ConstraintSpec
	for _, line := range splitConstraintLines(text) {
		field, rule, ok := strings.Cut(line, ":")
		if !ok {
			field, rule = "", line
		}
		field = strings.TrimSpace(field)
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		constraints = append(constraints, parseConstraintRule(field, rule, "manual")...)
	}
	return dedupeConstraints(constraints)
}

func InferConstraints(req GenerateRequest) []ConstraintSpec {
	var constraints []ConstraintSpec
	constraints = append(constraints, req.Constraints...)
	for _, field := range req.Fields {
		if strings.TrimSpace(field.Name) == "" {
			continue
		}
		if field.Required {
			constraints = append(constraints, ConstraintSpec{Field: field.Name, Kind: "required", Required: true, Source: "field"})
		}
		if strings.TrimSpace(field.Type) != "" {
			constraints = append(constraints, ConstraintSpec{Field: field.Name, Kind: "type", Type: normalizeConstraintType(field.Type), Source: "field"})
		}
	}
	constraints = append(constraints, inferDescriptionConstraints(req)...)
	constraints = append(constraints, ConstraintsFromSchema(req.ResponseSchema)...)
	return dedupeConstraints(constraints)
}

func ConstraintSummaries(req GenerateRequest) []string {
	constraints := InferConstraints(req)
	out := make([]string, 0, len(constraints))
	for _, constraint := range constraints {
		if summary := constraintSummary(constraint); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

func ConstraintsFromSchema(raw json.RawMessage) []ConstraintSpec {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var schema any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil
	}
	return constraintsFromSchemaValue("", schema, map[string]bool{})
}

func splitConstraintLines(text string) []string {
	normalized := strings.NewReplacer("\r\n", "\n", "\r", "\n", ";", "\n").Replace(text)
	var lines []string
	for _, line := range strings.Split(normalized, "\n") {
		line = strings.TrimSpace(strings.Trim(line, "-* "))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func parseConstraintRule(field, rule, source string) []ConstraintSpec {
	lower := strings.ToLower(rule)
	var out []ConstraintSpec
	if strings.Contains(lower, "required") {
		out = append(out, ConstraintSpec{Field: field, Kind: "required", Required: true, Source: source})
	}
	if strings.Contains(lower, "email") {
		out = append(out, ConstraintSpec{Field: field, Kind: "email", Source: source})
	}
	if values := parseEnumValues(rule); len(values) > 0 {
		out = append(out, ConstraintSpec{Field: field, Kind: "enum", Values: values, Source: source})
	}
	if min, max, ok := inferFloatRange(lower); ok {
		out = append(out, ConstraintSpec{Field: field, Kind: "range", Min: &min, Max: &max, Source: source})
	}
	if pattern := parsePattern(rule); pattern != "" {
		spec := ConstraintSpec{Field: field, Kind: "pattern", Pattern: pattern, Source: source}
		if prefix, digits := inferPatternPrefix(pattern); prefix != "" {
			spec.Prefix = prefix
			spec.Digits = digits
		}
		out = append(out, spec)
	}
	for _, typ := range []string{"integer", "int", "number", "string", "boolean", "bool", "array", "object"} {
		if wordPresent(lower, typ) {
			out = append(out, ConstraintSpec{Field: field, Kind: "type", Type: normalizeConstraintType(typ), Source: source})
			break
		}
	}
	return out
}

func inferDescriptionConstraints(req GenerateRequest) []ConstraintSpec {
	var constraints []ConstraintSpec
	rules := inferValueRules(req.Description)
	if strings.Contains(strings.ToLower(req.Description), "percentage") || strings.Contains(strings.ToLower(req.Description), "percent") {
		min, max := float64(rules.PercentageMin), float64(rules.PercentageMax)
		constraints = append(constraints, ConstraintSpec{Field: "percentage", Kind: "range", Min: &min, Max: &max, Source: "description"})
	}
	if rules.PassportPrefix != "" {
		pattern := rules.PassportPrefix
		if rules.PassportDigits > 0 {
			pattern += strings.Repeat("x", rules.PassportDigits)
		}
		constraints = append(constraints, ConstraintSpec{
			Field:   "passport",
			Kind:    "pattern",
			Pattern: pattern,
			Prefix:  rules.PassportPrefix,
			Digits:  rules.PassportDigits,
			Source:  "description",
		})
	}
	return constraints
}

func constraintsFromSchemaValue(field string, value any, seen map[string]bool) []ConstraintSpec {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if ref, _ := object["$ref"].(string); ref != "" {
		if seen[ref] {
			return nil
		}
		seen[ref] = true
	}
	if object["type"] == "array" {
		return constraintsFromSchemaValue(field, object["items"], seen)
	}

	var constraints []ConstraintSpec
	if properties, ok := object["properties"].(map[string]any); ok {
		if itemSchema, ok := schemaArrayItemsProperty(properties, "items"); ok {
			return constraintsFromSchemaValue("", itemSchema, seen)
		}
		required := stringSetFromArray(object["required"])
		for name, prop := range properties {
			constraints = append(constraints, constraintsFromSchemaProperty(name, prop, required[name], seen)...)
		}
		return constraints
	}
	if field != "" {
		constraints = append(constraints, constraintsFromSchemaProperty(field, object, false, seen)...)
	}
	return constraints
}

func schemaArrayItemsProperty(properties map[string]any, key string) (any, bool) {
	prop, ok := properties[key].(map[string]any)
	if !ok {
		return nil, false
	}
	if typ, _ := prop["type"].(string); typ != "array" {
		return nil, false
	}
	return prop["items"], prop["items"] != nil
}

func constraintsFromSchemaProperty(field string, prop any, required bool, seen map[string]bool) []ConstraintSpec {
	object, ok := prop.(map[string]any)
	if !ok {
		return nil
	}
	if object["type"] == "array" {
		return constraintsFromSchemaValue(field, object["items"], seen)
	}
	var constraints []ConstraintSpec
	if required {
		constraints = append(constraints, ConstraintSpec{Field: field, Kind: "required", Required: true, Source: "schema"})
	}
	if typ, _ := object["type"].(string); typ != "" {
		constraints = append(constraints, ConstraintSpec{Field: field, Kind: "type", Type: normalizeConstraintType(typ), Source: "schema"})
	}
	if pattern, _ := object["pattern"].(string); pattern != "" {
		spec := ConstraintSpec{Field: field, Kind: "pattern", Pattern: pattern, Source: "schema"}
		if prefix, digits := inferPatternPrefix(pattern); prefix != "" {
			spec.Prefix = prefix
			spec.Digits = digits
		}
		constraints = append(constraints, spec)
	}
	min, hasMin := numberFromSchema(object["minimum"])
	max, hasMax := numberFromSchema(object["maximum"])
	if hasMin || hasMax {
		constraints = append(constraints, ConstraintSpec{Field: field, Kind: "range", Min: optionalFloat(hasMin, min), Max: optionalFloat(hasMax, max), Source: "schema"})
	}
	if values := stringValuesFromArray(object["enum"]); len(values) > 0 {
		constraints = append(constraints, ConstraintSpec{Field: field, Kind: "enum", Values: values, Source: "schema"})
	}
	if format, _ := object["format"].(string); strings.EqualFold(format, "email") {
		constraints = append(constraints, ConstraintSpec{Field: field, Kind: "email", Source: "schema"})
	}
	return constraints
}

func constraintSummary(c ConstraintSpec) string {
	field := c.Field
	if field == "" {
		field = "response"
	}
	switch c.Kind {
	case "required":
		return field + " is required"
	case "type":
		return field + " must be " + c.Type
	case "email":
		return field + " must be a valid email string"
	case "enum":
		return field + " must be one of " + strings.Join(c.Values, ", ")
	case "range":
		switch {
		case c.Min != nil && c.Max != nil:
			return fmt.Sprintf("%s must be between %s and %s", field, formatFloat(*c.Min), formatFloat(*c.Max))
		case c.Min != nil:
			return fmt.Sprintf("%s must be at least %s", field, formatFloat(*c.Min))
		case c.Max != nil:
			return fmt.Sprintf("%s must be at most %s", field, formatFloat(*c.Max))
		}
	case "pattern":
		if c.Prefix != "" && c.Digits > 0 {
			return fmt.Sprintf("%s must match %s plus %d digits", field, c.Prefix, c.Digits)
		}
		return field + " must match pattern " + c.Pattern
	}
	return c.Description
}

func constraintMatchesField(c ConstraintSpec, field string) bool {
	candidate := strings.ToLower(strings.TrimSpace(field))
	target := strings.ToLower(strings.TrimSpace(c.Field))
	if target == "" || candidate == "" {
		return false
	}
	if candidate == target {
		return true
	}
	return strings.Contains(candidate, target) || strings.Contains(target, candidate)
}

func dedupeConstraints(input []ConstraintSpec) []ConstraintSpec {
	var out []ConstraintSpec
	seen := map[string]bool{}
	for _, c := range input {
		c.Field = strings.TrimSpace(c.Field)
		c.Kind = strings.ToLower(strings.TrimSpace(c.Kind))
		c.Type = normalizeConstraintType(c.Type)
		if c.Kind == "" {
			continue
		}
		keyBytes, _ := json.Marshal(c)
		key := string(keyBytes)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func parseEnumValues(rule string) []string {
	lower := strings.ToLower(rule)
	index := strings.Index(lower, "enum")
	if index < 0 {
		index = strings.Index(lower, "one of")
	}
	if index < 0 {
		return nil
	}
	valuesText := rule[index:]
	valuesText = strings.TrimPrefix(strings.TrimSpace(valuesText), "enum")
	valuesText = strings.TrimPrefix(strings.TrimSpace(valuesText), "one of")
	valuesText = strings.Trim(valuesText, " :[]")
	var values []string
	for _, part := range strings.Split(valuesText, ",") {
		value := strings.Trim(strings.TrimSpace(part), `"'`)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func parsePattern(rule string) string {
	re := regexp.MustCompile(`(?i)(?:pattern|format|matches?)\s+([A-Za-z0-9_\-#xX.*+?^${}()[\]\\|]+)`)
	if match := re.FindStringSubmatch(rule); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	if prefix, digits := inferPatternPrefix(rule); prefix != "" {
		return prefix + strings.Repeat("x", digits)
	}
	return ""
}

func inferFloatRange(value string) (float64, float64, bool) {
	re := regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*(?:to|\.\.|-|and)\s*(-?\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(value)
	if len(match) != 3 {
		return 0, 0, false
	}
	min, err1 := strconv.ParseFloat(match[1], 64)
	max, err2 := strconv.ParseFloat(match[2], 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if min > max {
		min, max = max, min
	}
	return min, max, true
}

func wordPresent(value, word string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(value)
}

func normalizeConstraintType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "int", "integer":
		return "integer"
	case "float", "decimal", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "array", "list":
		return "array"
	case "object":
		return "object"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func numberFromSchema(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		n, err := typed.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func optionalFloat(ok bool, value float64) *float64 {
	if !ok {
		return nil
	}
	return &value
}

func stringSetFromArray(value any) map[string]bool {
	set := map[string]bool{}
	for _, item := range stringValuesFromArray(value) {
		set[item] = true
	}
	return set
}

func stringValuesFromArray(value any) []string {
	array, ok := value.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range array {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func formatFloat(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
