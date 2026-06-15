package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseConstraintTextPatternAndRange(t *testing.T) {
	constraints := ParseConstraintText("passportNumber: pattern WS-xxxxxxx\nage: integer 18-70\nemail: email")
	if len(constraints) < 4 {
		t.Fatalf("expected several constraints, got %#v", constraints)
	}
	var foundPattern, foundRange, foundEmail bool
	for _, constraint := range constraints {
		if constraint.Field == "passportNumber" && constraint.Kind == "pattern" && constraint.Prefix == "WS-" && constraint.Digits == 7 {
			foundPattern = true
		}
		if constraint.Field == "age" && constraint.Kind == "range" && constraint.Min != nil && constraint.Max != nil && *constraint.Min == 18 && *constraint.Max == 70 {
			foundRange = true
		}
		if constraint.Field == "email" && constraint.Kind == "email" {
			foundEmail = true
		}
	}
	if !foundPattern || !foundRange || !foundEmail {
		t.Fatalf("missing expected constraints: %#v", constraints)
	}
}

func TestValidateMockQualityUsesManualConstraints(t *testing.T) {
	req := GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list 2 people",
		Constraints: ParseConstraintText("passportNumber: pattern WS-xxxxxxx\nage: integer 18-70"),
	}
	body := json.RawMessage(`{"items":[{"passportNumber":"P1000000","age":71},{"passportNumber":"WS-1000001","age":22}],"total":2}`)
	issues := ValidateMockQuality(body, req)
	messages := strings.Join(QualityIssueMessages(issues), "\n")
	if !strings.Contains(messages, "passportNumber") {
		t.Fatalf("expected passport constraint issue, got %s", messages)
	}
}

func TestTemplateProviderUsesManualConstraints(t *testing.T) {
	raw, err := BuildTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list 2 people",
		Fields: []FieldSpec{
			{Name: "passportNumber", Type: "string", Required: true},
			{Name: "age", Type: "integer", Required: true},
		},
		Constraints: ParseConstraintText("passportNumber: pattern WS-xxxxxxx\nage: integer 18-70"),
	})
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(body.Items))
	}
	passport, _ := body.Items[0]["passportNumber"].(string)
	age, _ := body.Items[0]["age"].(float64)
	if !strings.HasPrefix(passport, "WS-") || age < 18 || age > 70 {
		t.Fatalf("constraints were not applied: %#v", body.Items[0])
	}
}
