// Tests for individual lint rules.
package lint

import (
	"fmt"
	"strings"
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func lintYAML(t *testing.T, yaml string) []Diagnostic {
	t.Helper()
	result, err := contract.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return New().Lint(result.Contract, result.Node)
}

func requireDiagnostic(t *testing.T, diags []Diagnostic, severity Severity, msgContains string) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == severity && strings.Contains(d.Message, msgContains) {
			return
		}
	}
	t.Errorf("expected %s diagnostic containing %q, got: %v", severity, msgContains, diags)
}

func requireNoDiagnostics(t *testing.T, diags []Diagnostic) {
	t.Helper()
	if len(diags) > 0 {
		for _, d := range diags {
			t.Errorf("unexpected: %s: %s", d.Severity, d.Message)
		}
	}
}

func TestRuleMissingAccordVersion(t *testing.T) {
	diags := lintYAML(t, `
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
	requireDiagnostic(t, diags, Error, "accord version is required")
}

func TestRuleMissingConsumerName(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: ""
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
	requireDiagnostic(t, diags, Error, "consumer.name is required")
}

func TestRuleMissingProviderName(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
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
	requireDiagnostic(t, diags, Error, "provider.name is required")
}

func TestRuleNoInteractions(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions: []
`)
	requireDiagnostic(t, diags, Error, "at least one interaction is required")
}

func TestRuleMissingDescription(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - request:
      method: GET
      path: /test
    response:
      status: 200
`)
	requireDiagnostic(t, diags, Error, "description is required")
}

func TestRuleMissingMethod(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "test"
    request:
      path: /test
    response:
      status: 200
`)
	requireDiagnostic(t, diags, Error, "request.method is required")
}

func TestRuleInvalidMethod(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "test"
    request:
      method: YEET
      path: /test
    response:
      status: 200
`)
	requireDiagnostic(t, diags, Error, "invalid HTTP method")
}

func TestRuleMissingPath(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "test"
    request:
      method: GET
    response:
      status: 200
`)
	requireDiagnostic(t, diags, Error, "request.path is required")
}

func TestRuleMissingStatus(t *testing.T) {
	diags := lintYAML(t, `
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
      headers:
        Content-Type: application/json
`)
	requireDiagnostic(t, diags, Error, "response.status is required")
}

func TestRuleInvalidStatus(t *testing.T) {
	diags := lintYAML(t, `
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
      status: 999
`)
	requireDiagnostic(t, diags, Error, "invalid status code")
}

func TestRuleValidStatusRange(t *testing.T) {
	for _, status := range []int{100, 200, 301, 404, 500, 599} {
		yaml := fmt.Sprintf(`
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
      status: %d
`, status)
		diags := lintYAML(t, yaml)
		for _, d := range diags {
			if strings.Contains(d.Message, "status") {
				t.Errorf("status %d: unexpected diagnostic: %s", status, d.Message)
			}
		}
	}
}

func TestRuleDuplicateDescriptions(t *testing.T) {
	diags := lintYAML(t, `
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "get user"
    request:
      method: GET
      path: /users/1
    response:
      status: 200
  - description: "get user"
    request:
      method: GET
      path: /users/2
    response:
      status: 200
`)
	requireDiagnostic(t, diags, Error, "duplicate interaction description")
}

func TestRuleMatchingRulePathMissingPrefix(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        id: 1
    matching_rules:
      "body.id":
        match: type
`)
	requireDiagnostic(t, diags, Warning, "should start with")
}

func TestRuleMatchingRuleUnknownType(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        id: 1
    matching_rules:
      "$.body.id":
        match: fuzzy
`)
	requireDiagnostic(t, diags, Error, "unknown match type")
}

func TestRuleMatchingRuleRegexMissingPattern(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        name: "Jane"
    matching_rules:
      "$.body.name":
        match: regex
`)
	requireDiagnostic(t, diags, Error, "regex field is required")
}

func TestRuleMatchingRuleInvalidRegex(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        name: "Jane"
    matching_rules:
      "$.body.name":
        match: regex
        regex: "[invalid"
`)
	requireDiagnostic(t, diags, Error, "invalid regex")
}

func TestRuleValidMatchingRules(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        id: 1
        name: "Jane"
        email: "jane@example.com"
    matching_rules:
      "$.body.id":
        match: type
      "$.body.name":
        match: exact
      "$.body.email":
        match: regex
        regex: "^[^@]+@[^@]+$"
`)
	requireNoDiagnostics(t, diags)
}

func TestRuleDiagnosticLineNumbers(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        id: 1
    matching_rules:
      "body.id":
        match: type
`)
	// The warning about "body.id" path should have a line number > 1
	for _, d := range diags {
		if strings.Contains(d.Message, "should start with") {
			if d.Line <= 1 {
				t.Errorf("expected line > 1 for path warning, got %d", d.Line)
			}
			return
		}
	}
	t.Error("expected path warning diagnostic")
}

func TestRuleMatchingRuleValidBracketSyntax(t *testing.T) {
	diags := lintYAML(t, `
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
      body:
        users:
          - name: "Jane"
    matching_rules:
      "$.body.users[0].name":
        match: type
      "$.body.users[*].name":
        match: type
`)
	requireNoDiagnostics(t, diags)
}

func TestRuleMatchingRuleInvalidBracketSyntax(t *testing.T) {
	tests := []struct {
		name string
		path string
		msg  string
	}{
		{"empty brackets", "$.body.items[].name", "invalid bracket syntax"},
		{"non-numeric", "$.body.items[abc].name", "invalid bracket syntax"},
		{"negative index", "$.body.items[-1].name", "invalid bracket syntax"},
		{"unclosed bracket", "$.body.items[0.name", "invalid bracket syntax"},
		{"filter expression", "$.body.items[?(@.id>1)].name", "invalid bracket syntax"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := fmt.Sprintf(`
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
      body:
        items:
          - name: "x"
    matching_rules:
      "%s":
        match: type
`, tt.path)
			diags := lintYAML(t, yaml)
			requireDiagnostic(t, diags, Warning, tt.msg)
		})
	}
}
