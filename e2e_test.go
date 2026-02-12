// End-to-end tests: build the accord binary and test both commands.
package accord_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "accord-e2e")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "accord")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/accord/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build accord: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestE2ELintValidFiles(t *testing.T) {
	cmd := exec.Command(binaryPath, "lint", "testdata/valid/user_service.yaml", "testdata/valid/minimal.yaml", "testdata/valid/array_matching.yaml", "testdata/valid/nfr_example.yaml", "testdata/valid/extended_matching.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}
	if len(output) != 0 {
		t.Errorf("expected no output for valid files, got: %s", output)
	}
}

func TestE2ELintInvalidFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "lint", "testdata/invalid/missing_fields.yaml")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid file")
	}

	out := string(output)
	expectedMessages := []string{
		"consumer.name is required",
		"provider.name is required",
		"description is required",
		"request.method is required",
		"response.status is required",
	}
	for _, msg := range expectedMessages {
		if !strings.Contains(out, msg) {
			t.Errorf("expected output to contain %q, got:\n%s", msg, out)
		}
	}
}

func TestE2ELintMissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "lint", "nonexistent.yaml")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for missing file")
	}
	if !strings.Contains(string(output), "nonexistent.yaml") {
		t.Errorf("expected error to mention filename, got: %s", output)
	}
}

func TestE2ELintNoArgs(t *testing.T) {
	cmd := exec.Command(binaryPath, "lint")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit with no arguments")
	}
}

func TestE2EVerifyPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"name":  "Jane Doe",
			"email": "jane@example.com",
		})
	}))
	defer server.Close()

	// Write a contract that matches the test server
	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "get user"
    request:
      method: GET
      path: /users/123
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        id: 1
        name: "Jane Doe"
        email: "jane@example.com"
    matching_rules:
      "$.body.id":
        match: type
      "$.body.name":
        match: type
      "$.body.email":
        match: regex
        regex: "^[^@]+@[^@]+$"
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output, got: %s", out)
	}
	if !strings.Contains(out, "All interactions passed") {
		t.Errorf("expected success message, got: %s", out)
	}
}

func TestE2EVerifyFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "get user"
    request:
      method: GET
      path: /users/123
    response:
      status: 200
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for verification failure")
	}

	out := string(output)
	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL in output, got: %s", out)
	}
}

func TestE2EVerifyMissingProviderURL(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "testdata/valid/minimal.yaml")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit without --provider-url")
	}
}

func TestE2EVerifyArrayWildcard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"users": []any{
				map[string]any{
					"name":  "Xavier Jones",
					"email": "xavier@test.com",
					"roles": []any{"admin", "user"},
				},
				map[string]any{
					"name":  "Yolanda Park",
					"email": "yolanda@test.com",
					"roles": []any{"user"},
				},
			},
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "list users with wildcard matching"
    request:
      method: GET
      path: /users
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        users:
          - name: "Jane Doe"
            email: "jane@example.com"
            roles:
              - "admin"
              - "user"
          - name: "Bob Smith"
            email: "bob@example.com"
            roles:
              - "user"
    matching_rules:
      "$.body.users[*].name":
        match: type
      "$.body.users[*].email":
        match: regex
        regex: "^[^@]+@[^@]+$"
      "$.body.users[0].roles[0]":
        match: exact
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output, got: %s", out)
	}
}

func TestE2EVersion(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
	if !strings.HasPrefix(string(output), "accord ") {
		t.Errorf("expected output to start with 'accord ', got: %s", output)
	}
}

func TestE2EVerifyNFRPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "small response"
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 1000
      max_round_trip_ms:
        threshold: 5000
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output, got: %s", out)
	}
}

func TestE2EVerifyNFRWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"data": "this response body is intentionally larger than 10 bytes"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "oversized warning"
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 10
        severity: warning
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 (warning only), got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "WARN") {
		t.Errorf("expected WARN in output, got: %s", out)
	}
	if !strings.Contains(out, "[warning]") {
		t.Errorf("expected [warning] prefix in output, got: %s", out)
	}
}

func TestE2EVerifyNFRError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"data": "this response body is intentionally larger than 10 bytes"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "oversized error"
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 10
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for NFR error")
	}

	out := string(output)
	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL in output, got: %s", out)
	}
	if !strings.Contains(out, "threshold exceeded") {
		t.Errorf("expected threshold exceeded message, got: %s", out)
	}
}

func TestE2ELintNFRInvalid(t *testing.T) {
	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "bad nfr"
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 0
        severity: fatal
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "lint", contractPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid NFR")
	}

	out := string(output)
	if !strings.Contains(out, "threshold must be > 0") {
		t.Errorf("expected threshold error, got: %s", out)
	}
	if !strings.Contains(out, "invalid severity") {
		t.Errorf("expected severity error, got: %s", out)
	}
}

func TestE2EVerifyNewMatchTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"count":      42,
			"score":      7.5,
			"message":    "hello world",
			"created_at": "2024-06-15T14:00:00Z",
			"status":     "active",
			"id":         999,
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "contract.yaml")
	contractYAML := `
accord: "0.1"
consumer:
  name: "test-consumer"
provider:
  name: "test-provider"
interactions:
  - description: "verify new match types"
    request:
      method: GET
      path: /data
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        count: 10
        score: 5.0
        message: "placeholder"
        created_at: "2024-01-01T00:00:00Z"
        status: "pending"
        id: 1
    matching_rules:
      "$.body.count":
        match: min
        min: 1
      "$.body.score":
        match: max
        max: 100
      "$.body.message":
        match: includes
        includes: "world"
      "$.body.created_at":
        match: datetime
      "$.body.status":
        match: enum
        values:
          - "active"
          - "inactive"
          - "pending"
      "$.body.id":
        match: not_null
`
	if err := os.WriteFile(contractPath, []byte(contractYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "verify", "--provider-url", server.URL, contractPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output, got: %s", out)
	}
}

func TestE2EGenerateDryRun(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate", "--dry-run", "testdata/openapi/petstore.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "accord:") {
		t.Errorf("expected 'accord:' in output, got: %s", out)
	}
	if !strings.Contains(out, "consumer:") {
		t.Errorf("expected 'consumer:' in output, got: %s", out)
	}
	if !strings.Contains(out, "provider:") {
		t.Errorf("expected 'provider:' in output, got: %s", out)
	}
	if !strings.Contains(out, "interactions:") {
		t.Errorf("expected 'interactions:' in output, got: %s", out)
	}
}

func TestE2EGenerateWithConsumer(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate", "--dry-run", "--consumer", "order-service", "testdata/openapi/petstore.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "order-service") {
		t.Errorf("expected consumer name in output, got: %s", out)
	}
	if !strings.Contains(out, "order-service--petstore.yaml") {
		t.Errorf("expected filename header in output, got: %s", out)
	}
}

func TestE2EGenerateWriteFile(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(binaryPath, "generate", "--output-dir", dir, "--consumer", "test-svc", "testdata/openapi/minimal.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	path := filepath.Join(dir, "test-svc--minimal-service.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output file")
	}
}

func TestE2EGenerateThenLint(t *testing.T) {
	// Generate a contract, then lint it to prove the output is valid.
	dir := t.TempDir()
	genCmd := exec.Command(binaryPath, "generate", "--output-dir", dir, "--consumer", "test-svc", "testdata/openapi/petstore.yaml")
	genOutput, err := genCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate failed: %v\noutput: %s", err, genOutput)
	}

	contractPath := filepath.Join(dir, "test-svc--petstore.yaml")
	lintCmd := exec.Command(binaryPath, "lint", contractPath)
	lintOutput, err := lintCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lint failed on generated contract: %v\noutput: %s", err, lintOutput)
	}
	if len(lintOutput) != 0 {
		t.Errorf("expected no lint output for generated contract, got: %s", lintOutput)
	}
}

func TestE2EGenerateEndpointFilter(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate", "--dry-run", "--endpoints", "/pets", "testdata/openapi/petstore.yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "List all pets") {
		t.Errorf("expected interaction for 'List all pets' (GET /pets), got: %s", out)
	}
	if !strings.Contains(out, "Create a pet") {
		t.Errorf("expected interaction for 'Create a pet' (POST /pets), got: %s", out)
	}
	if strings.Contains(out, "Get a pet by ID") {
		t.Errorf("expected 'Get a pet by ID' (GET /pets/{petId}) to be filtered out")
	}
	if strings.Contains(out, "Get pet health record") {
		t.Errorf("expected 'Get pet health record' to be filtered out")
	}
}

func TestE2EGenerateBadPattern(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate", "--dry-run", "--endpoints", "[invalid", "testdata/openapi/petstore.yaml")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for malformed glob pattern")
	}
	if !strings.Contains(string(output), "invalid endpoint filter pattern") {
		t.Errorf("expected descriptive error, got: %s", output)
	}
}

func TestE2EGenerateMissingSpec(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate", "nonexistent.yaml")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for missing spec")
	}
}

func TestE2EGenerateNoArgs(t *testing.T) {
	cmd := exec.Command(binaryPath, "generate")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit with no arguments")
	}
}

func TestE2EConvertDryRun(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "--dry-run", "testdata/pact/v2_basic.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "accord:") {
		t.Errorf("expected 'accord:' in output, got: %s", out)
	}
	if !strings.Contains(out, "web-app") {
		t.Errorf("expected consumer name in output, got: %s", out)
	}
	if !strings.Contains(out, "user-api") {
		t.Errorf("expected provider name in output, got: %s", out)
	}
	if !strings.Contains(out, "web-app--user-api.yaml") {
		t.Errorf("expected filename header in output, got: %s", out)
	}
}

func TestE2EConvertV3DryRun(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "--dry-run", "testdata/pact/v3_with_matching.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "$.body.id") {
		t.Errorf("expected body matching rule path in output, got: %s", out)
	}
	if !strings.Contains(out, "match: type") {
		t.Errorf("expected type matcher in output, got: %s", out)
	}
}

func TestE2EConvertWriteFile(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(binaryPath, "convert", "--output-dir", dir, "testdata/pact/v2_basic.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	path := filepath.Join(dir, "web-app--user-api.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output file")
	}
}

func TestE2EConvertThenLint(t *testing.T) {
	dir := t.TempDir()
	convertCmd := exec.Command(binaryPath, "convert", "--output-dir", dir, "testdata/pact/v2_with_matching.json")
	convertOutput, err := convertCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("convert failed: %v\noutput: %s", err, convertOutput)
	}

	contractPath := filepath.Join(dir, "web-app--user-api.yaml")
	lintCmd := exec.Command(binaryPath, "lint", contractPath)
	lintOutput, err := lintCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lint failed on converted contract: %v\noutput: %s", err, lintOutput)
	}
	if len(lintOutput) != 0 {
		t.Errorf("expected no lint output for converted contract, got: %s", lintOutput)
	}
}

func TestE2EConvertWarnings(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "--dry-run", "testdata/pact/v3_unsupported.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 (warnings only), got error: %v\noutput: %s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "warning:") {
		t.Errorf("expected warnings in stderr, got: %s", out)
	}
	if !strings.Contains(out, "providerState") {
		t.Errorf("expected providerStates warning, got: %s", out)
	}
	if !strings.Contains(out, "message") {
		t.Errorf("expected messages warning, got: %s", out)
	}
}

func TestE2EConvertMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(binaryPath, "convert", "--output-dir", dir, "testdata/pact/v2_basic.json", "testdata/pact/v3_basic.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, output)
	}

	// Both should produce the same filename (same consumer/provider).
	path := filepath.Join(dir, "web-app--user-api.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestE2EConvertMissingFile(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "nonexistent.json")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for missing file")
	}
	if !strings.Contains(string(output), "nonexistent.json") {
		t.Errorf("expected error to mention filename, got: %s", output)
	}
}

func TestE2EConvertNoArgs(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit with no arguments")
	}
}
