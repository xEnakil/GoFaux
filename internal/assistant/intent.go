package assistant

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func responseKind(req GenerateRequest) string {
	if isListRequest(req) {
		return "list"
	}
	return "object"
}

func isListRequest(req GenerateRequest) bool {
	return inferListIntent(req.Description, req.Endpoint, req.Method)
}

func looksLikeList(description, endpoint string) bool {
	return inferListIntent(description, endpoint, "")
}

func inferListIntent(description, endpoint, method string) bool {
	value := strings.ToLower(description + " " + endpoint)
	if strings.Contains(value, "list") ||
		strings.Contains(value, "array") ||
		strings.Contains(value, "collection") ||
		strings.Contains(value, "multiple") ||
		strings.Contains(value, "many ") ||
		strings.Contains(value, "all ") {
		return true
	}

	if strings.Contains(endpoint, "{") || strings.Contains(endpoint, ":") {
		return false
	}
	if method != "" && !strings.EqualFold(method, http.MethodGet) {
		return false
	}

	resource := endpointResource(endpoint)
	return resource == "people" || resource == "children" || (strings.HasSuffix(resource, "s") && resource != "status")
}

func requestedItemCount(req GenerateRequest) int {
	value := strings.ToLower(req.Description)
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "!", " ", "?", " ")
	words := strings.Fields(replacer.Replace(value))
	resourceWords := map[string]bool{
		"amount": true, "count": true, "item": true, "items": true, "record": true, "records": true,
		"object": true, "objects": true, "row": true, "rows": true, singularResource(endpointResource(req.Endpoint)): true,
		endpointResource(req.Endpoint): true,
	}
	for i, word := range words {
		if word == "from" || word == "to" {
			continue
		}
		if count, err := strconv.Atoi(word); err == nil && count > 0 {
			if hasCountContext(words, i, resourceWords) {
				return clampItemCount(count)
			}
		}
		if count, ok := numberWords[word]; ok {
			if hasCountContext(words, i, resourceWords) {
				return clampItemCount(count)
			}
		}
	}
	return 0
}

func hasCountContext(words []string, index int, resourceWords map[string]bool) bool {
	windowStart := index - 2
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := index + 2
	if windowEnd >= len(words) {
		windowEnd = len(words) - 1
	}
	for i := windowStart; i <= windowEnd; i++ {
		if i == index {
			continue
		}
		word := words[i]
		if word == "list" || word == "generate" || word == "return" || resourceWords[word] {
			return true
		}
	}
	return false
}

func defaultItemCount(req GenerateRequest) int {
	if count := requestedItemCount(req); count > 0 {
		return count
	}
	return 5
}

func clampItemCount(count int) int {
	if count < 1 {
		return 1
	}
	if count > 50 {
		return 50
	}
	return count
}

func endpointResource(endpoint string) string {
	trimmed := strings.Trim(strings.ToLower(endpoint), "/ ")
	if trimmed == "" {
		return "item"
	}

	parts := strings.Split(trimmed, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.Trim(parts[i], " {}:")
		if part == "" || part == "id" {
			continue
		}
		part = strings.ReplaceAll(part, "-", "_")
		return part
	}
	return "item"
}

type valueRules struct {
	PercentageMin  int
	PercentageMax  int
	PassportPrefix string
	PassportDigits int
}

func inferValueRules(description string) valueRules {
	lower := strings.ToLower(description)
	rules := valueRules{}
	if strings.Contains(lower, "percentage") || strings.Contains(lower, "percent") {
		rules.PercentageMin = 0
		rules.PercentageMax = 100
		if min, max, ok := inferNumericRange(lower); ok {
			rules.PercentageMin = min
			rules.PercentageMax = max
		}
	}
	if strings.Contains(lower, "passport") {
		rules.PassportPrefix, rules.PassportDigits = inferPatternPrefix(description)
		if rules.PassportPrefix == "" {
			rules.PassportPrefix = inferStringPrefix(description)
		}
	}
	return rules
}

func inferNumericRange(value string) (int, int, bool) {
	re := regexp.MustCompile(`(?:from\s+)?(\d+)\s*(?:to|-|and)\s*(\d+)`)
	match := re.FindStringSubmatch(value)
	if len(match) != 3 {
		return 0, 0, false
	}
	min, err1 := strconv.Atoi(match[1])
	max, err2 := strconv.Atoi(match[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if min > max {
		min, max = max, min
	}
	return min, max, true
}

func inferStringPrefix(description string) string {
	re := regexp.MustCompile(`(?i)(?:start(?:ing)?\s+with|prefix(?:ed)?\s+with)\s+([A-Za-z0-9_-]+)`)
	match := re.FindStringSubmatch(description)
	if len(match) == 2 {
		return strings.ToUpper(match[1])
	}
	return ""
}

func inferPatternPrefix(description string) (string, int) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b([A-Z]{1,8}[-_]?)(x{3,}|#{3,}|0{3,})\b(?:\s*(?:pattern|format))?`),
		regexp.MustCompile(`(?i)\b([A-Z]{1,8}[-_]?)(\d{3,})\b\s*(?:pattern|format)`),
	}
	for _, re := range patterns {
		match := re.FindStringSubmatch(description)
		if len(match) == 3 {
			prefix := strings.ToUpper(strings.ReplaceAll(match[1], " ", ""))
			return prefix, len(match[2])
		}
	}
	return "", 0
}

func singularResource(resource string) string {
	switch resource {
	case "people", "persons":
		return "person"
	case "children":
		return "child"
	case "categories":
		return "category"
	case "companies":
		return "company"
	}
	if strings.HasSuffix(resource, "ies") && len(resource) > 3 {
		return strings.TrimSuffix(resource, "ies") + "y"
	}
	if strings.HasSuffix(resource, "s") && len(resource) > 1 {
		return strings.TrimSuffix(resource, "s")
	}
	return resource
}

var numberWords = map[string]int{
	"one":       1,
	"two":       2,
	"three":     3,
	"four":      4,
	"five":      5,
	"six":       6,
	"seven":     7,
	"eight":     8,
	"nine":      9,
	"ten":       10,
	"eleven":    11,
	"twelve":    12,
	"thirteen":  13,
	"fourteen":  14,
	"fifteen":   15,
	"sixteen":   16,
	"seventeen": 17,
	"eighteen":  18,
	"nineteen":  19,
	"twenty":    20,
}
