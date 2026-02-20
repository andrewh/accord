// Path segment parsing for JSONPath-style bracket notation (e.g. "users[0]", "items[*]").
package contract

import (
	"fmt"
	"strconv"
	"strings"
)

// PathSegment represents a single element of a dot-separated path,
// optionally with an array index or wildcard suffix.
type PathSegment struct {
	Field    string
	Index    int // -1 means no index
	Wildcard bool
}

// ParseSegment parses a path segment like "name", "users[0]", or "items[*]".
func ParseSegment(s string) (PathSegment, error) {
	open := strings.IndexByte(s, '[')
	close := strings.IndexByte(s, ']')

	// No brackets - plain field name.
	if open == -1 && close == -1 {
		return PathSegment{Field: s, Index: -1}, nil
	}

	// Mismatched brackets.
	if open == -1 || close == -1 || close != len(s)-1 || close <= open+1 {
		return PathSegment{}, fmt.Errorf("invalid bracket syntax in segment %q", s)
	}

	field := s[:open]
	if field == "" {
		return PathSegment{}, fmt.Errorf("missing field name in segment %q", s)
	}

	inner := s[open+1 : close]

	if inner == "*" {
		return PathSegment{Field: field, Index: -1, Wildcard: true}, nil
	}

	idx, err := strconv.Atoi(inner)
	if err != nil || idx < 0 {
		return PathSegment{}, fmt.Errorf("invalid array index in segment %q", s)
	}

	return PathSegment{Field: field, Index: idx}, nil
}
