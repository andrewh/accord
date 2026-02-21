// Pact JSON types, parsing, and version detection.
package convert

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PactFile represents a parsed Pact contract file.
type PactFile struct {
	Consumer     PactParty         `json:"consumer"`
	Provider     PactParty         `json:"provider"`
	Interactions []PactInteraction `json:"interactions"`
	Messages     []json.RawMessage `json:"messages,omitempty"`
	Metadata     PactMetadata      `json:"metadata"`
}

// PactParty identifies a consumer or provider service.
type PactParty struct {
	Name string `json:"name"`
}

// PactMetadata holds spec version and other metadata.
type PactMetadata struct {
	PactSpecification struct {
		Version string `json:"version"`
	} `json:"pactSpecification"`
}

// PactInteraction represents a single Pact interaction.
type PactInteraction struct {
	Description    string            `json:"description"`
	ProviderStates []json.RawMessage `json:"providerStates,omitempty"`
	ProviderState  string            `json:"providerState,omitempty"`
	Request        PactRequest       `json:"request"`
	Response       PactResponse      `json:"response"`
	MatchingRules  json.RawMessage   `json:"matchingRules,omitempty"`
}

// PactRequest describes a Pact HTTP request.
type PactRequest struct {
	Method     string              `json:"method"`
	Path       string              `json:"path"`
	Headers    map[string]string   `json:"headers,omitempty"`
	QueryV2    string              // populated by custom unmarshal for v2 string queries
	QueryV3    map[string][]string // populated by custom unmarshal for v3 map queries
	Body       any                 `json:"body,omitempty"`
	Generators json.RawMessage     `json:"generators,omitempty"`
}

// UnmarshalJSON handles the polymorphic query field (string in v2, map in v3).
func (r *PactRequest) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid recursive unmarshal.
	type Alias PactRequest
	aux := &struct {
		Query json.RawMessage `json:"query,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if len(aux.Query) == 0 {
		return nil
	}

	// Try string first (v2), then map (v3).
	var qs string
	if err := json.Unmarshal(aux.Query, &qs); err == nil {
		r.QueryV2 = qs
		return nil
	}

	var qm map[string][]string
	if err := json.Unmarshal(aux.Query, &qm); err == nil {
		r.QueryV3 = qm
		return nil
	}

	return fmt.Errorf("unsupported query format")
}

// PactResponse describes a Pact HTTP response.
type PactResponse struct {
	Status        int               `json:"status"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          any               `json:"body,omitempty"`
	MatchingRules json.RawMessage   `json:"matchingRules,omitempty"`
	Generators    json.RawMessage   `json:"generators,omitempty"`
}

// parsePactFile parses raw JSON into a PactFile.
func parsePactFile(data []byte) (*PactFile, error) {
	var pf PactFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing Pact JSON: %w", err)
	}
	return &pf, nil
}

// detectVersion determines the Pact specification version (2 or 3).
func detectVersion(pf *PactFile) int {
	// Check metadata first.
	v := pf.Metadata.PactSpecification.Version
	if strings.HasPrefix(v, "3") {
		return 3
	}
	if strings.HasPrefix(v, "2") {
		return 2
	}

	// Fallback: inspect matching rules structure on first interaction response.
	for _, ix := range pf.Interactions {
		if len(ix.Response.MatchingRules) == 0 {
			continue
		}
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(ix.Response.MatchingRules, &keys); err != nil {
			continue
		}
		for k := range keys {
			// V3 uses category keys like "body", "header", "status".
			if k == "body" || k == "header" || k == "headers" || k == "status" {
				return 3
			}
			// V2 uses $. paths.
			if strings.HasPrefix(k, "$.") {
				return 2
			}
		}
	}

	return 2
}
