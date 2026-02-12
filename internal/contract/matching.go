// Matching rule evaluation: path resolution and value comparison.
package contract

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
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
	for _, raw := range segments[1:] {
		seg, err := ParseSegment(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid segment in path %q: %w", path, err)
		}

		if seg.Wildcard {
			return nil, fmt.Errorf("wildcard [*] not supported in ResolvePath: %q", path)
		}

		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse into %T at segment %q in path %q", current, raw, path)
		}
		val, exists := m[seg.Field]
		if !exists {
			return nil, fmt.Errorf("field %q not found in path %q", seg.Field, path)
		}

		if seg.Index >= 0 {
			arr, ok := val.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array at %q in path %q, got %T", seg.Field, path, val)
			}
			if seg.Index >= len(arr) {
				return nil, fmt.Errorf("index %d out of bounds (length %d) at %q in path %q", seg.Index, len(arr), seg.Field, path)
			}
			current = arr[seg.Index]
		} else {
			current = val
		}
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

// MatchMin checks that the numeric value of actual is >= min.
func MatchMin(min float64, actual any) error {
	f, err := toFloat64(actual)
	if err != nil {
		return fmt.Errorf("cannot compare non-numeric value %v (%T) against min", actual, actual)
	}
	if f < min {
		return fmt.Errorf("value %v is less than minimum %v", actual, min)
	}
	return nil
}

// MatchMax checks that the numeric value of actual is <= max.
func MatchMax(max float64, actual any) error {
	f, err := toFloat64(actual)
	if err != nil {
		return fmt.Errorf("cannot compare non-numeric value %v (%T) against max", actual, actual)
	}
	if f > max {
		return fmt.Errorf("value %v is greater than maximum %v", actual, max)
	}
	return nil
}

// MatchIncludes checks that the string representation of actual contains substr.
func MatchIncludes(substr string, actual any) error {
	s := fmt.Sprintf("%v", actual)
	if !strings.Contains(s, substr) {
		return fmt.Errorf("value %q does not contain %q", s, substr)
	}
	return nil
}

// MatchDatetime checks that actual is a string matching the given time format.
// If format is empty, defaults to RFC3339.
func MatchDatetime(format string, actual any) error {
	s, ok := actual.(string)
	if !ok {
		return fmt.Errorf("expected string for datetime, got %T", actual)
	}
	if format == "" {
		format = time.RFC3339
	}
	if _, err := time.Parse(format, s); err != nil {
		return fmt.Errorf("value %q does not match datetime format %q: %w", s, format, err)
	}
	return nil
}

// MatchEnum checks that the string representation of actual is one of the allowed values.
func MatchEnum(values []string, actual any) error {
	s := fmt.Sprintf("%v", actual)
	for _, v := range values {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("value %q is not one of %v", s, values)
}

// MatchNotNull checks that actual is not nil.
func MatchNotNull(actual any) error {
	if actual == nil {
		return fmt.Errorf("expected non-null value, got nil")
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
	case "min":
		return MatchMin(*rule.Min, actual)
	case "max":
		return MatchMax(*rule.Max, actual)
	case "includes":
		return MatchIncludes(rule.Includes, actual)
	case "datetime":
		return MatchDatetime(rule.Format, actual)
	case "enum":
		return MatchEnum(rule.Values, actual)
	case "not_null":
		return MatchNotNull(actual)
	default:
		return fmt.Errorf("unknown match type: %q", match)
	}
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	default:
		return 0, fmt.Errorf("not a numeric type: %T", v)
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
