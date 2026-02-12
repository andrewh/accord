// Contract types and YAML parsing for Accord contract files.
package contract

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Contract represents a complete Accord contract file.
type Contract struct {
	Accord       string        `yaml:"accord"`
	Consumer     Party         `yaml:"consumer"`
	Provider     Party         `yaml:"provider"`
	Interactions []Interaction `yaml:"interactions"`
}

// Party identifies a consumer or provider service.
type Party struct {
	Name string `yaml:"name"`
}

// Interaction defines a single request/response exchange.
type Interaction struct {
	Description   string        `yaml:"description"`
	Request       Request       `yaml:"request"`
	Response      Response      `yaml:"response"`
	MatchingRules MatchingRules `yaml:"matching_rules"`
	NFR           *NFR          `yaml:"nfr,omitempty"`
}

// NFRThreshold defines a not-to-exceed value with optional severity.
type NFRThreshold struct {
	Threshold int    `yaml:"threshold"`
	Severity  string `yaml:"severity,omitempty"`
}

// NFR holds non-functional requirement thresholds for an interaction.
type NFR struct {
	MaxResponseBytes     *NFRThreshold `yaml:"max_response_bytes,omitempty"`
	MaxTimeToFirstByteMs *NFRThreshold `yaml:"max_time_to_first_byte_ms,omitempty"`
	MaxRoundTripMs       *NFRThreshold `yaml:"max_round_trip_ms,omitempty"`
}

// Request describes the HTTP request to send.
type Request struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Query   map[string]string `yaml:"query"`
	Body    any               `yaml:"body"`
}

// Response describes the expected HTTP response.
type Response struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    any               `yaml:"body"`
}

// MatchingRules maps dot-notation paths to matching rules.
type MatchingRules map[string]MatchingRule

// MatchingRule defines how a response field should be compared.
type MatchingRule struct {
	Match    string   `yaml:"match"`
	Regex    string   `yaml:"regex,omitempty"`
	Min      *float64 `yaml:"min,omitempty"`
	Max      *float64 `yaml:"max,omitempty"`
	Values   []string `yaml:"values,omitempty"`
	Format   string   `yaml:"format,omitempty"`
	Includes string   `yaml:"includes,omitempty"`
}

// ParseResult holds both the structured contract and the raw YAML node tree.
type ParseResult struct {
	Contract *Contract
	Node     *yaml.Node
}

// ParseFile reads and parses a contract file from disk.
func ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading contract file: %w", err)
	}
	return Parse(data)
}

// Parse parses contract YAML from bytes.
func Parse(data []byte) (*ParseResult, error) {
	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing contract YAML: %w", err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing contract YAML nodes: %w", err)
	}

	return &ParseResult{Contract: &c, Node: &node}, nil
}
