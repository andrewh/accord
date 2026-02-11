// Lint engine: runs rules against a parsed contract and collects diagnostics.
package lint

import (
	"fmt"
	"sort"

	"github.com/andrewh/accord/internal/contract"
	"gopkg.in/yaml.v3"
)

// Severity indicates whether a diagnostic is an error or warning.
type Severity int

const (
	Error   Severity = iota
	Warning
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "unknown"
	}
}

// Diagnostic represents a single lint finding.
type Diagnostic struct {
	Severity Severity
	Message  string
	Line     int
	Column   int
	Path     string // e.g. "interactions[0].request.method"
}

// FormatDiagnostic formats a diagnostic with a filename prefix.
func FormatDiagnostic(filename string, d Diagnostic) string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", filename, d.Line, d.Column, d.Severity, d.Message)
}

// Rule is a function that examines a contract and returns diagnostics.
type Rule func(c *contract.Contract, node *yaml.Node) []Diagnostic

// Linter runs a set of rules against a parsed contract.
type Linter struct {
	rules []Rule
}

// New creates a linter with the default set of rules.
func New() *Linter {
	return &Linter{
		rules: DefaultRules(),
	}
}

// Lint runs all rules and returns diagnostics sorted by line number.
func (l *Linter) Lint(c *contract.Contract, node *yaml.Node) []Diagnostic {
	var all []Diagnostic
	for _, rule := range l.rules {
		all = append(all, rule(c, node)...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Line != all[j].Line {
			return all[i].Line < all[j].Line
		}
		return all[i].Column < all[j].Column
	})
	return all
}

// HasErrors returns true if any diagnostic has Error severity.
func HasErrors(diagnostics []Diagnostic) bool {
	for _, d := range diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}
