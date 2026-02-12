// Unit tests for Pact-to-Accord conversion orchestration.
package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func TestFromFileV2Basic(t *testing.T) {
	outputs, warnings, err := FromFile("../../testdata/pact/v2_basic.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(outputs) != 1 {
		t.Fatalf("outputs: got %d, want 1", len(outputs))
	}

	out := outputs[0]
	if out.Contract.Consumer.Name != "web-app" {
		t.Errorf("consumer: got %q", out.Contract.Consumer.Name)
	}
	if out.Contract.Provider.Name != "user-api" {
		t.Errorf("provider: got %q", out.Contract.Provider.Name)
	}
	if out.Contract.Accord != "0.1" {
		t.Errorf("accord version: got %q", out.Contract.Accord)
	}
	if len(out.Contract.Interactions) != 1 {
		t.Fatalf("interactions: got %d, want 1", len(out.Contract.Interactions))
	}

	ix := out.Contract.Interactions[0]
	if ix.Description != "a request for a user" {
		t.Errorf("description: got %q", ix.Description)
	}
	if ix.Request.Method != "GET" {
		t.Errorf("method: got %q", ix.Request.Method)
	}
	if ix.Request.Path != "/users/1" {
		t.Errorf("path: got %q", ix.Request.Path)
	}
	if ix.Response.Status != 200 {
		t.Errorf("status: got %d", ix.Response.Status)
	}

	// Headers should be carried over.
	if ix.Request.Headers["Accept"] != "application/json" {
		t.Errorf("request Accept header: got %q", ix.Request.Headers["Accept"])
	}
	if ix.Response.Headers["Content-Type"] != "application/json" {
		t.Errorf("response Content-Type header: got %q", ix.Response.Headers["Content-Type"])
	}

	// No matching rules in v2_basic, so matching_rules should be empty.
	if len(ix.MatchingRules) != 0 {
		t.Errorf("expected no matching rules, got %d", len(ix.MatchingRules))
	}

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestFromFileV2WithMatching(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v2_with_matching.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	ix := outputs[0].Contract.Interactions[0]

	// Should have matching rules translated from v2 format.
	if len(ix.MatchingRules) != 4 {
		t.Fatalf("matching_rules: got %d, want 4", len(ix.MatchingRules))
	}
	if r, ok := ix.MatchingRules["$.body.id"]; !ok || r.Match != "type" {
		t.Errorf("$.body.id: got %v", ix.MatchingRules["$.body.id"])
	}
	if r, ok := ix.MatchingRules["$.body.email"]; !ok || r.Regex != "^[^@]+@[^@]+$" {
		t.Errorf("$.body.email: got %v", ix.MatchingRules["$.body.email"])
	}
	if r, ok := ix.MatchingRules["$.headers.Content-Type"]; !ok || r.Match != "regex" {
		t.Errorf("$.headers.Content-Type: got %v", ix.MatchingRules["$.headers.Content-Type"])
	}

	// V2 query string should be converted to map.
	if ix.Request.Query["page"] != "1" || ix.Request.Query["size"] != "10" {
		t.Errorf("query: got %v", ix.Request.Query)
	}
}

func TestFromFileV3Basic(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v3_basic.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	ix := outputs[0].Contract.Interactions[0]

	// V3 query map should be converted (first value per key).
	if ix.Request.Query["include"] != "email" {
		t.Errorf("query: got %v", ix.Request.Query)
	}
}

func TestFromFileV3WithMatching(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v3_with_matching.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	ix := outputs[0].Contract.Interactions[0]

	if len(ix.MatchingRules) != 4 {
		t.Fatalf("matching_rules: got %d, want 4", len(ix.MatchingRules))
	}
	if r, ok := ix.MatchingRules["$.body.id"]; !ok || r.Match != "type" {
		t.Errorf("$.body.id: got %v", ix.MatchingRules["$.body.id"])
	}
	if r, ok := ix.MatchingRules["$.headers.Content-Type"]; !ok || r.Match != "regex" {
		t.Errorf("$.headers.Content-Type: got %v", ix.MatchingRules["$.headers.Content-Type"])
	}
}

func TestFromFileV3Unsupported(t *testing.T) {
	outputs, warnings, err := FromFile("../../testdata/pact/v3_unsupported.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	if len(outputs) != 1 {
		t.Fatalf("outputs: got %d, want 1", len(outputs))
	}

	// Should have warnings for messages, providerStates, generators.
	warnMessages := warningStrings(warnings)

	hasProviderStates := false
	hasGenerators := false
	hasMessages := false
	for _, w := range warnMessages {
		if strings.Contains(w, "providerState") {
			hasProviderStates = true
		}
		if strings.Contains(w, "generator") {
			hasGenerators = true
		}
		if strings.Contains(w, "message") {
			hasMessages = true
		}
	}

	if !hasProviderStates {
		t.Errorf("expected providerStates warning, got: %v", warnMessages)
	}
	if !hasGenerators {
		t.Errorf("expected generators warning, got: %v", warnMessages)
	}
	if !hasMessages {
		t.Errorf("expected messages warning, got: %v", warnMessages)
	}
}

func TestFromFileFilename(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v2_basic.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	if outputs[0].Filename != "web-app--user-api.yaml" {
		t.Errorf("filename: got %q, want %q", outputs[0].Filename, "web-app--user-api.yaml")
	}
}

func TestFromFileOutputDir(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v2_basic.json", Options{OutputDir: "/tmp/out"})
	if err != nil {
		t.Fatal(err)
	}

	if outputs[0].OutputDir != "/tmp/out" {
		t.Errorf("output dir: got %q", outputs[0].OutputDir)
	}
}

func TestFromFileYAMLIsValid(t *testing.T) {
	outputs, _, err := FromFile("../../testdata/pact/v2_with_matching.json", Options{})
	if err != nil {
		t.Fatal(err)
	}

	// The YAML should parse as a valid Accord contract.
	result, err := contract.Parse(outputs[0].YAML)
	if err != nil {
		t.Fatalf("generated YAML is not valid Accord: %v", err)
	}
	if result.Contract.Accord != "0.1" {
		t.Errorf("parsed accord version: got %q", result.Contract.Accord)
	}
}

func TestFromFileMissingFile(t *testing.T) {
	_, _, err := FromFile("nonexistent.json", Options{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFromFileBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := FromFile(path, Options{})
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestWriteFiles(t *testing.T) {
	dir := t.TempDir()
	outputs := []Output{{
		YAML:      []byte("accord: \"0.1\"\n"),
		Filename:  "test.yaml",
		OutputDir: dir,
	}}

	if err := WriteFiles(outputs); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "accord: \"0.1\"\n" {
		t.Errorf("file content: got %q", string(data))
	}
}

func TestV2QueryStringConversion(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  map[string]string
	}{
		{"simple", "page=1&size=10", map[string]string{"page": "1", "size": "10"}},
		{"single", "key=value", map[string]string{"key": "value"}},
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := convertV2Query(tt.query)
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("got %v, want empty", got)
				}
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestV3QueryMapConversion(t *testing.T) {
	input := map[string][]string{
		"page":   {"1"},
		"tags":   {"a", "b", "c"},
	}

	got, warnings := convertV3Query(input)
	if got["page"] != "1" {
		t.Errorf("page: got %q, want %q", got["page"], "1")
	}
	if got["tags"] != "a" {
		t.Errorf("tags: got %q, want %q (first value)", got["tags"], "a")
	}

	// Should warn about multi-value parameter.
	hasMultiWarn := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "tags") {
			hasMultiWarn = true
		}
	}
	if !hasMultiWarn {
		t.Errorf("expected warning for multi-value query param 'tags'")
	}
}

func warningStrings(ws []Warning) []string {
	var ss []string
	for _, w := range ws {
		ss = append(ss, w.Message)
	}
	return ss
}
