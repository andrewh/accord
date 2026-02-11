// Tests for contract generation orchestration.
package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func TestGenerateFromFile(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer: "order-service",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	c := contracts[0]
	if c.Contract.Consumer.Name != "order-service" {
		t.Errorf("expected consumer 'order-service', got %q", c.Contract.Consumer.Name)
	}
	if c.Contract.Provider.Name != "petstore" {
		t.Errorf("expected provider 'petstore', got %q", c.Contract.Provider.Name)
	}
	if c.Contract.Accord != "0.1" {
		t.Errorf("expected accord '0.1', got %q", c.Contract.Accord)
	}
	if len(c.Contract.Interactions) == 0 {
		t.Error("expected at least one interaction")
	}
}

func TestGenerateDefaultConsumer(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/minimal.yaml", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contracts[0].Contract.Consumer.Name != "my-service" {
		t.Errorf("expected default consumer 'my-service', got %q", contracts[0].Contract.Consumer.Name)
	}
}

func TestGenerateEndpointFilter(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer:  "test",
		Endpoints: "/pets",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := contracts[0]
	// Should match GET /pets and POST /pets but not /pets/{petId} or /pets/{petId}/health.
	for _, ix := range c.Contract.Interactions {
		desc := strings.ToLower(ix.Description)
		if strings.Contains(desc, "health") {
			t.Error("expected /pets/{petId}/health to be filtered out")
		}
		if strings.Contains(desc, "by id") || strings.Contains(desc, "pet by") {
			t.Error("expected /pets/{petId} to be filtered out")
		}
	}
}

func TestGenerateEndpointFilterWildcard(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer:  "test",
		Endpoints: "/pets/*",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := contracts[0]
	if len(c.Contract.Interactions) == 0 {
		t.Error("expected interactions matching /pets/*")
	}
	// Should include /pets/{petId} but not /pets or /pets/{petId}/health.
	for _, ix := range c.Contract.Interactions {
		desc := strings.ToLower(ix.Description)
		if strings.Contains(desc, "list") {
			t.Error("expected GET /pets (list) to be filtered out by /pets/*")
		}
	}
}

func TestGenerateFilename(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer: "order-service",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contracts[0].Filename != "order-service--petstore.yaml" {
		t.Errorf("expected filename 'order-service--petstore.yaml', got %q", contracts[0].Filename)
	}
}

func TestGenerateProviderNameSanitised(t *testing.T) {
	// The minimal service has title "Minimal Service" which should become "minimal-service".
	contracts, err := FromFile("../../testdata/openapi/minimal.yaml", Options{
		Consumer: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contracts[0].Contract.Provider.Name != "minimal-service" {
		t.Errorf("expected provider 'minimal-service', got %q", contracts[0].Contract.Provider.Name)
	}
	if contracts[0].Filename != "test--minimal-service.yaml" {
		t.Errorf("expected filename 'test--minimal-service.yaml', got %q", contracts[0].Filename)
	}
}

func TestGenerateOutputIsValidYAML(t *testing.T) {
	contracts, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yaml := contracts[0].YAML
	if len(yaml) == 0 {
		t.Fatal("expected non-empty YAML output")
	}

	// Parse it back to verify it's valid contract YAML.
	result, err := contract.Parse(yaml)
	if err != nil {
		t.Fatalf("generated YAML is not valid contract format: %v\n%s", err, yaml)
	}
	if result.Contract.Accord != "0.1" {
		t.Errorf("expected accord '0.1' in parsed output, got %q", result.Contract.Accord)
	}
}

func TestGenerateWriteFiles(t *testing.T) {
	dir := t.TempDir()
	contracts, err := FromFile("../../testdata/openapi/minimal.yaml", Options{
		Consumer:  "test",
		OutputDir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := WriteFiles(contracts); err != nil {
		t.Fatalf("unexpected error writing files: %v", err)
	}

	path := filepath.Join(dir, "test--minimal-service.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestGenerateFileNotFound(t *testing.T) {
	_, err := FromFile("nonexistent.yaml", Options{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGenerateBadEndpointPattern(t *testing.T) {
	_, err := FromFile("../../testdata/openapi/petstore.yaml", Options{
		Consumer:  "test",
		Endpoints: "[invalid",
	})
	if err == nil {
		t.Fatal("expected error for malformed glob pattern")
	}
	if !strings.Contains(err.Error(), "invalid endpoint filter pattern") {
		t.Errorf("expected descriptive error, got: %v", err)
	}
}
