// Map OpenAPI operations to Accord contract interactions with matching rules.
package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/openapi"
)

// BuildInteractions converts an OpenAPI endpoint into one or more Accord
// interactions. Each 2xx response produces a separate interaction.
func BuildInteractions(ep openapi.Endpoint) []contract.Interaction {
	// Sort status codes for deterministic output.
	codes := make([]int, 0, len(ep.Responses))
	for code := range ep.Responses {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	var interactions []contract.Interaction
	for _, code := range codes {
		schema := ep.Responses[code]
		ix := buildInteraction(ep, code, schema, len(codes) > 1)
		interactions = append(interactions, ix)
	}
	return interactions
}

func buildInteraction(ep openapi.Endpoint, status int, respSchema *openapi.SchemaRef, multipleResponses bool) contract.Interaction {
	desc := interactionDescription(ep, status, multipleResponses)
	path := substitutePath(ep.Path, ep.Parameters)

	ix := contract.Interaction{
		Description: desc,
		Request: contract.Request{
			Method: ep.Method,
			Path:   path,
		},
		Response: contract.Response{
			Status: status,
		},
		MatchingRules: make(contract.MatchingRules),
	}

	// Request body from required fields only.
	if ep.RequestBody != nil {
		ix.Request.Body = openapi.ExampleValue(ep.RequestBody)
	}

	// Response body and matching rules.
	if respSchema != nil {
		ix.Response.Body = openapi.ExampleValue(respSchema)
		buildMatchingRules(respSchema, "$.body", ix.MatchingRules)
	}

	return ix
}

func interactionDescription(ep openapi.Endpoint, status int, multipleResponses bool) string {
	base := ep.Summary
	if base == "" {
		base = ep.Method + " " + ep.Path
	}
	if multipleResponses {
		return fmt.Sprintf("%s (%d)", base, status)
	}
	return base
}

// substitutePath replaces {param} placeholders with example values.
func substitutePath(path string, params []openapi.Parameter) string {
	for _, p := range params {
		if p.In != "path" {
			continue
		}
		placeholder := "{" + p.Name + "}"
		val := pathParamValue(p)
		path = strings.ReplaceAll(path, placeholder, val)
	}
	return path
}

// pathParamValue returns an example value string for a path parameter.
func pathParamValue(p openapi.Parameter) string {
	if p.Schema != nil && p.Schema.Example != nil {
		return fmt.Sprintf("%v", p.Schema.Example)
	}
	if p.Schema != nil && p.Schema.Type == "integer" {
		return "1"
	}
	return p.Name
}

// buildMatchingRules recursively creates matching rules for required fields.
func buildMatchingRules(schema *openapi.SchemaRef, prefix string, rules contract.MatchingRules) {
	if schema == nil {
		return
	}

	// Handle allOf by merging.
	if len(schema.AllOf) > 0 {
		for _, sub := range schema.AllOf {
			buildMatchingRules(sub, prefix, rules)
		}
		return
	}

	// Handle oneOf/anyOf by using first variant.
	if len(schema.OneOf) > 0 {
		buildMatchingRules(schema.OneOf[0], prefix, rules)
		return
	}
	if len(schema.AnyOf) > 0 {
		buildMatchingRules(schema.AnyOf[0], prefix, rules)
		return
	}

	if schema.Type == "object" {
		required := make(map[string]bool, len(schema.Required))
		for _, r := range schema.Required {
			required[r] = true
		}
		for name, prop := range schema.Properties {
			if required[name] {
				fieldPath := prefix + "." + name
				buildMatchingRules(prop, fieldPath, rules)
			}
		}
		return
	}

	if schema.Type == "array" {
		// We don't generate per-element matching rules for arrays.
		// The array as a whole gets a type match.
		rules[prefix] = matchingRule(schema)
		return
	}

	// Leaf field: add a matching rule.
	rules[prefix] = matchingRule(schema)
}

// matchingRule determines the appropriate matching rule for a schema.
func matchingRule(schema *openapi.SchemaRef) contract.MatchingRule {
	// Pattern -> regex match.
	if schema.Pattern != "" {
		return contract.MatchingRule{Match: "regex", Regex: schema.Pattern}
	}
	// Single-value enum -> exact match.
	if len(schema.Enum) == 1 {
		return contract.MatchingRule{Match: "exact"}
	}
	// Everything else -> type match.
	return contract.MatchingRule{Match: "type"}
}
