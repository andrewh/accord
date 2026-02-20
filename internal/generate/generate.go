// Orchestrate contract generation from OpenAPI specs.
package generate

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/openapi"
	"gopkg.in/yaml.v3"
)

// Options configures contract generation.
type Options struct {
	Consumer  string // Consumer service name (default: "my-service").
	Endpoints string // Glob pattern to filter endpoint paths (empty = all).
	OutputDir string // Directory for output files (used by WriteFiles).
}

// Output holds a generated contract with its serialised YAML and filename.
type Output struct {
	Contract  *contract.Contract
	YAML      []byte
	Filename  string
	OutputDir string
}

// FromFile generates contracts from an OpenAPI spec file.
func FromFile(specPath string, opts Options) ([]Output, error) {
	spec, err := openapi.ParseFile(specPath)
	if err != nil {
		return nil, err
	}
	return fromSpec(spec, opts)
}

func fromSpec(spec *openapi.Spec, opts Options) ([]Output, error) {
	consumer := opts.Consumer
	if consumer == "" {
		consumer = "my-service"
	}
	provider := sanitiseName(spec.Title)

	endpoints, err := filterEndpoints(spec.Endpoints, opts.Endpoints)
	if err != nil {
		return nil, err
	}

	var interactions []contract.Interaction
	for _, ep := range endpoints {
		interactions = append(interactions, BuildInteractions(ep)...)
	}

	if len(interactions) == 0 {
		return nil, nil
	}

	c := &contract.Contract{
		Accord:       "0.1",
		Consumer:     contract.Party{Name: consumer},
		Provider:     contract.Party{Name: provider},
		Interactions: interactions,
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshalling contract: %w", err)
	}

	filename := consumer + "--" + provider + ".yaml"

	return []Output{{
		Contract:  c,
		YAML:      data,
		Filename:  filename,
		OutputDir: opts.OutputDir,
	}}, nil
}

// WriteFiles writes generated contracts to disk.
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

// filterEndpoints returns endpoints matching the given glob pattern.
// An empty pattern matches all endpoints.
func filterEndpoints(endpoints []openapi.Endpoint, pattern string) ([]openapi.Endpoint, error) {
	if pattern == "" {
		return endpoints, nil
	}
	var filtered []openapi.Endpoint
	for _, ep := range endpoints {
		matched, err := path.Match(pattern, ep.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint filter pattern %q: %w", pattern, err)
		}
		if matched {
			filtered = append(filtered, ep)
		}
	}
	return filtered, nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// sanitiseName converts a spec title to a lowercase, hyphenated name.
func sanitiseName(title string) string {
	name := strings.ToLower(title)
	name = nonAlphanumeric.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}
