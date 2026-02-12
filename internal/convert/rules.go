// Matching rule translation from Pact v2/v3 to Accord format.
package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andrewh/accord/internal/contract"
)

// PactMatcher represents a single Pact matcher entry.
type PactMatcher struct {
	Match   string   `json:"match"`
	Regex   string   `json:"regex,omitempty"`
	Value   string   `json:"value,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Format  string   `json:"format,omitempty"`
}

// pactV3MatcherSet holds matchers and combine mode for a v3 matching rule.
type pactV3MatcherSet struct {
	Matchers []PactMatcher `json:"matchers"`
	Combine  string        `json:"combine,omitempty"`
}

// translateMatcher converts a single Pact matcher to an Accord matching rule.
func translateMatcher(m PactMatcher, path string) (contract.MatchingRule, []Warning) {
	var warnings []Warning

	switch m.Match {
	case "type":
		return contract.MatchingRule{Match: "type"}, nil

	case "regex":
		return contract.MatchingRule{Match: "regex", Regex: m.Regex}, nil

	case "include":
		return contract.MatchingRule{Match: "includes", Includes: m.Value}, nil

	case "equality", "values":
		return contract.MatchingRule{Match: "exact"}, nil

	case "min":
		return contract.MatchingRule{Match: "min", Min: m.Min}, nil

	case "max":
		return contract.MatchingRule{Match: "max", Max: m.Max}, nil

	case "integer", "decimal", "number":
		warnings = append(warnings, Warning{
			Message: fmt.Sprintf("%s: Pact %q matcher mapped to type (Accord does not distinguish numeric subtypes)", path, m.Match),
		})
		return contract.MatchingRule{Match: "type"}, warnings

	case "timestamp", "date", "time":
		warnings = append(warnings, Warning{
			Message: fmt.Sprintf("%s: Pact %q matcher mapped to datetime (Java date format may need manual review)", path, m.Match),
		})
		return contract.MatchingRule{Match: "datetime"}, warnings

	case "null":
		warnings = append(warnings, Warning{
			Message: fmt.Sprintf("%s: Pact null matcher mapped to exact (no direct Accord equivalent)", path),
		})
		return contract.MatchingRule{Match: "exact"}, warnings

	default:
		warnings = append(warnings, Warning{
			Message: fmt.Sprintf("%s: unknown Pact matcher %q mapped to exact", path, m.Match),
		})
		return contract.MatchingRule{Match: "exact"}, warnings
	}
}

// convertMatchingRulesV2 converts v2 flat matching rules to Accord format.
// V2 structure: {"$.body.id": {"match": "type"}, "$.header.X": {"match": "regex", ...}}
func convertMatchingRulesV2(raw json.RawMessage) (contract.MatchingRules, []Warning) {
	if len(raw) == 0 {
		return nil, nil
	}

	var flat map[string]PactMatcher
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, []Warning{{Message: fmt.Sprintf("failed to parse v2 matching rules: %v", err)}}
	}

	rules := make(contract.MatchingRules)
	var warnings []Warning

	for pactPath, matcher := range flat {
		accordPath := translateV2Path(pactPath)
		if accordPath == "" {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("skipping unsupported matching rule path %q", pactPath),
			})
			continue
		}

		rule, ws := translateMatcher(matcher, accordPath)
		warnings = append(warnings, ws...)
		rules[accordPath] = rule
	}

	return rules, warnings
}

// translateV2Path converts a v2 matching rule path to an Accord path.
// Returns empty string for unsupported paths ($.path, $.query).
func translateV2Path(pactPath string) string {
	if pactPath == "$.path" || strings.HasPrefix(pactPath, "$.query") {
		return ""
	}
	// $.header.X -> $.headers.X
	if strings.HasPrefix(pactPath, "$.header.") {
		return "$.headers." + pactPath[len("$.header."):]
	}
	return pactPath
}

// convertMatchingRulesV3 converts v3 categorised matching rules to Accord format.
// V3 structure: {"body": {"$.id": {"matchers": [...]}}, "header": {"X": {"matchers": [...]}}}
func convertMatchingRulesV3(raw json.RawMessage) (contract.MatchingRules, []Warning) {
	if len(raw) == 0 {
		return nil, nil
	}

	var categories map[string]map[string]pactV3MatcherSet
	if err := json.Unmarshal(raw, &categories); err != nil {
		return nil, []Warning{{Message: fmt.Sprintf("failed to parse v3 matching rules: %v", err)}}
	}

	rules := make(contract.MatchingRules)
	var warnings []Warning

	for category, paths := range categories {
		switch category {
		case "path", "query":
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("skipping unsupported matching rule category %q", category),
			})
			continue
		}

		for fieldPath, ms := range paths {
			accordPath := translateV3Path(category, fieldPath)

			if ms.Combine == "OR" {
				warnings = append(warnings, Warning{
					Message: fmt.Sprintf("%s: combine OR not supported, using first matcher only", accordPath),
				})
			}
			if len(ms.Matchers) > 1 {
				warnings = append(warnings, Warning{
					Message: fmt.Sprintf("%s: multiple matchers found, using first only", accordPath),
				})
			}

			if len(ms.Matchers) == 0 {
				continue
			}

			rule, ws := translateMatcher(ms.Matchers[0], accordPath)
			warnings = append(warnings, ws...)
			rules[accordPath] = rule
		}
	}

	return rules, warnings
}

// translateV3Path builds an Accord path from a v3 category and field path.
func translateV3Path(category, fieldPath string) string {
	switch category {
	case "body":
		// $.field -> $.body.field
		if strings.HasPrefix(fieldPath, "$.") {
			return "$.body." + fieldPath[2:]
		}
		return "$.body." + fieldPath
	case "header", "headers":
		return "$.headers." + fieldPath
	case "status":
		return "$.status"
	default:
		return "$." + category + "." + fieldPath
	}
}
