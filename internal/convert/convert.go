// Orchestrate conversion from Pact contract files to Accord YAML.
package convert

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/andrewh/accord/internal/contract"
	"gopkg.in/yaml.v3"
)

// Warning represents a non-fatal issue found during conversion.
type Warning struct {
	File    string
	Message string
}

// Options configures the Pact-to-Accord conversion.
type Options struct {
	OutputDir string
}

// Output holds a converted contract with its serialised YAML and filename.
type Output struct {
	Contract  *contract.Contract
	YAML      []byte
	Filename  string
	OutputDir string
}

// FromFile reads a Pact JSON file and converts it to Accord format.
func FromFile(path string, opts Options) ([]Output, []Warning, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading Pact file: %w", err)
	}

	pf, err := parsePactFile(data)
	if err != nil {
		return nil, nil, err
	}

	version := detectVersion(pf)
	var warnings []Warning

	// Warn about unsupported message interactions.
	for _, raw := range pf.Messages {
		desc := messageDescription(raw)
		warnings = append(warnings, Warning{
			File:    path,
			Message: fmt.Sprintf("message %q: message interactions are not supported and were skipped", desc),
		})
	}

	var interactions []contract.Interaction
	for _, ix := range pf.Interactions {
		ci, ws := convertInteraction(ix, version, path)
		warnings = append(warnings, ws...)
		interactions = append(interactions, ci)
	}

	c := &contract.Contract{
		Accord:       "0.1",
		Consumer:     contract.Party{Name: pf.Consumer.Name},
		Provider:     contract.Party{Name: pf.Provider.Name},
		Interactions: interactions,
	}

	yamlData, err := yaml.Marshal(c)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling contract: %w", err)
	}

	filename := sanitiseName(pf.Consumer.Name) + "--" + sanitiseName(pf.Provider.Name) + ".yaml"

	outputs := []Output{{
		Contract:  c,
		YAML:      yamlData,
		Filename:  filename,
		OutputDir: opts.OutputDir,
	}}

	warnings = append(warnings, summariseSkipped(warnings, path)...)

	return outputs, warnings, nil
}

// WriteFiles writes converted contracts to disk.
func WriteFiles(outputs []Output) error {
	for _, out := range outputs {
		dir := out.OutputDir
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, out.Filename)
		if err := os.WriteFile(path, out.YAML, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return nil
}

// convertInteraction converts a single Pact interaction to Accord format.
func convertInteraction(ix PactInteraction, version int, file string) (contract.Interaction, []Warning) {
	var warnings []Warning

	// Warn about provider states.
	if len(ix.ProviderStates) > 0 || ix.ProviderState != "" {
		warnings = append(warnings, Warning{
			File:    file,
			Message: fmt.Sprintf("interaction %q: providerStates are not supported and were skipped", ix.Description),
		})
	}

	// Warn about generators.
	if len(ix.Response.Generators) > 0 || len(ix.Request.Generators) > 0 {
		warnings = append(warnings, Warning{
			File:    file,
			Message: fmt.Sprintf("interaction %q: generators are not supported and were skipped", ix.Description),
		})
	}

	// Convert query parameters.
	query, qws := convertQuery(ix.Request, version, file)
	warnings = append(warnings, qws...)

	// Convert matching rules from the response.
	var rules contract.MatchingRules
	raw := matchingRulesRaw(ix, version)
	if len(raw) > 0 {
		var rws []Warning
		if version >= 3 {
			rules, rws = convertMatchingRulesV3(raw)
		} else {
			rules, rws = convertMatchingRulesV2(raw)
		}
		for i := range rws {
			rws[i].File = file
		}
		warnings = append(warnings, rws...)
	}

	ci := contract.Interaction{
		Description:   ix.Description,
		Request:       buildRequest(ix.Request, query),
		Response:      buildResponse(ix.Response),
		MatchingRules: rules,
	}

	return ci, warnings
}

// matchingRulesRaw returns the raw matching rules JSON for the interaction.
// V2 can have rules at interaction or response level; v3 always at response level.
func matchingRulesRaw(ix PactInteraction, version int) json.RawMessage {
	if len(ix.Response.MatchingRules) > 0 {
		return ix.Response.MatchingRules
	}
	if version < 3 && len(ix.MatchingRules) > 0 {
		return ix.MatchingRules
	}
	return nil
}

// buildRequest creates an Accord Request from a Pact request.
func buildRequest(pr PactRequest, query map[string]string) contract.Request {
	r := contract.Request{
		Method:  pr.Method,
		Path:    pr.Path,
		Headers: pr.Headers,
		Query:   query,
		Body:    pr.Body,
	}
	return r
}

// buildResponse creates an Accord Response from a Pact response.
func buildResponse(pr PactResponse) contract.Response {
	return contract.Response{
		Status:  pr.Status,
		Headers: pr.Headers,
		Body:    pr.Body,
	}
}

// convertQuery converts Pact query parameters to Accord map format.
func convertQuery(req PactRequest, version int, file string) (map[string]string, []Warning) {
	if version >= 3 && req.QueryV3 != nil {
		q, ws := convertV3Query(req.QueryV3)
		for i := range ws {
			ws[i].File = file
		}
		return q, ws
	}
	if req.QueryV2 != "" {
		q, ws := convertV2Query(req.QueryV2)
		for i := range ws {
			ws[i].File = file
		}
		return q, ws
	}
	return nil, nil
}

// convertV2Query parses a v2 query string into a map.
func convertV2Query(qs string) (map[string]string, []Warning) {
	if qs == "" {
		return nil, nil
	}

	values, err := url.ParseQuery(qs)
	if err != nil {
		return nil, []Warning{{Message: fmt.Sprintf("failed to parse query string %q: %v", qs, err)}}
	}

	result := make(map[string]string)
	var warnings []Warning
	for k, vs := range values {
		result[k] = vs[0]
		if len(vs) > 1 {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("query parameter %q has multiple values, using first only", k),
			})
		}
	}
	return result, warnings
}

// convertV3Query converts a v3 query map (values are arrays) to Accord flat map.
func convertV3Query(qm map[string][]string) (map[string]string, []Warning) {
	if len(qm) == 0 {
		return nil, nil
	}

	result := make(map[string]string)
	var warnings []Warning
	for k, vs := range qm {
		if len(vs) == 0 {
			continue
		}
		result[k] = vs[0]
		if len(vs) > 1 {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("query parameter %q has multiple values, using first only", k),
			})
		}
	}
	return result, warnings
}

// messageDescription extracts the description from a raw Pact message JSON.
func messageDescription(raw json.RawMessage) string {
	var m struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &m); err != nil || m.Description == "" {
		return "(unknown)"
	}
	return m.Description
}

// summariseSkipped appends a summary warning when multiple unsupported features were skipped.
func summariseSkipped(warnings []Warning, file string) []Warning {
	var providerStates, generators, messages int
	for _, w := range warnings {
		switch {
		case strings.Contains(w.Message, "providerStates are not supported"):
			providerStates++
		case strings.Contains(w.Message, "generators are not supported"):
			generators++
		case strings.Contains(w.Message, "message interactions are not supported"):
			messages++
		}
	}

	total := providerStates + generators + messages
	if total < 2 {
		return nil
	}

	var parts []string
	if providerStates > 0 {
		parts = append(parts, fmt.Sprintf("%d provider state(s)", providerStates))
	}
	if generators > 0 {
		parts = append(parts, fmt.Sprintf("%d generator(s)", generators))
	}
	if messages > 0 {
		parts = append(parts, fmt.Sprintf("%d message(s)", messages))
	}

	return []Warning{{
		File:    file,
		Message: fmt.Sprintf("summary: skipped unsupported features: %s", strings.Join(parts, ", ")),
	}}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// sanitiseName converts a service name to a lowercase, hyphenated string.
func sanitiseName(name string) string {
	n := strings.ToLower(name)
	n = nonAlphanumeric.ReplaceAllString(n, "-")
	n = strings.Trim(n, "-")
	return n
}
