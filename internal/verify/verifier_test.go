// Tests for the contract verification engine using httptest servers.
package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrewh/accord/internal/contract"
)

func TestVerifyExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("include") != "email" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"name":  "Jane",
			"email": "jane@example.com",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request: contract.Request{
					Method: "GET",
					Path:   "/users/1",
					Query:  map[string]string{"include": "email"},
				},
				Response: contract.Response{
					Status:  200,
					Headers: map[string]string{"Content-Type": "application/json"},
					Body: map[string]any{
						"id":    1,
						"name":  "Jane",
						"email": "jane@example.com",
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyStatusMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response:    contract.Response{Status: 200},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for status mismatch")
	}
	if len(results[0].Failures) == 0 {
		t.Error("expected at least one failure")
	}
	if results[0].Failures[0].Field != "status" {
		t.Errorf("failure field = %q, want %q", results[0].Failures[0].Field, "status")
	}
}

func TestVerifyHeaderMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get data",
				Request:     contract.Request{Method: "GET", Path: "/data"},
				Response: contract.Response{
					Status:  200,
					Headers: map[string]string{"Content-Type": "application/json"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for header mismatch")
	}
}

func TestVerifyBodyFieldMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   1,
			"name": "Bob",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"id":   1,
						"name": "Jane",
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for body mismatch")
	}
}

func TestVerifyWithTypeMatchingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   999,
			"name": "Bob",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"id":   1,
						"name": "Jane",
					},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.id":   {Match: "type"},
					"$.body.name": {Match: "type"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass with type matching, got failures: %v", results[0].Failures)
	}
}

func TestVerifyWithRegexMatchingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"email": "bob@test.com",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status: 200,
					Body:   map[string]any{"email": "jane@example.com"},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.email": {Match: "regex", Regex: "^[^@]+@[^@]+$"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass with regex matching, got failures: %v", results[0].Failures)
	}
}

func TestVerifyRegexFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"email": "not-an-email",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status: 200,
					Body:   map[string]any{"email": "jane@example.com"},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.email": {Match: "regex", Regex: "^[^@]+@[^@]+$"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for regex mismatch")
	}
}

func TestVerifyWithRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["name"] != "Jane" {
			t.Errorf("request body name = %v, want %q", body["name"], "Jane")
		}
		w.WriteHeader(201)
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "create user",
				Request: contract.Request{
					Method: "POST",
					Path:   "/users",
					Body:   map[string]any{"name": "Jane"},
				},
				Response: contract.Response{Status: 201},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyWithRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer token123")
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "authenticated request",
				Request: contract.Request{
					Method:  "GET",
					Path:    "/protected",
					Headers: map[string]string{"Authorization": "Bearer token123"},
				},
				Response: contract.Response{Status: 200},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyCollectsAllFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"error": "oops"})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status:  200,
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    map[string]any{"id": 1},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failures")
	}
	// Should have failures for status, header, and body
	if len(results[0].Failures) < 2 {
		t.Errorf("expected multiple failures, got %d: %v", len(results[0].Failures), results[0].Failures)
	}
}

func TestVerifyMultipleInteractions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(200)
		case "/users/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		}
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "health check",
				Request:     contract.Request{Method: "GET", Path: "/health"},
				Response:    contract.Response{Status: 200},
			},
			{
				Description: "get user",
				Request:     contract.Request{Method: "GET", Path: "/users/1"},
				Response: contract.Response{
					Status: 200,
					Body:   map[string]any{"id": 1},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.id": {Match: "type"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, r := range results {
		if !r.Passed {
			t.Errorf("interaction %d (%s) failed: %v", i, r.Interaction, r.Failures)
		}
	}
}

func TestVerifyArrayExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"tags": []any{"go", "testing"},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get tags",
				Request:     contract.Request{Method: "GET", Path: "/tags"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"tags": []any{"go", "testing"},
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyArrayLengthMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"tags": []any{"go"},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get tags",
				Request:     contract.Request{Method: "GET", Path: "/tags"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"tags": []any{"go", "testing"},
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for array length mismatch")
	}
}

func TestVerifyArrayTypeMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"tags": "not-an-array",
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "get tags",
				Request:     contract.Request{Method: "GET", Path: "/tags"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"tags": []any{"go"},
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for type mismatch (string vs array)")
	}
}

func TestVerifyNestedArrayObjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"users": []any{
				map[string]any{"name": "Alice", "email": "alice@test.com"},
				map[string]any{"name": "Bob", "email": "bob@test.com"},
			},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "list users",
				Request:     contract.Request{Method: "GET", Path: "/users"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"users": []any{
							map[string]any{"name": "Alice", "email": "alice@test.com"},
							map[string]any{"name": "Bob", "email": "bob@test.com"},
						},
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyArrayElementMatchingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"users": []any{
				map[string]any{"name": "DifferentName", "email": "someone@test.com"},
			},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "list users with type matching",
				Request:     contract.Request{Method: "GET", Path: "/users"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"users": []any{
							map[string]any{"name": "Alice", "email": "alice@test.com"},
						},
					},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.users[0].name":  {Match: "type"},
					"$.body.users[0].email": {Match: "regex", Regex: "^[^@]+@[^@]+$"},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if !results[0].Passed {
		t.Errorf("expected pass with element matching rules, got failures: %v", results[0].Failures)
	}
}

func TestVerifyArrayElementFailurePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{"actual_a", "actual_b"},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "check items",
				Request:     contract.Request{Method: "GET", Path: "/items"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"items": []any{"expected_a", "expected_b"},
					},
				},
			},
		},
	}

	results := Verify(c, server.URL)
	if results[0].Passed {
		t.Error("expected failure for element mismatch")
	}
	// Failure paths should reference individual elements
	for _, f := range results[0].Failures {
		if f.Field == "body.items[0]" || f.Field == "body.items[1]" {
			return // found element-level path
		}
	}
	t.Errorf("expected element-level failure paths (body.items[0] or body.items[1]), got: %v", results[0].Failures)
}
