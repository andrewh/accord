// Tests for schema-to-example value generation.
package openapi

import (
	"reflect"
	"testing"
)

func TestExampleValueString(t *testing.T) {
	s := &SchemaRef{Type: "string"}
	got := ExampleValue(s)
	if got != "string" {
		t.Errorf("expected \"string\", got %v", got)
	}
}

func TestExampleValueInteger(t *testing.T) {
	s := &SchemaRef{Type: "integer"}
	got := ExampleValue(s)
	if got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestExampleValueNumber(t *testing.T) {
	s := &SchemaRef{Type: "number"}
	got := ExampleValue(s)
	if got != 0.0 {
		t.Errorf("expected 0.0, got %v", got)
	}
}

func TestExampleValueBoolean(t *testing.T) {
	s := &SchemaRef{Type: "boolean"}
	got := ExampleValue(s)
	if got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestExampleValueWithExplicitExample(t *testing.T) {
	s := &SchemaRef{Type: "string", Example: "custom-value"}
	got := ExampleValue(s)
	if got != "custom-value" {
		t.Errorf("expected \"custom-value\", got %v", got)
	}
}

func TestExampleValueSingleEnum(t *testing.T) {
	s := &SchemaRef{Type: "string", Enum: []any{"active"}}
	got := ExampleValue(s)
	if got != "active" {
		t.Errorf("expected \"active\", got %v", got)
	}
}

func TestExampleValueMultiEnum(t *testing.T) {
	s := &SchemaRef{Type: "string", Enum: []any{"available", "pending", "sold"}}
	got := ExampleValue(s)
	if got != "available" {
		t.Errorf("expected \"available\", got %v", got)
	}
}

func TestExampleValueFormatDateTime(t *testing.T) {
	s := &SchemaRef{Type: "string", Format: "date-time"}
	got := ExampleValue(s)
	if got != "2024-01-01T00:00:00Z" {
		t.Errorf("expected date-time default, got %v", got)
	}
}

func TestExampleValueFormatDate(t *testing.T) {
	s := &SchemaRef{Type: "string", Format: "date"}
	got := ExampleValue(s)
	if got != "2024-01-01" {
		t.Errorf("expected date default, got %v", got)
	}
}

func TestExampleValueFormatEmail(t *testing.T) {
	s := &SchemaRef{Type: "string", Format: "email"}
	got := ExampleValue(s)
	if got != "user@example.com" {
		t.Errorf("expected email default, got %v", got)
	}
}

func TestExampleValueFormatUUID(t *testing.T) {
	s := &SchemaRef{Type: "string", Format: "uuid"}
	got := ExampleValue(s)
	if got != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected uuid default, got %v", got)
	}
}

func TestExampleValueFormatURI(t *testing.T) {
	s := &SchemaRef{Type: "string", Format: "uri"}
	got := ExampleValue(s)
	if got != "https://example.com" {
		t.Errorf("expected uri default, got %v", got)
	}
}

func TestExampleValueArray(t *testing.T) {
	s := &SchemaRef{
		Type:  "array",
		Items: &SchemaRef{Type: "string"},
	}
	got := ExampleValue(s)
	expected := []any{"string"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestExampleValueObject(t *testing.T) {
	s := &SchemaRef{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*SchemaRef{
			"name":     {Type: "string"},
			"optional": {Type: "string"},
		},
	}
	got := ExampleValue(s)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["name"] != "string" {
		t.Errorf("expected name to be \"string\", got %v", m["name"])
	}
	if _, hasOptional := m["optional"]; hasOptional {
		t.Error("expected optional field to be excluded")
	}
}

func TestExampleValueAllOf(t *testing.T) {
	s := &SchemaRef{
		AllOf: []*SchemaRef{
			{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*SchemaRef{
					"id": {Type: "integer"},
				},
			},
			{
				Type:     "object",
				Required: []string{"active"},
				Properties: map[string]*SchemaRef{
					"active": {Type: "boolean"},
				},
			},
		},
	}
	got := ExampleValue(s)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["id"] != 0 {
		t.Errorf("expected id to be 0, got %v", m["id"])
	}
	if m["active"] != false {
		t.Errorf("expected active to be false, got %v", m["active"])
	}
}

func TestExampleValueOneOf(t *testing.T) {
	s := &SchemaRef{
		OneOf: []*SchemaRef{
			{
				Type:     "object",
				Required: []string{"kind"},
				Properties: map[string]*SchemaRef{
					"kind": {Type: "string"},
				},
			},
			{
				Type:     "object",
				Required: []string{"other"},
				Properties: map[string]*SchemaRef{
					"other": {Type: "integer"},
				},
			},
		},
	}
	got := ExampleValue(s)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	// Should pick first variant only.
	if _, hasKind := m["kind"]; !hasKind {
		t.Error("expected 'kind' from first oneOf variant")
	}
	if _, hasOther := m["other"]; hasOther {
		t.Error("expected 'other' from second oneOf variant to be absent")
	}
}

func TestExampleValueDepthLimit(t *testing.T) {
	// Create a self-referencing schema (simulated via deep nesting).
	s := &SchemaRef{
		Type:     "object",
		Required: []string{"child"},
		Properties: map[string]*SchemaRef{
			"child": nil, // will be set below
		},
	}
	// Build 15 levels of nesting (exceeds depth limit of 10).
	current := s
	for i := 0; i < 15; i++ {
		next := &SchemaRef{
			Type:     "object",
			Required: []string{"child"},
			Properties: map[string]*SchemaRef{
				"child": nil,
			},
		}
		current.Properties["child"] = next
		current = next
	}
	current.Properties["child"] = &SchemaRef{Type: "string"}

	// Should not panic, should return something.
	got := ExampleValue(s)
	if got == nil {
		t.Error("expected non-nil result even with deep nesting")
	}
}

func TestExampleValueNilSchema(t *testing.T) {
	got := ExampleValue(nil)
	if got != nil {
		t.Errorf("expected nil for nil schema, got %v", got)
	}
}

func TestExampleValueFormatInt64(t *testing.T) {
	s := &SchemaRef{Type: "integer", Format: "int64"}
	got := ExampleValue(s)
	if got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestExampleValueExplicitExampleOverridesEnum(t *testing.T) {
	s := &SchemaRef{
		Type:    "string",
		Enum:    []any{"a", "b"},
		Example: "custom",
	}
	got := ExampleValue(s)
	if got != "custom" {
		t.Errorf("expected \"custom\", got %v", got)
	}
}

func TestExampleValueArrayNoItems(t *testing.T) {
	s := &SchemaRef{Type: "array"}
	got := ExampleValue(s)
	expected := []any{}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected empty array, got %v", got)
	}
}
