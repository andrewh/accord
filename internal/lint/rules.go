// Individual lint rule implementations for Accord contracts.
package lint

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/andrewh/accord/internal/contract"
	"gopkg.in/yaml.v3"
)

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

var validMatchTypes = map[string]bool{
	"exact": true, "type": true, "regex": true,
	"min": true, "max": true, "includes": true,
	"datetime": true, "enum": true, "not_null": true,
}

// validBracket matches well-formed bracket syntax: [digits] or [*].
var validBracket = regexp.MustCompile(`^\[\d+\]$|^\[\*\]$`)

var validSeverities = map[string]bool{
	"error":   true,
	"warning": true,
}

// DefaultRules returns the complete set of MVP lint rules.
func DefaultRules() []Rule {
	return []Rule{
		ruleRequiredTopLevel,
		ruleInteractions,
		ruleDuplicateDescriptions,
		ruleMatchingRules,
		ruleNFR,
	}
}

// ruleRequiredTopLevel checks that accord version, consumer.name, and provider.name are present.
func ruleRequiredTopLevel(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var diags []Diagnostic

	if c.Accord == "" {
		line, col := findKeyPosition(node, "accord")
		diags = append(diags, Diagnostic{
			Severity: Error,
			Message:  "accord version is required",
			Line:     line, Column: col,
			Path: "accord",
		})
	}

	if c.Consumer.Name == "" {
		line, col := findNestedKeyPosition(node, "consumer", "name")
		diags = append(diags, Diagnostic{
			Severity: Error,
			Message:  "consumer.name is required",
			Line:     line, Column: col,
			Path: "consumer.name",
		})
	}

	if c.Provider.Name == "" {
		line, col := findNestedKeyPosition(node, "provider", "name")
		diags = append(diags, Diagnostic{
			Severity: Error,
			Message:  "provider.name is required",
			Line:     line, Column: col,
			Path: "provider.name",
		})
	}

	if len(c.Interactions) == 0 {
		line, col := findKeyPosition(node, "interactions")
		diags = append(diags, Diagnostic{
			Severity: Error,
			Message:  "at least one interaction is required",
			Line:     line, Column: col,
			Path: "interactions",
		})
	}

	return diags
}

// ruleInteractions checks each interaction for required fields and valid values.
func ruleInteractions(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var diags []Diagnostic

	for i, ix := range c.Interactions {
		prefix := fmt.Sprintf("interactions[%d]", i)
		ixNode := findInteractionNode(node, i)

		if ix.Description == "" {
			line, col := interactionFieldPosition(ixNode, "description")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  "description is required",
				Line:     line, Column: col,
				Path: prefix + ".description",
			})
		}

		if ix.Request.Method == "" {
			line, col := interactionNestedFieldPosition(ixNode, "request", "method")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  "request.method is required",
				Line:     line, Column: col,
				Path: prefix + ".request.method",
			})
		} else if !validMethods[strings.ToUpper(ix.Request.Method)] {
			line, col := interactionNestedFieldPosition(ixNode, "request", "method")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("invalid HTTP method: %q", ix.Request.Method),
				Line:     line, Column: col,
				Path: prefix + ".request.method",
			})
		}

		if ix.Request.Path == "" {
			line, col := interactionNestedFieldPosition(ixNode, "request", "path")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  "request.path is required",
				Line:     line, Column: col,
				Path: prefix + ".request.path",
			})
		}

		if ix.Response.Status == 0 {
			line, col := interactionNestedFieldPosition(ixNode, "response", "status")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  "response.status is required",
				Line:     line, Column: col,
				Path: prefix + ".response.status",
			})
		} else if ix.Response.Status < 100 || ix.Response.Status > 599 {
			line, col := interactionNestedFieldPosition(ixNode, "response", "status")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("invalid status code: %d (must be 100-599)", ix.Response.Status),
				Line:     line, Column: col,
				Path: prefix + ".response.status",
			})
		}
	}

	return diags
}

// ruleDuplicateDescriptions checks for duplicate interaction descriptions.
func ruleDuplicateDescriptions(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var diags []Diagnostic
	seen := make(map[string]int)

	for i, ix := range c.Interactions {
		if ix.Description == "" {
			continue
		}
		if first, ok := seen[ix.Description]; ok {
			ixNode := findInteractionNode(node, i)
			line, col := interactionFieldPosition(ixNode, "description")
			diags = append(diags, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("duplicate interaction description %q (first at interactions[%d])", ix.Description, first),
				Line:     line, Column: col,
				Path: fmt.Sprintf("interactions[%d].description", i),
			})
		} else {
			seen[ix.Description] = i
		}
	}

	return diags
}

// ruleMatchingRules validates matching rule paths and types.
func ruleMatchingRules(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var diags []Diagnostic

	for i, ix := range c.Interactions {
		prefix := fmt.Sprintf("interactions[%d].matching_rules", i)

		for path, rule := range ix.MatchingRules {
			if !strings.HasPrefix(path, "$.") {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Warning,
					Message:  fmt.Sprintf("matching rule path %q should start with %q", path, "$."),
					Line:     line, Column: col,
					Path: prefix + "[" + path + "]",
				})
			}

			if diag, ok := validateBrackets(path, prefix, findInteractionNode(node, i)); !ok {
			diags = append(diags, diag)
		}

		matchType := rule.Match
			if matchType == "" {
				matchType = "exact"
			}
			if !validMatchTypes[matchType] {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("unknown match type %q", rule.Match),
					Line:     line, Column: col,
					Path: prefix + "[" + path + "].match",
				})
			}

			if rule.Match == "regex" {
				if rule.Regex == "" {
					ixNode := findInteractionNode(node, i)
					line, col := matchingRuleKeyPosition(ixNode, path)
					diags = append(diags, Diagnostic{
						Severity: Error,
						Message:  "regex field is required when match is \"regex\"",
						Line:     line, Column: col,
						Path: prefix + "[" + path + "].regex",
					})
				} else if _, err := regexp.Compile(rule.Regex); err != nil {
					ixNode := findInteractionNode(node, i)
					line, col := matchingRuleKeyPosition(ixNode, path)
					diags = append(diags, Diagnostic{
						Severity: Error,
						Message:  fmt.Sprintf("invalid regex %q: %v", rule.Regex, err),
						Line:     line, Column: col,
						Path: prefix + "[" + path + "].regex",
					})
				}
			}

			if rule.Match == "min" && rule.Min == nil {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  "min field is required when match is \"min\"",
					Line:     line, Column: col,
					Path: prefix + "[" + path + "].min",
				})
			}

			if rule.Match == "max" && rule.Max == nil {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  "max field is required when match is \"max\"",
					Line:     line, Column: col,
					Path: prefix + "[" + path + "].max",
				})
			}

			if rule.Match == "enum" && len(rule.Values) == 0 {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  "values field is required when match is \"enum\"",
					Line:     line, Column: col,
					Path: prefix + "[" + path + "].values",
				})
			}

			if rule.Match == "includes" && rule.Includes == "" {
				ixNode := findInteractionNode(node, i)
				line, col := matchingRuleKeyPosition(ixNode, path)
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  "includes field is required when match is \"includes\"",
					Line:     line, Column: col,
					Path: prefix + "[" + path + "].includes",
				})
			}
		}
	}

	return diags
}

// validateBrackets checks that any bracket notation in a matching rule path is well-formed.
func validateBrackets(path, prefix string, ixNode *yaml.Node) (Diagnostic, bool) {
	for i := 0; i < len(path); i++ {
		if path[i] != '[' {
			continue
		}
		close := strings.IndexByte(path[i:], ']')
		if close == -1 {
			line, col := matchingRuleKeyPosition(ixNode, path)
			return Diagnostic{
				Severity: Warning,
				Message:  fmt.Sprintf("invalid bracket syntax in path %q: unclosed bracket", path),
				Line:     line, Column: col,
				Path: prefix + "[" + path + "]",
			}, false
		}
		bracket := path[i : i+close+1]
		if !validBracket.MatchString(bracket) {
			line, col := matchingRuleKeyPosition(ixNode, path)
			return Diagnostic{
				Severity: Warning,
				Message:  fmt.Sprintf("invalid bracket syntax in path %q: %s", path, bracket),
				Line:     line, Column: col,
				Path: prefix + "[" + path + "]",
			}, false
		}
		i += close
	}
	return Diagnostic{}, true
}

// ruleNFR validates NFR thresholds and severity values.
func ruleNFR(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var diags []Diagnostic

	type nfrField struct {
		name      string
		threshold *contract.NFRThreshold
	}

	for i, ix := range c.Interactions {
		if ix.NFR == nil {
			continue
		}
		prefix := fmt.Sprintf("interactions[%d].nfr", i)
		ixNode := findInteractionNode(node, i)

		fields := []nfrField{
			{"max_response_bytes", ix.NFR.MaxResponseBytes},
			{"max_time_to_first_byte_ms", ix.NFR.MaxTimeToFirstByteMs},
			{"max_round_trip_ms", ix.NFR.MaxRoundTripMs},
		}

		for _, f := range fields {
			if f.threshold == nil {
				continue
			}
			if f.threshold.Threshold <= 0 {
				line, col := nfrFieldPosition(ixNode, f.name, "threshold")
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("threshold must be > 0 for %s", f.name),
					Line:     line, Column: col,
					Path: prefix + "." + f.name + ".threshold",
				})
			}
			if f.threshold.Severity != "" && !validSeverities[f.threshold.Severity] {
				line, col := nfrFieldPosition(ixNode, f.name, "severity")
				diags = append(diags, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("invalid severity %q for %s (must be %q or %q)", f.threshold.Severity, f.name, "error", "warning"),
					Line:     line, Column: col,
					Path: prefix + "." + f.name + ".severity",
				})
			}
		}
	}

	return diags
}

// nfrFieldPosition returns the position of a subfield within an NFR entry.
// Navigates: interaction -> nfr -> fieldName -> subfield.
func nfrFieldPosition(ixNode *yaml.Node, fieldName, subfield string) (int, int) {
	if ixNode == nil {
		return 1, 1
	}
	// Find "nfr" key in interaction
	for i := 0; i < len(ixNode.Content)-1; i += 2 {
		if ixNode.Content[i].Value != "nfr" {
			continue
		}
		nfrMapping := ixNode.Content[i+1]
		// Find fieldName (e.g. "max_response_bytes")
		for j := 0; j < len(nfrMapping.Content)-1; j += 2 {
			if nfrMapping.Content[j].Value != fieldName {
				continue
			}
			fieldMapping := nfrMapping.Content[j+1]
			// Find subfield (e.g. "threshold" or "severity")
			for k := 0; k < len(fieldMapping.Content)-1; k += 2 {
				if fieldMapping.Content[k].Value == subfield {
					return fieldMapping.Content[k].Line, fieldMapping.Content[k].Column
				}
			}
			return nfrMapping.Content[j].Line, nfrMapping.Content[j].Column
		}
		return ixNode.Content[i].Line, ixNode.Content[i].Column
	}
	return ixNode.Line, ixNode.Column
}

// findKeyPosition returns the line and column of a top-level key in the YAML document.
// Returns (1, 1) if the key is not found.
func findKeyPosition(root *yaml.Node, key string) (int, int) {
	doc := documentContent(root)
	if doc == nil {
		return 1, 1
	}

	for i := 0; i < len(doc.Content)-1; i += 2 {
		if doc.Content[i].Value == key {
			return doc.Content[i].Line, doc.Content[i].Column
		}
	}
	return 1, 1
}

// findNestedKeyPosition returns the position of a key nested one level deep.
func findNestedKeyPosition(root *yaml.Node, parent, key string) (int, int) {
	doc := documentContent(root)
	if doc == nil {
		return 1, 1
	}

	for i := 0; i < len(doc.Content)-1; i += 2 {
		if doc.Content[i].Value == parent {
			mapping := doc.Content[i+1]
			for j := 0; j < len(mapping.Content)-1; j += 2 {
				if mapping.Content[j].Value == key {
					return mapping.Content[j].Line, mapping.Content[j].Column
				}
			}
			return doc.Content[i].Line, doc.Content[i].Column
		}
	}
	return 1, 1
}

// findInteractionNode returns the YAML node for the i-th interaction.
func findInteractionNode(root *yaml.Node, index int) *yaml.Node {
	doc := documentContent(root)
	if doc == nil {
		return nil
	}

	for i := 0; i < len(doc.Content)-1; i += 2 {
		if doc.Content[i].Value == "interactions" {
			seq := doc.Content[i+1]
			if index < len(seq.Content) {
				return seq.Content[index]
			}
		}
	}
	return nil
}

// interactionFieldPosition returns the position of a field within an interaction node.
func interactionFieldPosition(ixNode *yaml.Node, key string) (int, int) {
	if ixNode == nil {
		return 1, 1
	}
	for i := 0; i < len(ixNode.Content)-1; i += 2 {
		if ixNode.Content[i].Value == key {
			return ixNode.Content[i].Line, ixNode.Content[i].Column
		}
	}
	return ixNode.Line, ixNode.Column
}

// interactionNestedFieldPosition returns the position of a nested field within an interaction.
func interactionNestedFieldPosition(ixNode *yaml.Node, parent, key string) (int, int) {
	if ixNode == nil {
		return 1, 1
	}
	for i := 0; i < len(ixNode.Content)-1; i += 2 {
		if ixNode.Content[i].Value == parent {
			mapping := ixNode.Content[i+1]
			for j := 0; j < len(mapping.Content)-1; j += 2 {
				if mapping.Content[j].Value == key {
					return mapping.Content[j].Line, mapping.Content[j].Column
				}
			}
			return ixNode.Content[i].Line, ixNode.Content[i].Column
		}
	}
	return ixNode.Line, ixNode.Column
}

// matchingRuleKeyPosition returns the position of a matching rule key.
func matchingRuleKeyPosition(ixNode *yaml.Node, rulePath string) (int, int) {
	if ixNode == nil {
		return 1, 1
	}
	for i := 0; i < len(ixNode.Content)-1; i += 2 {
		if ixNode.Content[i].Value == "matching_rules" {
			mapping := ixNode.Content[i+1]
			for j := 0; j < len(mapping.Content)-1; j += 2 {
				if mapping.Content[j].Value == rulePath {
					return mapping.Content[j].Line, mapping.Content[j].Column
				}
			}
			return ixNode.Content[i].Line, ixNode.Content[i].Column
		}
	}
	return ixNode.Line, ixNode.Column
}

// documentContent unwraps the document node to get the root mapping node.
func documentContent(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	if root.Kind == yaml.MappingNode {
		return root
	}
	return nil
}
