// Tests that parse the testdata fixture files.
package contract

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func TestParseFixtureValidUserService(t *testing.T) {
	result, err := ParseFile(filepath.Join(testdataDir(), "valid", "user_service.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := result.Contract
	if c.Accord != "0.1" {
		t.Errorf("Accord = %q, want %q", c.Accord, "0.1")
	}
	if len(c.Interactions) != 1 {
		t.Fatalf("len(Interactions) = %d, want 1", len(c.Interactions))
	}
	if c.Interactions[0].Description != "get a user by ID" {
		t.Errorf("Description = %q, want %q", c.Interactions[0].Description, "get a user by ID")
	}
	if len(c.Interactions[0].MatchingRules) != 3 {
		t.Errorf("len(MatchingRules) = %d, want 3", len(c.Interactions[0].MatchingRules))
	}
}

func TestParseFixtureValidMinimal(t *testing.T) {
	result, err := ParseFile(filepath.Join(testdataDir(), "valid", "minimal.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := result.Contract
	if c.Consumer.Name != "client-a" {
		t.Errorf("Consumer.Name = %q, want %q", c.Consumer.Name, "client-a")
	}
	if c.Provider.Name != "service-b" {
		t.Errorf("Provider.Name = %q, want %q", c.Provider.Name, "service-b")
	}
}
