// Tests for the lint engine.
package lint

import (
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func TestLintValidContract(t *testing.T) {
	data := []byte(`
accord: "0.1"
consumer:
  name: "client"
provider:
  name: "server"
interactions:
  - description: "health check"
    request:
      method: GET
      path: /health
    response:
      status: 200
`)
	result, err := contract.Parse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	linter := New()
	diags := linter.Lint(result.Contract, result.Node)

	if len(diags) != 0 {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic: %s: %s", d.Path, d.Message)
		}
	}
}

func TestLintSortsDiagnostics(t *testing.T) {
	data := []byte(`
accord: "0.1"
consumer:
  name: ""
provider:
  name: ""
interactions:
  - description: "test"
    request:
      method: GET
      path: /test
    response:
      status: 200
`)
	result, err := contract.Parse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	linter := New()
	diags := linter.Lint(result.Contract, result.Node)

	for i := 1; i < len(diags); i++ {
		if diags[i].Line < diags[i-1].Line {
			t.Errorf("diagnostics not sorted by line: %d < %d", diags[i].Line, diags[i-1].Line)
		}
	}
}

func TestHasErrors(t *testing.T) {
	if HasErrors(nil) {
		t.Error("HasErrors(nil) = true, want false")
	}
	if HasErrors([]Diagnostic{{Severity: Warning}}) {
		t.Error("HasErrors with only warnings = true, want false")
	}
	if !HasErrors([]Diagnostic{{Severity: Error}}) {
		t.Error("HasErrors with error = false, want true")
	}
}

func TestFormatDiagnostic(t *testing.T) {
	d := Diagnostic{
		Severity: Error,
		Message:  "request.method is required",
		Line:     12,
		Column:   5,
		Path:     "interactions[0].request.method",
	}
	got := FormatDiagnostic("contracts/user.yaml", d)
	want := "contracts/user.yaml:12:5: error: request.method is required"
	if got != want {
		t.Errorf("FormatDiagnostic = %q, want %q", got, want)
	}
}
