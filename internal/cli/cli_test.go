// Unit tests for CLI commands exercised through the cobra command tree.
package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// executeCommand runs a CLI command through rootCmd, capturing stdout, stderr,
// and the returned error. Flag variables are reset to defaults before each call.
func executeCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Reset package-level flag variables to their declared defaults so that
	// values from a previous test invocation do not leak across tests.
	genConsumer = "my-service"
	genEndpoints = ""
	genOutputDir = "."
	genDryRun = false
	convertOutputDir = "."
	convertDryRun = false
	providerURL = ""
	timeout = 30

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// --- version -----------------------------------------------------------------

func TestVersionOutput(t *testing.T) {
	stdout, _, err := executeCommand(t, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "accord dev\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// --- lint --------------------------------------------------------------------

func TestLintValidFile(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "lint", "../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("expected no stderr, got %q", stderr)
	}
}

func TestLintInvalidFile(t *testing.T) {
	stdout, _, err := executeCommand(t, "lint", "../../testdata/invalid/missing_fields.yaml")
	if err == nil {
		t.Fatal("expected error for invalid file")
	}
	for _, msg := range []string{
		"consumer.name is required",
		"provider.name is required",
		"description is required",
		"request.method is required",
		"response.status is required",
	} {
		if !strings.Contains(stdout, msg) {
			t.Errorf("expected stdout to contain %q, got:\n%s", msg, stdout)
		}
	}
}

func TestLintMissingFile(t *testing.T) {
	_, stderr, err := executeCommand(t, "lint", "nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(stderr, "nonexistent.yaml") {
		t.Errorf("expected filename in stderr, got: %s", stderr)
	}
}

func TestLintMultipleFilesMixed(t *testing.T) {
	stdout, stderr, err := executeCommand(t, "lint",
		"../../testdata/valid/minimal.yaml",
		"nonexistent.yaml",
		"../../testdata/invalid/missing_fields.yaml",
	)
	if err == nil {
		t.Fatal("expected error when some files are invalid")
	}
	if !strings.Contains(stderr, "nonexistent.yaml") {
		t.Errorf("expected missing file error in stderr, got: %s", stderr)
	}
	if !strings.Contains(stdout, "consumer.name is required") {
		t.Errorf("expected lint diagnostics in stdout, got: %s", stdout)
	}
}

// --- generate ----------------------------------------------------------------

func TestGenerateDryRun(t *testing.T) {
	stdout, _, err := executeCommand(t, "generate", "--dry-run", "../../testdata/openapi/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"accord:", "consumer:", "provider:", "interactions:"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in stdout, got:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stdout, "# ") {
		t.Error("expected filename comment header in stdout")
	}
}

func TestGenerateCustomConsumer(t *testing.T) {
	stdout, _, err := executeCommand(t, "generate", "--dry-run", "--consumer", "order-svc", "../../testdata/openapi/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "order-svc") {
		t.Errorf("expected consumer name in output, got:\n%s", stdout)
	}
}

func TestGenerateNoInteractions(t *testing.T) {
	_, stderr, err := executeCommand(t, "generate", "--dry-run", "--endpoints", "/nonexistent", "../../testdata/openapi/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "no interactions generated") {
		t.Errorf("expected warning on stderr, got: %s", stderr)
	}
}

func TestGenerateWriteMode(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := executeCommand(t, "generate", "--output-dir", dir, "--consumer", "test-svc", "../../testdata/openapi/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "wrote") {
		t.Errorf("expected 'wrote' confirmation, got: %s", stdout)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one file in output directory")
	}
}

func TestGenerateMissingSpec(t *testing.T) {
	_, stderr, err := executeCommand(t, "generate", "--dry-run", "nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing spec")
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

// --- convert -----------------------------------------------------------------

func TestConvertDryRun(t *testing.T) {
	stdout, _, err := executeCommand(t, "convert", "--dry-run", "../../testdata/pact/v2_basic.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "accord:") {
		t.Errorf("expected YAML with 'accord:', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "web-app") {
		t.Errorf("expected consumer name in output, got:\n%s", stdout)
	}
}

func TestConvertMissingFile(t *testing.T) {
	_, stderr, err := executeCommand(t, "convert", "nonexistent.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(stderr, "error:") {
		t.Errorf("expected 'error:' prefix on stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "nonexistent.json") {
		t.Errorf("expected filename in stderr, got: %s", stderr)
	}
}

func TestConvertWarnings(t *testing.T) {
	_, stderr, err := executeCommand(t, "convert", "--dry-run", "../../testdata/pact/v3_unsupported.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "warning:") {
		t.Errorf("expected warnings on stderr, got: %s", stderr)
	}
}

func TestConvertWriteMode(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := executeCommand(t, "convert", "--output-dir", dir, "../../testdata/pact/v2_basic.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "wrote") {
		t.Errorf("expected 'wrote' confirmation, got: %s", stdout)
	}
	path := filepath.Join(dir, "web-app--user-api.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected output file to exist: %v", err)
	}
}

func TestConvertMultipleFiles(t *testing.T) {
	stdout, _, err := executeCommand(t, "convert", "--dry-run",
		"../../testdata/pact/v2_basic.json",
		"../../testdata/pact/v3_basic.json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(stdout, "# "); count < 2 {
		t.Errorf("expected at least 2 filename headers, got %d in:\n%s", count, stdout)
	}
}

// --- verify ------------------------------------------------------------------

func TestVerifyPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	stdout, _, err := executeCommand(t, "verify", "--provider-url", server.URL, "../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "PASS") {
		t.Errorf("expected PASS in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All interactions passed") {
		t.Errorf("expected success message, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Verifying") {
		t.Errorf("expected 'Verifying' header, got:\n%s", stdout)
	}
}

func TestVerifyFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	stdout, _, err := executeCommand(t, "verify", "--provider-url", server.URL, "../../testdata/valid/minimal.yaml")
	if err == nil {
		t.Fatal("expected error for verification failure")
	}
	if !strings.Contains(stdout, "FAIL") {
		t.Errorf("expected FAIL in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "health check") {
		t.Errorf("expected interaction description in output, got:\n%s", stdout)
	}
}

func TestVerifyWarn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"data": "this is larger than 10 bytes"})
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

	stdout, _, err := executeCommand(t, "verify", "--provider-url", server.URL, contractPath)
	if err != nil {
		t.Fatalf("expected no error (warning only), got: %v", err)
	}
	if !strings.Contains(stdout, "WARN") {
		t.Errorf("expected WARN in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "[warning]") {
		t.Errorf("expected [warning] prefix in output, got:\n%s", stdout)
	}
}

func TestVerifyMissingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	_, stderr, err := executeCommand(t, "verify", "--provider-url", server.URL, "nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(stderr, "nonexistent.yaml") {
		t.Errorf("expected filename in stderr, got: %s", stderr)
	}
}
