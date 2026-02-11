// Tests for path segment parsing.
package contract

import (
	"testing"
)

func TestParseSegment(t *testing.T) {
	tests := []struct {
		input    string
		want     PathSegment
		wantErr  bool
	}{
		{"name", PathSegment{Field: "name", Index: -1}, false},
		{"users[0]", PathSegment{Field: "users", Index: 0}, false},
		{"users[42]", PathSegment{Field: "users", Index: 42}, false},
		{"items[*]", PathSegment{Field: "items", Wildcard: true, Index: -1}, false},
		{"[0]", PathSegment{}, true},          // no field name
		{"users[]", PathSegment{}, true},      // empty brackets
		{"users[-1]", PathSegment{}, true},    // negative index
		{"users[abc]", PathSegment{}, true},   // non-numeric, non-wildcard
		{"users[0", PathSegment{}, true},      // unclosed bracket
		{"users0]", PathSegment{}, true},      // no opening bracket
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSegment(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseSegment(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
