// Matching rule evaluation: path resolution and value comparison.
package contract

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ResolvePath extracts a value from a nested map using dot-notation (e.g. "$.body.address.city").
// The path must start with "$.body." and navigates through the body structure.
func ResolvePath(path string, body any) (any, error) {
	if !strings.HasPrefix(path, "$.") {
		return nil, fmt.Errorf("path must start with %q: %q", "$.", path)
	}

	segments := strings.Split(path[2:], ".")
	if len(segments) < 2 || segments[0] != "body" {
		return nil, fmt.Errorf("ResolvePath only handles $.body paths: %q", path)
	}

	current := body
	for _, seg := range segments[1:] {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse into %T at segment %q in path %q", current, seg, path)
		}
		val, exists := m[seg]
		if !exists {
			return nil, fmt.Errorf("field %q not found in path %q", seg, path)
		}
		current = val
	}
	return current, nil
}

// ResolvePathInResponse extracts a value from the response using a dot-notation path.
// Handles $.status, $.headers.X, and $.body.X paths.
func ResolvePathInResponse(path string, status int, headers map[string]string, body any) (any, error) {
	if path == "$.status" {
		return status, nil
	}

	if strings.HasPrefix(path, "$.headers.") {
		key := path[len("$.headers."):]
		val, ok := headers[key]
		if !ok {
			return nil, fmt.Errorf("header %q not found", key)
		}
		return val, nil
	}

	if strings.HasPrefix(path, "$.body") {
		return ResolvePath(path, body)
	}

	return nil, fmt.Errorf("unsupported path: %q", path)
}

// MatchExact checks that actual equals expected using deep equality.
func MatchExact(expected, actual any) error {
	if !reflect.DeepEqual(normaliseNumeric(expected), normaliseNumeric(actual)) {
		return fmt.Errorf("expected %v (%T), got %v (%T)", expected, expected, actual, actual)
	}
	return nil
}

// MatchType checks that actual is the same general type as expected.
// Numeric types (int, float) are treated as the same type.
func MatchType(expected, actual any) error {
	et := generalType(expected)
	at := generalType(actual)
	if et != at {
		return fmt.Errorf("expected type %s, got type %s", et, at)
	}
	return nil
}

// MatchRegex checks that the string representation of actual matches the regex pattern.
func MatchRegex(pattern string, actual any) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	s := fmt.Sprintf("%v", actual)
	if !re.MatchString(s) {
		return fmt.Errorf("value %q does not match pattern %q", s, pattern)
	}
	return nil
}

// ApplyRule applies a matching rule to compare expected and actual values.
// If the rule has no match type set, defaults to exact matching.
func ApplyRule(rule MatchingRule, expected, actual any) error {
	match := rule.Match
	if match == "" {
		match = "exact"
	}

	switch match {
	case "exact":
		return MatchExact(expected, actual)
	case "type":
		return MatchType(expected, actual)
	case "regex":
		return MatchRegex(rule.Regex, actual)
	default:
		return fmt.Errorf("unknown match type: %q", match)
	}
}

// generalType returns a broad type category for matching purposes.
func generalType(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return "number"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return reflect.TypeOf(v).String()
	}
}

// normaliseNumeric converts integer types to float64 for comparison,
// matching how JSON/YAML numbers are typically unmarshalled.
func normaliseNumeric(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	default:
		return v
	}
}
