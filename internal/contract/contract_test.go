// Tests for contract parsing and type marshalling.
package contract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidContract(t *testing.T) {
	data := []byte(`
accord: "0.1"
consumer:
  name: "order-service"
provider:
  name: "user-service"
interactions:
  - description: "get user"
    request:
      method: GET
      path: /users/1
      headers:
        Accept: application/json
      query:
        include: "email"
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        id: 1
        name: "Jane"
    matching_rules:
      "$.body.id":
        match: type
      "$.body.name":
        match: regex
        regex: "^[A-Z]"
`)

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := result.Contract
	if c.Accord != "0.1" {
		t.Errorf("Accord = %q, want %q", c.Accord, "0.1")
	}
	if c.Consumer.Name != "order-service" {
		t.Errorf("Consumer.Name = %q, want %q", c.Consumer.Name, "order-service")
	}
	if c.Provider.Name != "user-service" {
		t.Errorf("Provider.Name = %q, want %q", c.Provider.Name, "user-service")
	}
	if len(c.Interactions) != 1 {
		t.Fatalf("len(Interactions) = %d, want 1", len(c.Interactions))
	}

	ix := c.Interactions[0]
	if ix.Description != "get user" {
		t.Errorf("Description = %q, want %q", ix.Description, "get user")
	}
	if ix.Request.Method != "GET" {
		t.Errorf("Method = %q, want %q", ix.Request.Method, "GET")
	}
	if ix.Request.Path != "/users/1" {
		t.Errorf("Path = %q, want %q", ix.Request.Path, "/users/1")
	}
	if ix.Request.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header = %q, want %q", ix.Request.Headers["Accept"], "application/json")
	}
	if ix.Request.Query["include"] != "email" {
		t.Errorf("Query include = %q, want %q", ix.Request.Query["include"], "email")
	}
	if ix.Response.Status != 200 {
		t.Errorf("Status = %d, want 200", ix.Response.Status)
	}
	if ix.Response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header = %q, want %q", ix.Response.Headers["Content-Type"], "application/json")
	}

	body, ok := ix.Response.Body.(map[string]any)
	if !ok {
		t.Fatalf("Body type = %T, want map[string]any", ix.Response.Body)
	}
	if body["name"] != "Jane" {
		t.Errorf("body.name = %v, want %q", body["name"], "Jane")
	}

	if len(ix.MatchingRules) != 2 {
		t.Fatalf("len(MatchingRules) = %d, want 2", len(ix.MatchingRules))
	}
	if ix.MatchingRules["$.body.id"].Match != "type" {
		t.Errorf("$.body.id match = %q, want %q", ix.MatchingRules["$.body.id"].Match, "type")
	}
	nameRule := ix.MatchingRules["$.body.name"]
	if nameRule.Match != "regex" {
		t.Errorf("$.body.name match = %q, want %q", nameRule.Match, "regex")
	}
	if nameRule.Regex != "^[A-Z]" {
		t.Errorf("$.body.name regex = %q, want %q", nameRule.Regex, "^[A-Z]")
	}
}

func TestParseMinimalContract(t *testing.T) {
	data := []byte(`
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "health"
    request:
      method: GET
      path: /health
    response:
      status: 200
`)

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := result.Contract
	if c.Interactions[0].Request.Headers != nil {
		t.Errorf("expected nil headers, got %v", c.Interactions[0].Request.Headers)
	}
	if c.Interactions[0].MatchingRules != nil {
		t.Errorf("expected nil matching rules, got %v", c.Interactions[0].MatchingRules)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	data := []byte(`{{{not valid yaml`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseNodeTreePreserved(t *testing.T) {
	data := []byte(`
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "test"
    request:
      method: GET
      path: /test
    response:
      status: 200
`)

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Node == nil {
		t.Fatal("expected node tree, got nil")
	}
	// The root node should be a document node
	if result.Node.Kind != 1 { // yaml.DocumentNode
		t.Errorf("root node kind = %d, want DocumentNode (1)", result.Node.Kind)
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	data := []byte(`
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "test"
    request:
      method: GET
      path: /test
    response:
      status: 200
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Contract.Consumer.Name != "a" {
		t.Errorf("Consumer.Name = %q, want %q", result.Contract.Consumer.Name, "a")
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
