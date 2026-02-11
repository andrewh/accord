// Tests for OpenAPI spec parsing into Accord types.
package openapi

import (
	"testing"
)

func TestParseFile(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Title != "Petstore" {
		t.Errorf("expected title Petstore, got %q", spec.Title)
	}
	if spec.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", spec.Version)
	}
}

func TestParseFileEndpoints(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Petstore has: GET /pets, POST /pets, GET /pets/{petId}, GET /pets/{petId}/health
	if len(spec.Endpoints) != 4 {
		t.Fatalf("expected 4 endpoints, got %d", len(spec.Endpoints))
	}

	// Find GET /pets
	var listPets *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "GET" && spec.Endpoints[i].Path == "/pets" {
			listPets = &spec.Endpoints[i]
			break
		}
	}
	if listPets == nil {
		t.Fatal("expected GET /pets endpoint")
	}
	if listPets.Summary != "List all pets" {
		t.Errorf("expected summary 'List all pets', got %q", listPets.Summary)
	}
}

func TestParseFilePathParameters(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find GET /pets/{petId}
	var getPet *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "GET" && spec.Endpoints[i].Path == "/pets/{petId}" {
			getPet = &spec.Endpoints[i]
			break
		}
	}
	if getPet == nil {
		t.Fatal("expected GET /pets/{petId} endpoint")
	}
	if len(getPet.Parameters) == 0 {
		t.Fatal("expected at least one parameter")
	}
	param := getPet.Parameters[0]
	if param.Name != "petId" {
		t.Errorf("expected param name 'petId', got %q", param.Name)
	}
	if param.In != "path" {
		t.Errorf("expected param in 'path', got %q", param.In)
	}
	if param.Schema == nil {
		t.Fatal("expected param to have schema")
	}
	if param.Schema.Type != "integer" {
		t.Errorf("expected param type 'integer', got %q", param.Schema.Type)
	}
}

func TestParseFileRequestBody(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find POST /pets
	var createPet *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "POST" && spec.Endpoints[i].Path == "/pets" {
			createPet = &spec.Endpoints[i]
			break
		}
	}
	if createPet == nil {
		t.Fatal("expected POST /pets endpoint")
	}
	if createPet.RequestBody == nil {
		t.Fatal("expected POST /pets to have request body")
	}
	if createPet.RequestBody.Type != "object" {
		t.Errorf("expected request body type 'object', got %q", createPet.RequestBody.Type)
	}
	if len(createPet.RequestBody.Required) == 0 {
		t.Fatal("expected request body to have required fields")
	}
}

func TestParseFileResponses(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find GET /pets - should have 200 response
	var listPets *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "GET" && spec.Endpoints[i].Path == "/pets" {
			listPets = &spec.Endpoints[i]
			break
		}
	}
	if listPets == nil {
		t.Fatal("expected GET /pets endpoint")
	}
	resp, ok := listPets.Responses[200]
	if !ok {
		t.Fatal("expected 200 response")
	}
	if resp == nil {
		t.Fatal("expected response schema")
	}
	if resp.Type != "array" {
		t.Errorf("expected response type 'array', got %q", resp.Type)
	}
}

func TestParseFileAllOf(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find GET /pets/{petId}/health - uses allOf
	var getHealth *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "GET" && spec.Endpoints[i].Path == "/pets/{petId}/health" {
			getHealth = &spec.Endpoints[i]
			break
		}
	}
	if getHealth == nil {
		t.Fatal("expected GET /pets/{petId}/health endpoint")
	}
	resp, ok := getHealth.Responses[200]
	if !ok {
		t.Fatal("expected 200 response")
	}
	if len(resp.AllOf) == 0 {
		t.Fatal("expected allOf in health record schema")
	}
}

func TestParseFileMinimal(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.Title != "Minimal Service" {
		t.Errorf("expected title 'Minimal Service', got %q", spec.Title)
	}
	if len(spec.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(spec.Endpoints))
	}
	ep := spec.Endpoints[0]
	if ep.Method != "GET" || ep.Path != "/health" {
		t.Errorf("expected GET /health, got %s %s", ep.Method, ep.Path)
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseFileEnumAndPattern(t *testing.T) {
	spec, err := ParseFile("../../testdata/openapi/petstore.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find GET /pets/{petId} and check Pet schema's status field has enum
	var getPet *Endpoint
	for i := range spec.Endpoints {
		if spec.Endpoints[i].Method == "GET" && spec.Endpoints[i].Path == "/pets/{petId}" {
			getPet = &spec.Endpoints[i]
			break
		}
	}
	if getPet == nil {
		t.Fatal("expected GET /pets/{petId} endpoint")
	}
	resp := getPet.Responses[200]
	if resp == nil {
		t.Fatal("expected 200 response schema")
	}

	statusProp, ok := resp.Properties["status"]
	if !ok {
		t.Fatal("expected 'status' property in Pet schema")
	}
	if len(statusProp.Enum) != 3 {
		t.Errorf("expected 3 enum values for status, got %d", len(statusProp.Enum))
	}

	emailProp, ok := resp.Properties["email"]
	if !ok {
		t.Fatal("expected 'email' property in Pet schema")
	}
	if emailProp.Pattern != "^[^@]+@[^@]+$" {
		t.Errorf("expected email pattern, got %q", emailProp.Pattern)
	}
	if emailProp.Format != "email" {
		t.Errorf("expected email format, got %q", emailProp.Format)
	}
}
