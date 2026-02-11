// Tests for mapping OpenAPI operations to Accord interactions.
package generate

import (
	"testing"

	"github.com/andrewh/accord/internal/openapi"
)

func TestBuildInteractionBasic(t *testing.T) {
	ep := openapi.Endpoint{
		Method:  "GET",
		Path:    "/health",
		Summary: "Health check",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"status"},
				Properties: map[string]*openapi.SchemaRef{
					"status": {Type: "string"},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}

	ix := interactions[0]
	if ix.Description != "Health check" {
		t.Errorf("expected description 'Health check', got %q", ix.Description)
	}
	if ix.Request.Method != "GET" {
		t.Errorf("expected method GET, got %q", ix.Request.Method)
	}
	if ix.Request.Path != "/health" {
		t.Errorf("expected path /health, got %q", ix.Request.Path)
	}
	if ix.Response.Status != 200 {
		t.Errorf("expected status 200, got %d", ix.Response.Status)
	}
}

func TestBuildInteractionResponseBody(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/users",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"name", "active"},
				Properties: map[string]*openapi.SchemaRef{
					"name":   {Type: "string"},
					"active": {Type: "boolean"},
					"notes":  {Type: "string"}, // optional, should be excluded
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}

	body, ok := interactions[0].Response.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected map body, got %T", interactions[0].Response.Body)
	}
	if _, hasName := body["name"]; !hasName {
		t.Error("expected 'name' in response body")
	}
	if _, hasActive := body["active"]; !hasActive {
		t.Error("expected 'active' in response body")
	}
	if _, hasNotes := body["notes"]; hasNotes {
		t.Error("expected 'notes' to be excluded from response body")
	}
}

func TestBuildInteractionMatchingRulesType(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/item",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*openapi.SchemaRef{
					"id": {Type: "integer"},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	rule, ok := ix.MatchingRules["$.body.id"]
	if !ok {
		t.Fatal("expected matching rule for $.body.id")
	}
	if rule.Match != "type" {
		t.Errorf("expected match 'type', got %q", rule.Match)
	}
}

func TestBuildInteractionMatchingRulesRegex(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/item",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"email"},
				Properties: map[string]*openapi.SchemaRef{
					"email": {Type: "string", Pattern: "^[^@]+@[^@]+$"},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	rule, ok := ix.MatchingRules["$.body.email"]
	if !ok {
		t.Fatal("expected matching rule for $.body.email")
	}
	if rule.Match != "regex" {
		t.Errorf("expected match 'regex', got %q", rule.Match)
	}
	if rule.Regex != "^[^@]+@[^@]+$" {
		t.Errorf("expected regex pattern, got %q", rule.Regex)
	}
}

func TestBuildInteractionMatchingRulesExact(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/item",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"kind"},
				Properties: map[string]*openapi.SchemaRef{
					"kind": {Type: "string", Enum: []any{"widget"}},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	rule, ok := ix.MatchingRules["$.body.kind"]
	if !ok {
		t.Fatal("expected matching rule for $.body.kind")
	}
	if rule.Match != "exact" {
		t.Errorf("expected match 'exact' for single-value enum, got %q", rule.Match)
	}
}

func TestBuildInteractionPathParameters(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/users/{userId}/posts/{postId}",
		Parameters: []openapi.Parameter{
			{
				Name: "userId",
				In:   "path",
				Schema: &openapi.SchemaRef{
					Type:    "integer",
					Example: float64(42),
				},
			},
			{
				Name:   "postId",
				In:     "path",
				Schema: &openapi.SchemaRef{Type: "string"},
			},
		},
		Responses: map[int]*openapi.SchemaRef{200: nil},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	if ix.Request.Path != "/users/42/posts/postId" {
		t.Errorf("expected path with substituted params, got %q", ix.Request.Path)
	}
}

func TestBuildInteractionPathParamIntegerDefault(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/items/{itemId}",
		Parameters: []openapi.Parameter{
			{
				Name:   "itemId",
				In:     "path",
				Schema: &openapi.SchemaRef{Type: "integer"},
			},
		},
		Responses: map[int]*openapi.SchemaRef{200: nil},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	if ix.Request.Path != "/items/1" {
		t.Errorf("expected path /items/1, got %q", ix.Request.Path)
	}
}

func TestBuildInteractionRequestBody(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "POST",
		Path:   "/users",
		RequestBody: &openapi.SchemaRef{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*openapi.SchemaRef{
				"name":  {Type: "string"},
				"email": {Type: "string"}, // optional
			},
		},
		Responses: map[int]*openapi.SchemaRef{
			201: {
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*openapi.SchemaRef{
					"id": {Type: "integer"},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}
	ix := interactions[0]
	body, ok := ix.Request.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected map request body, got %T", ix.Request.Body)
	}
	if _, hasName := body["name"]; !hasName {
		t.Error("expected 'name' in request body")
	}
	if _, hasEmail := body["email"]; hasEmail {
		t.Error("expected 'email' to be excluded from request body")
	}
}

func TestBuildInteractionMultipleResponses(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "POST",
		Path:   "/items",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*openapi.SchemaRef{
					"id": {Type: "integer"},
				},
			},
			201: {
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*openapi.SchemaRef{
					"id": {Type: "integer"},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	if len(interactions) != 2 {
		t.Fatalf("expected 2 interactions (one per 2xx response), got %d", len(interactions))
	}
}

func TestBuildInteractionNoResponses(t *testing.T) {
	ep := openapi.Endpoint{
		Method:    "DELETE",
		Path:      "/items/{id}",
		Responses: map[int]*openapi.SchemaRef{},
	}

	interactions := BuildInteractions(ep)
	if len(interactions) != 0 {
		t.Errorf("expected 0 interactions for endpoint with no responses, got %d", len(interactions))
	}
}

func TestBuildInteractionNestedMatchingRules(t *testing.T) {
	ep := openapi.Endpoint{
		Method: "GET",
		Path:   "/item",
		Responses: map[int]*openapi.SchemaRef{
			200: {
				Type:     "object",
				Required: []string{"address"},
				Properties: map[string]*openapi.SchemaRef{
					"address": {
						Type:     "object",
						Required: []string{"city"},
						Properties: map[string]*openapi.SchemaRef{
							"city": {Type: "string"},
						},
					},
				},
			},
		},
	}

	interactions := BuildInteractions(ep)
	ix := interactions[0]
	if _, ok := ix.MatchingRules["$.body.address.city"]; !ok {
		t.Error("expected matching rule for $.body.address.city")
	}
}

func TestBuildInteractionDescriptionNoSummary(t *testing.T) {
	ep := openapi.Endpoint{
		Method:    "DELETE",
		Path:      "/items/{id}",
		Responses: map[int]*openapi.SchemaRef{200: nil},
	}

	interactions := BuildInteractions(ep)
	if interactions[0].Description != "DELETE /items/{id}" {
		t.Errorf("expected fallback description, got %q", interactions[0].Description)
	}
}

func TestBuildInteractionDescriptionWithSummary(t *testing.T) {
	ep := openapi.Endpoint{
		Method:  "GET",
		Path:    "/pets",
		Summary: "List all pets",
		Responses: map[int]*openapi.SchemaRef{
			200: nil,
		},
	}

	interactions := BuildInteractions(ep)
	if interactions[0].Description != "List all pets" {
		t.Errorf("expected description from summary, got %q", interactions[0].Description)
	}
}

func TestBuildInteractionDescriptionMultipleResponsesWithSummary(t *testing.T) {
	ep := openapi.Endpoint{
		Method:  "POST",
		Path:    "/items",
		Summary: "Create item",
		Responses: map[int]*openapi.SchemaRef{
			200: nil,
			201: nil,
		},
	}

	interactions := BuildInteractions(ep)
	// With multiple responses, description should include status to disambiguate.
	descs := make(map[string]bool)
	for _, ix := range interactions {
		descs[ix.Description] = true
	}
	if !descs["Create item (200)"] || !descs["Create item (201)"] {
		t.Errorf("expected descriptions with status codes, got %v", descs)
	}
}
