package openapi

import "testing"

func TestParseOpenAPIYAMLPreview(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: People API
  version: 1.0.0
paths:
  /people:
    get:
      operationId: listPeople
      summary: List people
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [items, total]
                properties:
                  items:
                    type: array
                    items:
                      $ref: "#/components/schemas/Person"
                  total:
                    type: integer
components:
  schemas:
    Person:
      type: object
      required: [name, passportNumber]
      properties:
        name:
          type: string
        age:
          type: integer
          minimum: 18
          maximum: 70
        passportNumber:
          type: string
          pattern: WS-xxxxxxx
`
	preview, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Title != "People API" || len(preview.Operations) != 1 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	op := preview.Operations[0]
	if op.Method != "GET" || op.Path != "/people" || op.Status != 200 {
		t.Fatalf("unexpected operation: %#v", op)
	}
	if len(op.Fields) == 0 {
		t.Fatalf("expected fields from schema: %#v", op)
	}
	if len(op.ResponseSchema) == 0 || len(op.Constraints) == 0 {
		t.Fatalf("expected schema and constraints: %#v", op)
	}
}
