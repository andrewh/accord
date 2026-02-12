// Unit tests for Pact matching rule translation.
package convert

import (
	"encoding/json"
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func TestTranslateMatcher(t *testing.T) {
	tests := []struct {
		name     string
		matcher  PactMatcher
		wantRule contract.MatchingRule
		wantWarn bool
	}{
		{
			name:     "type",
			matcher:  PactMatcher{Match: "type"},
			wantRule: contract.MatchingRule{Match: "type"},
		},
		{
			name:     "regex",
			matcher:  PactMatcher{Match: "regex", Regex: "^[a-z]+$"},
			wantRule: contract.MatchingRule{Match: "regex", Regex: "^[a-z]+$"},
		},
		{
			name:     "include",
			matcher:  PactMatcher{Match: "include", Value: "hello"},
			wantRule: contract.MatchingRule{Match: "includes", Includes: "hello"},
		},
		{
			name:     "equality",
			matcher:  PactMatcher{Match: "equality"},
			wantRule: contract.MatchingRule{Match: "exact"},
		},
		{
			name:     "values",
			matcher:  PactMatcher{Match: "values"},
			wantRule: contract.MatchingRule{Match: "exact"},
		},
		{
			name:     "min",
			matcher:  PactMatcher{Match: "min", Min: floatPtr(5)},
			wantRule: contract.MatchingRule{Match: "min", Min: floatPtr(5)},
		},
		{
			name:     "max",
			matcher:  PactMatcher{Match: "max", Max: floatPtr(100)},
			wantRule: contract.MatchingRule{Match: "max", Max: floatPtr(100)},
		},
		{
			name:     "integer maps to type with warning",
			matcher:  PactMatcher{Match: "integer"},
			wantRule: contract.MatchingRule{Match: "type"},
			wantWarn: true,
		},
		{
			name:     "decimal maps to type with warning",
			matcher:  PactMatcher{Match: "decimal"},
			wantRule: contract.MatchingRule{Match: "type"},
			wantWarn: true,
		},
		{
			name:     "number maps to type",
			matcher:  PactMatcher{Match: "number"},
			wantRule: contract.MatchingRule{Match: "type"},
			wantWarn: true,
		},
		{
			name:     "timestamp maps to datetime with warning",
			matcher:  PactMatcher{Match: "timestamp"},
			wantRule: contract.MatchingRule{Match: "datetime"},
			wantWarn: true,
		},
		{
			name:     "date maps to datetime with warning",
			matcher:  PactMatcher{Match: "date"},
			wantRule: contract.MatchingRule{Match: "datetime"},
			wantWarn: true,
		},
		{
			name:     "time maps to datetime with warning",
			matcher:  PactMatcher{Match: "time"},
			wantRule: contract.MatchingRule{Match: "datetime"},
			wantWarn: true,
		},
		{
			name:     "null maps to exact with warning",
			matcher:  PactMatcher{Match: "null"},
			wantRule: contract.MatchingRule{Match: "exact"},
			wantWarn: true,
		},
		{
			name:     "unknown maps to exact with warning",
			matcher:  PactMatcher{Match: "contentType"},
			wantRule: contract.MatchingRule{Match: "exact"},
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, warnings := translateMatcher(tt.matcher, "$.body.field")
			if rule.Match != tt.wantRule.Match {
				t.Errorf("match: got %q, want %q", rule.Match, tt.wantRule.Match)
			}
			if rule.Regex != tt.wantRule.Regex {
				t.Errorf("regex: got %q, want %q", rule.Regex, tt.wantRule.Regex)
			}
			if rule.Includes != tt.wantRule.Includes {
				t.Errorf("includes: got %q, want %q", rule.Includes, tt.wantRule.Includes)
			}
			if (rule.Min == nil) != (tt.wantRule.Min == nil) {
				t.Errorf("min: got %v, want %v", rule.Min, tt.wantRule.Min)
			} else if rule.Min != nil && *rule.Min != *tt.wantRule.Min {
				t.Errorf("min: got %v, want %v", *rule.Min, *tt.wantRule.Min)
			}
			if (rule.Max == nil) != (tt.wantRule.Max == nil) {
				t.Errorf("max: got %v, want %v", rule.Max, tt.wantRule.Max)
			} else if rule.Max != nil && *rule.Max != *tt.wantRule.Max {
				t.Errorf("max: got %v, want %v", *rule.Max, *tt.wantRule.Max)
			}
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning, got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestConvertMatchingRulesV2(t *testing.T) {
	raw := json.RawMessage(`{
		"$.body.id": {"match": "type"},
		"$.body.name": {"match": "type"},
		"$.body.email": {"match": "regex", "regex": "^[^@]+@[^@]+$"},
		"$.header.Content-Type": {"match": "regex", "regex": "application/json.*"}
	}`)

	rules, warnings := convertMatchingRulesV2(raw)

	if len(rules) != 4 {
		t.Fatalf("rules: got %d, want 4", len(rules))
	}

	// Body paths should be preserved as-is.
	if r, ok := rules["$.body.id"]; !ok || r.Match != "type" {
		t.Errorf("$.body.id: got %v", rules["$.body.id"])
	}
	if r, ok := rules["$.body.email"]; !ok || r.Regex != "^[^@]+@[^@]+$" {
		t.Errorf("$.body.email: got %v", rules["$.body.email"])
	}

	// Header path should be translated from $.header.X to $.headers.X.
	if r, ok := rules["$.headers.Content-Type"]; !ok || r.Regex != "application/json.*" {
		t.Errorf("$.headers.Content-Type: got %v, keys: %v", rules["$.headers.Content-Type"], keys(rules))
	}

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestConvertMatchingRulesV2SkipsPath(t *testing.T) {
	raw := json.RawMessage(`{
		"$.body.id": {"match": "type"},
		"$.path": {"match": "regex", "regex": "/users/\\d+"}
	}`)

	rules, warnings := convertMatchingRulesV2(raw)

	if len(rules) != 1 {
		t.Fatalf("rules: got %d, want 1 (path should be skipped)", len(rules))
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings: got %d, want 1", len(warnings))
	}
	if warnings[0].Message == "" {
		t.Error("expected non-empty warning message")
	}
}

func TestConvertMatchingRulesV3(t *testing.T) {
	raw := json.RawMessage(`{
		"body": {
			"$.id": {
				"matchers": [{"match": "type"}],
				"combine": "AND"
			},
			"$.name": {
				"matchers": [{"match": "type"}]
			},
			"$.email": {
				"matchers": [{"match": "regex", "regex": "^[^@]+@[^@]+$"}]
			}
		},
		"header": {
			"Content-Type": {
				"matchers": [{"match": "regex", "regex": "application/json.*"}]
			}
		}
	}`)

	rules, warnings := convertMatchingRulesV3(raw)

	if len(rules) != 4 {
		t.Fatalf("rules: got %d, want 4", len(rules))
	}

	// body + $.field -> $.body.field
	if r, ok := rules["$.body.id"]; !ok || r.Match != "type" {
		t.Errorf("$.body.id: got %v", rules["$.body.id"])
	}
	if r, ok := rules["$.body.email"]; !ok || r.Regex != "^[^@]+@[^@]+$" {
		t.Errorf("$.body.email: got %v", rules["$.body.email"])
	}

	// header + key -> $.headers.key
	if r, ok := rules["$.headers.Content-Type"]; !ok || r.Regex != "application/json.*" {
		t.Errorf("$.headers.Content-Type: got %v, keys: %v", rules["$.headers.Content-Type"], keys(rules))
	}

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestConvertMatchingRulesV3SkipsPathAndQuery(t *testing.T) {
	raw := json.RawMessage(`{
		"body": {
			"$.id": {"matchers": [{"match": "type"}]}
		},
		"path": {
			"$": {"matchers": [{"match": "regex", "regex": "/users/\\d+"}]}
		},
		"query": {
			"page": {"matchers": [{"match": "type"}]}
		}
	}`)

	rules, warnings := convertMatchingRulesV3(raw)

	if len(rules) != 1 {
		t.Fatalf("rules: got %d, want 1", len(rules))
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings: got %d, want 2 (path + query skipped)", len(warnings))
	}
}

func TestConvertMatchingRulesV3MultipleMatchersWarning(t *testing.T) {
	raw := json.RawMessage(`{
		"body": {
			"$.id": {
				"matchers": [
					{"match": "type"},
					{"match": "regex", "regex": "\\d+"}
				],
				"combine": "OR"
			}
		}
	}`)

	rules, warnings := convertMatchingRulesV3(raw)

	if len(rules) != 1 {
		t.Fatalf("rules: got %d, want 1", len(rules))
	}
	// Should take first matcher.
	if rules["$.body.id"].Match != "type" {
		t.Errorf("expected first matcher (type), got %q", rules["$.body.id"].Match)
	}

	// Should warn about multiple matchers and OR combine.
	if len(warnings) < 1 {
		t.Error("expected warnings for multiple matchers / OR combine")
	}
}

func TestConvertMatchingRulesV2EmptyInput(t *testing.T) {
	rules, warnings := convertMatchingRulesV2(nil)
	if len(rules) != 0 {
		t.Errorf("expected empty rules, got %d", len(rules))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestConvertMatchingRulesV3EmptyInput(t *testing.T) {
	rules, warnings := convertMatchingRulesV3(nil)
	if len(rules) != 0 {
		t.Errorf("expected empty rules, got %d", len(rules))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func keys(m contract.MatchingRules) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
