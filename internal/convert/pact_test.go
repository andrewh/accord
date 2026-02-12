// Unit tests for Pact JSON parsing and version detection.
package convert

import (
	"os"
	"testing"
)

func TestParsePactFileV2Basic(t *testing.T) {
	data, err := os.ReadFile("../../testdata/pact/v2_basic.json")
	if err != nil {
		t.Fatal(err)
	}

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}

	if pf.Consumer.Name != "web-app" {
		t.Errorf("consumer: got %q, want %q", pf.Consumer.Name, "web-app")
	}
	if pf.Provider.Name != "user-api" {
		t.Errorf("provider: got %q, want %q", pf.Provider.Name, "user-api")
	}
	if len(pf.Interactions) != 1 {
		t.Fatalf("interactions: got %d, want 1", len(pf.Interactions))
	}

	ix := pf.Interactions[0]
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
}

func TestParsePactFileV3WithQuery(t *testing.T) {
	data, err := os.ReadFile("../../testdata/pact/v3_basic.json")
	if err != nil {
		t.Fatal(err)
	}

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}

	ix := pf.Interactions[0]
	if ix.Request.QueryV3 == nil {
		t.Fatal("expected v3 query map to be populated")
	}
	if ix.Request.QueryV3["include"] == nil {
		t.Error("expected 'include' key in v3 query")
	}
}

func TestParsePactFileV3Unsupported(t *testing.T) {
	data, err := os.ReadFile("../../testdata/pact/v3_unsupported.json")
	if err != nil {
		t.Fatal(err)
	}

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(pf.Messages) == 0 {
		t.Error("expected messages to be populated")
	}

	ix := pf.Interactions[0]
	if len(ix.ProviderStates) == 0 {
		t.Error("expected providerStates to be populated")
	}
	if ix.Response.Generators == nil {
		t.Error("expected generators to be populated")
	}
}

func TestDetectVersionFromMetadata(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    int
	}{
		{"v2 from metadata", "../../testdata/pact/v2_basic.json", 2},
		{"v2 with matching", "../../testdata/pact/v2_with_matching.json", 2},
		{"v3 from metadata", "../../testdata/pact/v3_basic.json", 3},
		{"v3 with matching", "../../testdata/pact/v3_with_matching.json", 3},
		{"v3 unsupported", "../../testdata/pact/v3_unsupported.json", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}
			pf, err := parsePactFile(data)
			if err != nil {
				t.Fatal(err)
			}
			got := detectVersion(pf)
			if got != tt.want {
				t.Errorf("detectVersion: got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDetectVersionFallbackV3(t *testing.T) {
	// No metadata, but v3-style categorised matching rules on response.
	data := []byte(`{
		"consumer": {"name": "a"},
		"provider": {"name": "b"},
		"interactions": [{
			"description": "test",
			"request": {"method": "GET", "path": "/"},
			"response": {
				"status": 200,
				"matchingRules": {
					"body": {"$.id": {"matchers": [{"match": "type"}]}}
				}
			}
		}],
		"metadata": {}
	}`)

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}
	got := detectVersion(pf)
	if got != 3 {
		t.Errorf("detectVersion fallback: got %d, want 3", got)
	}
}

func TestDetectVersionFallbackV2(t *testing.T) {
	// No metadata, but v2-style $. paths on response matching rules.
	data := []byte(`{
		"consumer": {"name": "a"},
		"provider": {"name": "b"},
		"interactions": [{
			"description": "test",
			"request": {"method": "GET", "path": "/"},
			"response": {
				"status": 200,
				"matchingRules": {
					"$.body.id": {"match": "type"}
				}
			}
		}],
		"metadata": {}
	}`)

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}
	got := detectVersion(pf)
	if got != 2 {
		t.Errorf("detectVersion fallback: got %d, want 2", got)
	}
}

func TestDetectVersionDefaultsToV2(t *testing.T) {
	data := []byte(`{
		"consumer": {"name": "a"},
		"provider": {"name": "b"},
		"interactions": [{
			"description": "test",
			"request": {"method": "GET", "path": "/"},
			"response": {"status": 200}
		}],
		"metadata": {}
	}`)

	pf, err := parsePactFile(data)
	if err != nil {
		t.Fatal(err)
	}
	got := detectVersion(pf)
	if got != 2 {
		t.Errorf("detectVersion default: got %d, want 2", got)
	}
}
