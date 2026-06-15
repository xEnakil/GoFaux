package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLooksLikeInstructionEcho(t *testing.T) {
	raw := json.RawMessage(`{"task":"generate_mock_response_json","rules":["return json"],"endpoint":"/people"}`)
	if !LooksLikeInstructionEcho(raw) {
		t.Fatal("expected instruction echo to be detected")
	}
}

func TestLooksLikeInstructionEchoDetectsRequestDetails(t *testing.T) {
	raw := json.RawMessage(`{"http_method":"GET","endpoint":"/people","user_intent":"list people","response_kind":"list"}`)
	if !LooksLikeInstructionEcho(raw) {
		t.Fatal("expected request details echo to be detected")
	}
}

func TestLooksLikeInstructionEchoIgnoresNormalPayload(t *testing.T) {
	raw := json.RawMessage(`{"items":[{"id":1,"name":"Ava Johnson","age":28,"gender":"female","hairColor":"brown"}],"total":1}`)
	if LooksLikeInstructionEcho(raw) {
		t.Fatal("did not expect normal mock payload to be detected as an instruction echo")
	}
}

func TestValidateMockQualityAcceptsTopLevelListArray(t *testing.T) {
	raw := json.RawMessage(`[
		{"id":1,"name":"Ava Johnson","age":28},
		{"id":2,"name":"Noah Smith","age":31}
	]`)
	issues := ValidateMockQuality(raw, GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list 2 people with name and age",
	})
	if len(issues) != 0 {
		t.Fatalf("expected top-level array list to be accepted, got: %v", QualityIssueMessages(issues))
	}
}

func TestValidateMockQualityRejectsSingleItemForOpenList(t *testing.T) {
	raw := json.RawMessage(`{"items":[{"id":1,"name":"Ava Johnson"}],"total":1}`)
	issues := ValidateMockQuality(raw, GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list people with name",
	})
	if len(issues) == 0 {
		t.Fatal("expected single-item open list to be rejected")
	}
	if !strings.Contains(strings.Join(QualityIssueMessages(issues), "\n"), "multiple realistic items") {
		t.Fatalf("expected multiple item issue, got: %v", QualityIssueMessages(issues))
	}
}

func TestValidateMockQualityCatchesAlmostRightPeopleList(t *testing.T) {
	raw := json.RawMessage(`{
		"dtos": [
			{"id":"1","name":"John Doe","nationality":"American","passport":"1234567890","percentage":"50%"},
			{"id":"2","name":"Jane Smith","nationality":"British","passport":"1234567890","percentage":"40%"}
		],
		"status_code": 200,
		"total_count": 2,
		"user_intent": "list of people, like maybe 25 amount, and their nationality and percentage random from 50 to 100 and passport number all string starting with WS"
	}`)
	issues := ValidateMockQuality(raw, GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list of people, like maybe 25 amount, and their nationality and percentage random from 50 to 100 and passport number all string starting with WS",
	})
	if len(issues) == 0 {
		t.Fatal("expected quality issues")
	}
	messages := strings.Join(QualityIssueMessages(issues), "\n")
	for _, want := range []string{"status_code", "user_intent", "items array", "exactly 25", "percentage", "passport"} {
		if !strings.Contains(messages, want) {
			t.Fatalf("expected issue containing %q, got:\n%s", want, messages)
		}
	}
}

func TestBuildPeopleListJSON(t *testing.T) {
	raw, err := BuildPeopleListJSON(10)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Items []struct {
			Name      string `json:"name"`
			Age       int    `json:"age"`
			Gender    string `json:"gender"`
			HairColor string `json:"hairColor"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 10 || body.Total != 10 {
		t.Fatalf("expected 10 people, got %d total %d", len(body.Items), body.Total)
	}
	if body.Items[0].Name == "" || body.Items[0].Age == 0 || body.Items[0].HairColor == "" {
		t.Fatalf("unexpected first item: %#v", body.Items[0])
	}
}
