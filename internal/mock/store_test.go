package mock

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestMatchPathWithParameter(t *testing.T) {
	params, ok := MatchPath("/users/{id}/orders/{orderId}", "/users/42/orders/abc")
	if !ok {
		t.Fatal("expected path to match")
	}
	if params["id"] != "42" {
		t.Fatalf("expected id path param to be 42, got %q", params["id"])
	}
	if params["orderId"] != "abc" {
		t.Fatalf("expected orderId path param to be abc, got %q", params["orderId"])
	}
}

func TestDefinitionMatchesRequestWithQueryAndHeaders(t *testing.T) {
	body, err := ValidateJSON(`{"id":1}`)
	if err != nil {
		t.Fatal(err)
	}
	def, err := NewDefinition(Definition{
		Method:   http.MethodGet,
		Endpoint: "/users/{id}",
		Query: map[string]string{
			"active": "true",
		},
		RequestHeaders: map[string]string{
			"X-Tenant": "demo",
		},
		Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/users/42?active=true", nil)
	req.Header.Set("X-Tenant", "demo")
	match, ok := def.Matches(req)
	if !ok {
		t.Fatal("expected request to match")
	}
	if match.PathParams["id"] != "42" {
		t.Fatalf("expected path parameter id=42, got %q", match.PathParams["id"])
	}
}

func TestStorePersistsMocks(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mocks.json")
	store, err := NewStore(configPath)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ValidateJSON(`{"message":"ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Definition{
		Method:   http.MethodGet,
		Endpoint: "/health",
		Body:     body,
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStore(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Count() != 1 {
		t.Fatalf("expected 1 persisted mock, got %d", reloaded.Count())
	}
}
