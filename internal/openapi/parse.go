// Load and resolve OpenAPI specs into Accord's own types.
package openapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Spec holds the parsed OpenAPI specification in Accord's types.
type Spec struct {
	Title     string
	Version   string
	Endpoints []Endpoint
}

// Endpoint describes a single API operation.
type Endpoint struct {
	Method      string
	Path        string
	Summary     string
	Parameters  []Parameter
	RequestBody *SchemaRef
	Responses   map[int]*SchemaRef
}

// Parameter describes an API operation parameter.
type Parameter struct {
	Name   string
	In     string // "path", "query", "header"
	Schema *SchemaRef
}

// SchemaRef describes a data schema with enough detail to generate examples.
type SchemaRef struct {
	Type       string
	Format     string
	Properties map[string]*SchemaRef
	Items      *SchemaRef
	Required   []string
	Enum       []any
	Pattern    string
	Example    any
	AllOf      []*SchemaRef
	OneOf      []*SchemaRef
	AnyOf      []*SchemaRef
}

// ParseFile loads and resolves an OpenAPI spec from disk.
func ParseFile(path string) (*Spec, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validating OpenAPI spec: %w", err)
	}
	return convertSpec(doc), nil
}

func convertSpec(doc *openapi3.T) *Spec {
	spec := &Spec{
		Title:   doc.Info.Title,
		Version: doc.Info.Version,
	}

	// Sort paths for deterministic output.
	paths := make([]string, 0, len(doc.Paths.Map()))
	for p := range doc.Paths.Map() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := doc.Paths.Map()[path]
		for _, methodOp := range methodOperations(path, item) {
			spec.Endpoints = append(spec.Endpoints, methodOp)
		}
	}

	return spec
}

// methodOperations returns endpoints for each HTTP method defined on a path item.
func methodOperations(path string, item *openapi3.PathItem) []Endpoint {
	type methodEntry struct {
		method string
		op     *openapi3.Operation
	}

	// Ordered by conventional HTTP method order.
	candidates := []methodEntry{
		{"GET", item.Get},
		{"POST", item.Post},
		{"PUT", item.Put},
		{"PATCH", item.Patch},
		{"DELETE", item.Delete},
		{"HEAD", item.Head},
		{"OPTIONS", item.Options},
	}

	var endpoints []Endpoint
	for _, c := range candidates {
		if c.op == nil {
			continue
		}

		ep := Endpoint{
			Method:    c.method,
			Path:      path,
			Summary:   c.op.Summary,
			Responses: make(map[int]*SchemaRef),
		}

		// Collect parameters from path item and operation.
		for _, p := range item.Parameters {
			if p.Value != nil {
				ep.Parameters = append(ep.Parameters, convertParameter(p.Value))
			}
		}
		for _, p := range c.op.Parameters {
			if p.Value != nil {
				ep.Parameters = append(ep.Parameters, convertParameter(p.Value))
			}
		}

		// Request body.
		if c.op.RequestBody != nil && c.op.RequestBody.Value != nil {
			if ct := c.op.RequestBody.Value.Content.Get("application/json"); ct != nil && ct.Schema != nil {
				ep.RequestBody = convertSchemaRef(ct.Schema)
			}
		}

		// Responses: only 2xx and default.
		if c.op.Responses != nil {
			for code, resp := range c.op.Responses.Map() {
				status := parseStatusCode(code)
				if status < 0 {
					continue
				}
				if resp.Value == nil {
					continue
				}
				ct := resp.Value.Content.Get("application/json")
				if ct != nil && ct.Schema != nil {
					ep.Responses[status] = convertSchemaRef(ct.Schema)
				} else {
					// Response exists but has no JSON schema.
					ep.Responses[status] = nil
				}
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints
}

// parseStatusCode parses a response status code string.
// Returns -1 for non-2xx codes (we skip error responses).
// "default" is treated as 200.
func parseStatusCode(code string) int {
	if strings.EqualFold(code, "default") {
		return 200
	}
	n, err := strconv.Atoi(code)
	if err != nil {
		return -1
	}
	if n < 200 || n > 299 {
		return -1
	}
	return n
}

func convertParameter(p *openapi3.Parameter) Parameter {
	param := Parameter{
		Name: p.Name,
		In:   p.In,
	}
	if p.Schema != nil {
		param.Schema = convertSchemaRef(p.Schema)
	}
	return param
}

func convertSchemaRef(ref *openapi3.SchemaRef) *SchemaRef {
	if ref == nil || ref.Value == nil {
		return nil
	}
	s := ref.Value
	var typ string
	if types := s.Type.Slice(); len(types) > 0 {
		typ = types[0]
	}
	schema := &SchemaRef{
		Type:     typ,
		Format:   s.Format,
		Required: s.Required,
		Pattern:  s.Pattern,
		Example:  s.Example,
	}

	if len(s.Enum) > 0 {
		schema.Enum = s.Enum
	}

	if len(s.Properties) > 0 {
		schema.Properties = make(map[string]*SchemaRef, len(s.Properties))
		for name, prop := range s.Properties {
			schema.Properties[name] = convertSchemaRef(prop)
		}
	}

	if s.Items != nil {
		schema.Items = convertSchemaRef(s.Items)
	}

	for _, a := range s.AllOf {
		schema.AllOf = append(schema.AllOf, convertSchemaRef(a))
	}
	for _, o := range s.OneOf {
		schema.OneOf = append(schema.OneOf, convertSchemaRef(o))
	}
	for _, a := range s.AnyOf {
		schema.AnyOf = append(schema.AnyOf, convertSchemaRef(a))
	}

	return schema
}
