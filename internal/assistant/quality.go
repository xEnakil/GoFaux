package assistant

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type QualityIssue struct {
	Message string
}

func LooksLikeInstructionEcho(body json.RawMessage) bool {
	if len(body) == 0 {
		return false
	}

	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return false
	}

	_, hasTask := object["task"]
	_, hasRules := object["rules"]
	_, hasPreferredShape := object["preferred_shape"]
	if hasTask && hasRules {
		return true
	}
	_, hasHTTPMethod := object["http_method"]
	_, hasUserIntent := object["user_intent"]
	_, hasResponseKind := object["response_kind"]
	if hasHTTPMethod && hasUserIntent && hasResponseKind {
		return true
	}

	task, _ := object["task"].(string)
	if strings.Contains(strings.ToLower(task), "generate_mock_response") {
		return true
	}

	return hasPreferredShape && hasRules
}

func ValidateMockQuality(body json.RawMessage, req GenerateRequest) []QualityIssue {
	var issues []QualityIssue
	instructionEcho := LooksLikeInstructionEcho(body)
	if instructionEcho {
		issues = append(issues, QualityIssue{Message: "response copied request details instead of returning only the mock body"})
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		issues = append(issues, QualityIssue{Message: "response is not valid JSON"})
		return issues
	}

	if object, ok := parsed.(map[string]any); ok {
		var forbidden []string
		for _, key := range forbiddenTopLevelResponseKeys() {
			if _, exists := object[key]; exists {
				forbidden = append(forbidden, key)
			}
		}
		if len(forbidden) > 0 {
			issues = append(issues, QualityIssue{Message: fmt.Sprintf("response included request metadata fields instead of only mock data: %s", strings.Join(forbidden, ", "))})
		}
	}

	if instructionEcho {
		return issues
	}

	if isListRequest(req) {
		items, source, ok := listItems(parsed)
		if !ok {
			issues = append(issues, QualityIssue{Message: "list request should return a JSON array or a JSON object with an items array"})
			return issues
		}
		if source != "items" && source != "top-level array" {
			issues = append(issues, QualityIssue{Message: fmt.Sprintf("list wrapper should use an items array, not %q", source)})
		}
		requestedCount := requestedItemCount(req)
		if requestedCount > 0 && len(items) != requestedCount {
			issues = append(issues, QualityIssue{Message: fmt.Sprintf("list should contain exactly %d items, got %d", requestedCount, len(items))})
		}
		if requestedCount == 0 && len(items) < 3 {
			issues = append(issues, QualityIssue{Message: fmt.Sprintf("list should contain multiple realistic items, got %d", len(items))})
		}
		if source == "items" {
			issues = append(issues, validateTotalCount(parsed, len(items))...)
		}
		issues = append(issues, validateItemValueRules(items, req)...)
		issues = append(issues, validateConstraintSpecs(items, req)...)
	} else if object, ok := parsed.(map[string]any); ok {
		issues = append(issues, validateConstraintSpecs([]map[string]any{object}, req)...)
	}

	return issues
}

func QualityIssueMessages(issues []QualityIssue) []string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(issue.Message) != "" {
			messages = append(messages, issue.Message)
		}
	}
	return messages
}

func forbiddenTopLevelResponseKeys() []string {
	return []string{
		"http_method",
		"endpoint",
		"status_code",
		"user_intent",
		"response_kind",
		"list_guidance",
		"item_count_guidance",
		"requested_item_count",
		"exact_item_count",
		"forbidden_response_fields",
		"quality_rules",
		"response_contract",
		"value_constraints",
		"schema_requirement",
	}
}

func listItems(parsed any) ([]map[string]any, string, bool) {
	switch value := parsed.(type) {
	case []any:
		return objectsFromArray(value), "top-level array", true
	case map[string]any:
		if rawItems, ok := value["items"].([]any); ok {
			return objectsFromArray(rawItems), "items", true
		}
		for key, candidate := range value {
			if rawItems, ok := candidate.([]any); ok {
				return objectsFromArray(rawItems), key, true
			}
		}
	}
	return nil, "", false
}

func objectsFromArray(values []any) []map[string]any {
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			items = append(items, item)
		}
	}
	return items
}

func validateTotalCount(parsed any, itemCount int) []QualityIssue {
	object, ok := parsed.(map[string]any)
	if !ok {
		return nil
	}
	total, exists := object["total"]
	if !exists {
		return []QualityIssue{{Message: "list response object should include total equal to the item count"}}
	}
	totalNumber, ok := numericValue(total)
	if !ok || int(math.Round(totalNumber)) != itemCount {
		return []QualityIssue{{Message: fmt.Sprintf("total should equal item count %d", itemCount)}}
	}
	return nil
}

func validateItemValueRules(items []map[string]any, req GenerateRequest) []QualityIssue {
	rules := inferValueRules(req.Description)
	var issues []QualityIssue

	if strings.Contains(strings.ToLower(req.Description), "percentage") || strings.Contains(strings.ToLower(req.Description), "percent") {
		for i, item := range items {
			value, ok := firstFieldContaining(item, "percentage", "percent")
			if !ok {
				issues = append(issues, QualityIssue{Message: "items should include a percentage field"})
				break
			}
			number, ok := numericValue(value)
			if !ok || int(number) < rules.PercentageMin || int(number) > rules.PercentageMax {
				issues = append(issues, QualityIssue{Message: fmt.Sprintf("item %d percentage should be between %d and %d", i+1, rules.PercentageMin, rules.PercentageMax)})
				break
			}
		}
	}

	if rules.PassportPrefix != "" {
		for i, item := range items {
			value, ok := firstFieldContaining(item, "passport")
			if !ok {
				issues = append(issues, QualityIssue{Message: "items should include a passport number field"})
				break
			}
			text, ok := value.(string)
			if !ok || !strings.HasPrefix(strings.ToUpper(text), rules.PassportPrefix) {
				issues = append(issues, QualityIssue{Message: fmt.Sprintf("item %d passport number should start with %s", i+1, rules.PassportPrefix)})
				break
			}
			if rules.PassportDigits > 0 && !hasDigitSuffix(text, rules.PassportPrefix, rules.PassportDigits) {
				issues = append(issues, QualityIssue{Message: fmt.Sprintf("item %d passport number should match %s plus %d digits", i+1, rules.PassportPrefix, rules.PassportDigits)})
				break
			}
		}
	}

	return issues
}

func hasDigitSuffix(value, prefix string, digits int) bool {
	if digits <= 0 {
		return true
	}
	if !strings.HasPrefix(strings.ToUpper(value), strings.ToUpper(prefix)) {
		return false
	}
	suffix := value[len(prefix):]
	if len(suffix) != digits {
		return false
	}
	for _, char := range suffix {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func validateConstraintSpecs(items []map[string]any, req GenerateRequest) []QualityIssue {
	constraints := InferConstraints(req)
	if len(constraints) == 0 {
		return nil
	}
	var issues []QualityIssue
	for itemIndex, item := range items {
		for _, constraint := range constraints {
			if constraint.Kind == "" || constraint.Field == "" {
				continue
			}
			value, ok := fieldForConstraint(item, constraint)
			if !ok {
				if constraint.Required || constraint.Kind == "required" {
					issues = append(issues, QualityIssue{Message: fmt.Sprintf("item %d should include required field %s", itemIndex+1, constraint.Field)})
					return issues
				}
				continue
			}
			if issue, failed := validateConstraintValue(itemIndex, value, constraint); failed {
				issues = append(issues, issue)
				return issues
			}
		}
	}
	return issues
}

func fieldForConstraint(item map[string]any, constraint ConstraintSpec) (any, bool) {
	for key, value := range item {
		if constraintMatchesField(constraint, key) {
			return value, true
		}
	}
	return nil, false
}

func validateConstraintValue(itemIndex int, value any, constraint ConstraintSpec) (QualityIssue, bool) {
	field := constraint.Field
	switch constraint.Kind {
	case "required":
		if value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return QualityIssue{Message: fmt.Sprintf("item %d required field %s is empty", itemIndex+1, field)}, true
		}
	case "type":
		if !valueMatchesType(value, constraint.Type) {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be %s", itemIndex+1, field, constraint.Type)}, true
		}
	case "email":
		text, ok := value.(string)
		if !ok || !strings.Contains(text, "@") || !strings.Contains(text, ".") {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be a valid email", itemIndex+1, field)}, true
		}
	case "enum":
		text := fmt.Sprint(value)
		for _, allowed := range constraint.Values {
			if text == allowed {
				return QualityIssue{}, false
			}
		}
		return QualityIssue{Message: fmt.Sprintf("item %d field %s should be one of %s", itemIndex+1, field, strings.Join(constraint.Values, ", "))}, true
	case "range":
		number, ok := numericValue(value)
		if !ok {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be numeric", itemIndex+1, field)}, true
		}
		if constraint.Min != nil && number < *constraint.Min {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be at least %s", itemIndex+1, field, formatFloat(*constraint.Min))}, true
		}
		if constraint.Max != nil && number > *constraint.Max {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be at most %s", itemIndex+1, field, formatFloat(*constraint.Max))}, true
		}
	case "pattern":
		text, ok := value.(string)
		if !ok {
			return QualityIssue{Message: fmt.Sprintf("item %d field %s should be a string matching %s", itemIndex+1, field, constraint.Pattern)}, true
		}
		if constraint.Prefix != "" && constraint.Digits > 0 {
			if !hasDigitSuffix(text, constraint.Prefix, constraint.Digits) {
				return QualityIssue{Message: fmt.Sprintf("item %d field %s should match %s plus %d digits", itemIndex+1, field, constraint.Prefix, constraint.Digits)}, true
			}
			return QualityIssue{}, false
		}
		if constraint.Pattern != "" {
			re, err := regexp.Compile(constraint.Pattern)
			if err == nil && !re.MatchString(text) {
				return QualityIssue{Message: fmt.Sprintf("item %d field %s should match %s", itemIndex+1, field, constraint.Pattern)}, true
			}
		}
	}
	return QualityIssue{}, false
}

func valueMatchesType(value any, typ string) bool {
	switch normalizeConstraintType(typ) {
	case "integer":
		number, ok := numericValue(value)
		return ok && math.Round(number) == number
	case "number":
		_, ok := numericValue(value)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return true
	}
}

func firstFieldContaining(item map[string]any, tokens ...string) (any, bool) {
	for key, value := range item {
		lower := strings.ToLower(key)
		for _, token := range tokens {
			if strings.Contains(lower, token) {
				return value, true
			}
		}
	}
	return nil, false
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case string:
		cleaned := strings.TrimSpace(strings.TrimSuffix(typed, "%"))
		number, err := strconv.ParseFloat(cleaned, 64)
		return number, err == nil
	default:
		return 0, false
	}
}
