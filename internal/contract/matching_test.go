// Tests for matching rule path resolution and matchers.
package contract

import (
	"reflect"
	"testing"
)

func TestResolvePath(t *testing.T) {
	body := map[string]any{
		"id":   123,
		"name": "Jane",
		"address": map[string]any{
			"city": "London",
		},
	}

	tests := []struct {
		path    string
		data    any
		want    any
		wantErr bool
	}{
		{"$.body.id", body, 123, false},
		{"$.body.name", body, "Jane", false},
		{"$.body.address.city", body, "London", false},
		{"$.body.missing", body, nil, true},
		{"$.body.address.missing", body, nil, true},
		{"$.body.id", 42, nil, true}, // non-map body with plain segment
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ResolvePath(tt.path, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got value %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolvePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolvePathArrayIndex(t *testing.T) {
	body := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "email": "alice@example.com"},
			map[string]any{"name": "Bob", "email": "bob@example.com"},
		},
		"tags": []any{"go", "testing"},
	}

	tests := []struct {
		path    string
		want    any
		wantErr bool
	}{
		{"$.body.users[0].name", "Alice", false},
		{"$.body.users[1].email", "bob@example.com", false},
		{"$.body.tags[0]", "go", false},
		{"$.body.tags[1]", "testing", false},
		{"$.body.users[5]", nil, true},             // out of bounds
		{"$.body.tags[0].nested", nil, true},        // traversing into scalar
		{"$.body.users[*]", nil, true},              // wildcard not supported in ResolvePath
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ResolvePath(tt.path, body)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got value %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolvePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolvePathReturnsWholeArray(t *testing.T) {
	body := map[string]any{
		"items": []any{"a", "b", "c"},
	}
	got, err := ResolvePath("$.body.items", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []any{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ResolvePath = %v, want %v", got, want)
	}
}

func TestResolvePathStatus(t *testing.T) {
	got, err := ResolvePathInResponse("$.status", 200, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 200 {
		t.Errorf("got %v, want 200", got)
	}
}

func TestResolvePathHeaders(t *testing.T) {
	headers := map[string]string{"Content-Type": "application/json"}
	got, err := ResolvePathInResponse("$.headers.Content-Type", 0, headers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "application/json" {
		t.Errorf("got %v, want %q", got, "application/json")
	}
}

func TestMatchExact(t *testing.T) {
	tests := []struct {
		name     string
		expected any
		actual   any
		wantOK   bool
	}{
		{"same string", "hello", "hello", true},
		{"different string", "hello", "world", false},
		{"same int", 42, 42, true},
		{"different int", 42, 43, false},
		{"same bool", true, true, true},
		{"different bool", true, false, false},
		{"nil match", nil, nil, true},
		{"nil vs value", nil, "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchExact(tt.expected, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchType(t *testing.T) {
	tests := []struct {
		name     string
		expected any
		actual   any
		wantOK   bool
	}{
		{"string to string", "hello", "world", true},
		{"int to int", 42, 99, true},
		{"float to float", 1.5, 2.5, true},
		{"int to float", 42, 42.0, true},       // both numeric
		{"float to int", 1.5, 2, true},          // both numeric
		{"bool to bool", true, false, true},
		{"string to int", "hello", 42, false},
		{"nil to nil", nil, nil, true},
		{"nil to string", nil, "x", false},
		{"string to nil", "x", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchType(tt.expected, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchRegex(t *testing.T) {
	tests := []struct {
		name   string
		regex  string
		actual any
		wantOK bool
	}{
		{"email match", "^[^@]+@[^@]+$", "jane@example.com", true},
		{"email no match", "^[^@]+@[^@]+$", "not-an-email", false},
		{"number as string", "^\\d+$", 123, true},
		{"bool as string", "^true$", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchRegex(tt.regex, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchRegexInvalid(t *testing.T) {
	err := MatchRegex("[invalid", "test")
	if err == nil {
		t.Error("expected error for invalid regex, got nil")
	}
}

func TestApplyRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     MatchingRule
		expected any
		actual   any
		wantOK   bool
	}{
		{"exact match", MatchingRule{Match: "exact"}, "hello", "hello", true},
		{"exact mismatch", MatchingRule{Match: "exact"}, "hello", "world", false},
		{"type match", MatchingRule{Match: "type"}, 42, 99, true},
		{"type mismatch", MatchingRule{Match: "type"}, 42, "hello", false},
		{"regex match", MatchingRule{Match: "regex", Regex: "^[A-Z]"}, "", "Jane", true},
		{"regex mismatch", MatchingRule{Match: "regex", Regex: "^[A-Z]"}, "", "jane", false},
		{"default is exact", MatchingRule{}, "hello", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyRule(tt.rule, tt.expected, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchMin(t *testing.T) {
	tests := []struct {
		name   string
		min    float64
		actual any
		wantOK bool
	}{
		{"int above min", 5, 10, true},
		{"int below min", 5, 3, false},
		{"int at boundary", 5, 5, true},
		{"float64 above min", 1.5, 2.0, true},
		{"float64 below min", 1.5, 1.0, false},
		{"float64 at boundary", 1.5, 1.5, true},
		{"non-numeric", 5, "hello", false},
		{"zero min", 0, 0, true},
		{"negative value above negative min", -10, -5, true},
		{"negative value below negative min", -10, -15, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchMin(tt.min, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchMax(t *testing.T) {
	tests := []struct {
		name   string
		max    float64
		actual any
		wantOK bool
	}{
		{"int below max", 10, 5, true},
		{"int above max", 10, 15, false},
		{"int at boundary", 10, 10, true},
		{"float64 below max", 2.0, 1.5, true},
		{"float64 above max", 2.0, 2.5, false},
		{"float64 at boundary", 2.0, 2.0, true},
		{"non-numeric", 5, "hello", false},
		{"zero max", 0, 0, true},
		{"negative value below negative max", -5, -10, true},
		{"negative value above negative max", -10, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchMax(tt.max, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchIncludes(t *testing.T) {
	tests := []struct {
		name   string
		substr string
		actual any
		wantOK bool
	}{
		{"string contains substring", "world", "hello world", true},
		{"string does not contain", "xyz", "hello world", false},
		{"exact match", "hello", "hello", true},
		{"non-string actual contains", "42", 42, true},
		{"bool contains", "true", true, true},
		{"empty substring always matches", "", "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchIncludes(tt.substr, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchDatetime(t *testing.T) {
	tests := []struct {
		name   string
		format string
		actual any
		wantOK bool
	}{
		{"RFC3339 default", "", "2024-01-15T10:30:00Z", true},
		{"RFC3339 with offset", "", "2024-01-15T10:30:00+01:00", true},
		{"invalid datetime default", "", "not-a-date", false},
		{"custom format", "2006-01-02", "2024-01-15", true},
		{"custom format mismatch", "2006-01-02", "15/01/2024", false},
		{"non-string actual", "", 12345, false},
		{"date only format", "02/01/2006", "15/01/2024", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchDatetime(tt.format, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchEnum(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		actual any
		wantOK bool
	}{
		{"member string", []string{"active", "inactive", "pending"}, "active", true},
		{"non-member string", []string{"active", "inactive", "pending"}, "deleted", false},
		{"numeric stringified", []string{"1", "2", "3"}, 2, true},
		{"bool stringified", []string{"true", "false"}, true, true},
		{"empty values", []string{}, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchEnum(tt.values, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestMatchNotNull(t *testing.T) {
	tests := []struct {
		name   string
		actual any
		wantOK bool
	}{
		{"non-nil string", "hello", true},
		{"non-nil int", 42, true},
		{"non-nil bool", false, true},
		{"nil value", nil, false},
		{"empty string is not null", "", true},
		{"zero is not null", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchNotNull(tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestApplyRuleNewTypes(t *testing.T) {
	minVal := 5.0
	maxVal := 100.0

	tests := []struct {
		name     string
		rule     MatchingRule
		expected any
		actual   any
		wantOK   bool
	}{
		{"min pass", MatchingRule{Match: "min", Min: &minVal}, nil, 10, true},
		{"min fail", MatchingRule{Match: "min", Min: &minVal}, nil, 3, false},
		{"max pass", MatchingRule{Match: "max", Max: &maxVal}, nil, 50, true},
		{"max fail", MatchingRule{Match: "max", Max: &maxVal}, nil, 150, false},
		{"includes pass", MatchingRule{Match: "includes", Includes: "world"}, nil, "hello world", true},
		{"includes fail", MatchingRule{Match: "includes", Includes: "xyz"}, nil, "hello world", false},
		{"datetime pass", MatchingRule{Match: "datetime"}, nil, "2024-01-15T10:30:00Z", true},
		{"datetime fail", MatchingRule{Match: "datetime"}, nil, "not-a-date", false},
		{"enum pass", MatchingRule{Match: "enum", Values: []string{"a", "b", "c"}}, nil, "b", true},
		{"enum fail", MatchingRule{Match: "enum", Values: []string{"a", "b", "c"}}, nil, "z", false},
		{"not_null pass", MatchingRule{Match: "not_null"}, nil, "something", true},
		{"not_null fail", MatchingRule{Match: "not_null"}, nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyRule(tt.rule, tt.expected, tt.actual)
			if tt.wantOK && err != nil {
				t.Errorf("expected match, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("expected mismatch, got match")
			}
		})
	}
}

func TestApplyRuleMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		rule MatchingRule
	}{
		{"min without min value", MatchingRule{Match: "min"}},
		{"max without max value", MatchingRule{Match: "max"}},
		{"includes without includes value", MatchingRule{Match: "includes"}},
		{"enum without values", MatchingRule{Match: "enum"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyRule(tt.rule, nil, 42)
			if err == nil {
				t.Error("expected error for missing required field, got nil")
			}
		})
	}
}

func TestApplyRuleUnknownType(t *testing.T) {
	err := ApplyRule(MatchingRule{Match: "unknown"}, "a", "b")
	if err == nil {
		t.Error("expected error for unknown match type, got nil")
	}
}
