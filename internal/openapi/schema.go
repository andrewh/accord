// Walk OpenAPI schemas to produce example values for contract generation.
package openapi

const maxDepth = 10

// ExampleValue generates a representative value from a schema.
// It prefers explicit examples, then enum values, then format-aware defaults.
func ExampleValue(s *SchemaRef) any {
	return exampleValue(s, 0)
}

func exampleValue(s *SchemaRef, depth int) any {
	if s == nil || depth > maxDepth {
		return nil
	}

	// Explicit example takes priority.
	if s.Example != nil {
		return s.Example
	}

	// allOf: merge all schemas' required properties.
	if len(s.AllOf) > 0 {
		return exampleValueAllOf(s.AllOf, depth)
	}

	// oneOf/anyOf: pick first variant.
	if len(s.OneOf) > 0 {
		return exampleValue(s.OneOf[0], depth+1)
	}
	if len(s.AnyOf) > 0 {
		return exampleValue(s.AnyOf[0], depth+1)
	}

	// Enum values.
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}

	switch s.Type {
	case "string":
		return exampleString(s)
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return false
	case "array":
		return exampleArray(s, depth)
	case "object":
		return exampleObject(s, depth)
	default:
		return nil
	}
}

func exampleString(s *SchemaRef) any {
	switch s.Format {
	case "date-time":
		return "2024-01-01T00:00:00Z"
	case "date":
		return "2024-01-01"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "uri", "url":
		return "https://example.com"
	default:
		return "string"
	}
}

func exampleArray(s *SchemaRef, depth int) any {
	if s.Items == nil {
		return []any{}
	}
	return []any{exampleValue(s.Items, depth+1)}
}

func exampleObject(s *SchemaRef, depth int) any {
	result := make(map[string]any)
	required := make(map[string]bool, len(s.Required))
	for _, r := range s.Required {
		required[r] = true
	}
	for name, prop := range s.Properties {
		if required[name] {
			result[name] = exampleValue(prop, depth+1)
		}
	}
	return result
}

func exampleValueAllOf(schemas []*SchemaRef, depth int) any {
	merged := make(map[string]any)
	for _, sub := range schemas {
		v := exampleValue(sub, depth+1)
		if m, ok := v.(map[string]any); ok {
			for k, val := range m {
				merged[k] = val
			}
		}
	}
	return merged
}
