package generator

import "testing"

func TestInferMockRequestSkipsFillerWords(t *testing.T) {
	method, endpoint, _ := InferMockRequest("list of people with age")
	if method != "GET" || endpoint != "/people" {
		t.Fatalf("unexpected inference: %s %s", method, endpoint)
	}
}

func TestParseFieldsNaturalTextIgnored(t *testing.T) {
	fields := ParseFields("all strings except age integer")
	if len(fields) != 0 {
		t.Fatalf("expected natural text not to become field names: %#v", fields)
	}
}

func TestParseFieldsStructured(t *testing.T) {
	fields := ParseFields("id:int,name:string,age:int")
	if len(fields) != 3 {
		t.Fatalf("expected three fields, got %#v", fields)
	}
	if fields[0].Name != "id" || fields[0].Type != "int" {
		t.Fatalf("unexpected first field: %#v", fields[0])
	}
}
