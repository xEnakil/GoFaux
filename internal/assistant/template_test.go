package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTemplateProviderUsesFields(t *testing.T) {
	provider := TemplateProvider{}
	result, err := provider.GenerateMock(context.Background(), GenerateRequest{
		Method:      "GET",
		Endpoint:    "/users/{id}",
		Status:      200,
		Description: "User DTO",
		Fields: []FieldSpec{
			{Name: "id", Type: "int", Required: true},
			{Name: "email", Type: "string", Required: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Body) == 0 {
		t.Fatal("expected generated body")
	}
	if result.Provider != "template" {
		t.Fatalf("expected template provider, got %q", result.Provider)
	}
}

func TestExtractJSONFromModelText(t *testing.T) {
	raw := "Here is the response:\n```json\n{\"id\":1,\"name\":\"Example\"}\n```"
	body, err := ExtractJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"id":1,"name":"Example"}` {
		t.Fatalf("unexpected JSON: %s", string(body))
	}
}

func TestTemplateProviderBuildsGenericListFromIntent(t *testing.T) {
	raw, err := BuildTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/products",
		Status:      200,
		Description: "list three products with price and status",
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 3 || len(body.Items) != 3 {
		t.Fatalf("expected three product items, got total=%d len=%d", body.Total, len(body.Items))
	}
	if _, ok := body.Items[0]["price"]; !ok {
		t.Fatalf("expected inferred price field: %#v", body.Items[0])
	}
	if _, ok := body.Items[0]["status"]; !ok {
		t.Fatalf("expected inferred status field: %#v", body.Items[0])
	}
}

func TestTemplateProviderKeepsPathParamEndpointSingular(t *testing.T) {
	raw, err := BuildTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/users/{id}",
		Status:      200,
		Description: "user with name and age",
	})
	if err != nil {
		t.Fatal(err)
	}

	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	if _, ok := object["items"]; ok {
		t.Fatalf("did not expect list response for path-param endpoint: %#v", object)
	}
	if _, ok := object["age"]; !ok {
		t.Fatalf("expected inferred age field: %#v", object)
	}
}

func TestTemplateProviderHonorsPeopleListConstraints(t *testing.T) {
	raw, err := BuildTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Status:      200,
		Description: "list of people, like maybe 25 amount, and their nationality and percentage random from 50 to 100 and passport number all string starting with WS",
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 25 || len(body.Items) != 25 {
		t.Fatalf("expected 25 items, got total=%d len=%d", body.Total, len(body.Items))
	}
	first := body.Items[0]
	if first["nationality"] == "" {
		t.Fatalf("expected nationality: %#v", first)
	}
	if value, ok := first["passportNumber"].(string); !ok || !strings.HasPrefix(value, "WS") {
		t.Fatalf("expected WS passport number, got %#v", first["passportNumber"])
	}
	if value, ok := first["percentage"].(float64); !ok || value < 50 || value > 100 {
		t.Fatalf("expected percentage between 50 and 100, got %#v", first["percentage"])
	}
}

func TestTemplateProviderHonorsPassportPattern(t *testing.T) {
	raw, err := BuildTemplateJSON(GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Status:      200,
		Description: "list people with name, age and nationality, hair color, and passport number with WS-xxxxxxx pattern",
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
	if len(body.Items) == 0 {
		t.Fatal("expected list items")
	}
	value, ok := body.Items[0]["passportNumber"].(string)
	if !ok || !strings.HasPrefix(value, "WS-") || len(strings.TrimPrefix(value, "WS-")) != 7 {
		t.Fatalf("expected WS-xxxxxxx passport number, got %#v", body.Items[0]["passportNumber"])
	}
}
