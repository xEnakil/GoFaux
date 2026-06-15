package assistant

import "testing"

func TestRequestedItemCountUsesContext(t *testing.T) {
	req := GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list of people, like maybe 25 amount, and percentage random from 50 to 100",
	}
	if got := requestedItemCount(req); got != 25 {
		t.Fatalf("expected count 25, got %d", got)
	}
}

func TestRequestedItemCountIgnoresPercentageRangeWithoutCountContext(t *testing.T) {
	req := GenerateRequest{
		Method:      "GET",
		Endpoint:    "/people",
		Description: "list people with percentage random from 50 to 100",
	}
	if got := requestedItemCount(req); got != 0 {
		t.Fatalf("expected no explicit count, got %d", got)
	}
}

func TestInferValueRules(t *testing.T) {
	rules := inferValueRules("percentage random from 50 to 100 and passport number all string starting with WS")
	if rules.PercentageMin != 50 || rules.PercentageMax != 100 {
		t.Fatalf("unexpected percentage range: %#v", rules)
	}
	if rules.PassportPrefix != "WS" {
		t.Fatalf("expected passport prefix WS, got %q", rules.PassportPrefix)
	}
}

func TestInferValueRulesPassportPattern(t *testing.T) {
	rules := inferValueRules("passport number with WS-xxxxxxx pattern")
	if rules.PassportPrefix != "WS-" || rules.PassportDigits != 7 {
		t.Fatalf("expected WS- pattern with 7 digits, got %#v", rules)
	}
}
