// Tests for the contract verification engine using httptest servers.
package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	results := Verify(c, server.URL, 30*time.Second)

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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

	results := Verify(c, server.URL, 30*time.Second)
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

func TestVerifyWildcardMatchingRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"users": []any{
				map[string]any{"name": "Xavier", "email": "x@test.com"},
				map[string]any{"name": "Yolanda", "email": "y@test.com"},
			},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "list users with wildcard rules",
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
				MatchingRules: contract.MatchingRules{
					"$.body.users[*].name":  {Match: "type"},
					"$.body.users[*].email": {Match: "regex", Regex: "^[^@]+@[^@]+$"},
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	if !results[0].Passed {
		t.Errorf("expected pass with wildcard matching rules, got failures: %v", results[0].Failures)
	}
}

func TestSeverityWarningOnlyPasses(t *testing.T) {
	r := Result{
		Interaction: "test",
		Failures: []Failure{
			{Field: "nfr.max_response_bytes", Message: "exceeded", Severity: SeverityWarning},
		},
	}
	r.Passed = !hasErrors(r.Failures)
	if !r.Passed {
		t.Error("expected Passed=true when only warnings present")
	}
}

func TestSeverityErrorFails(t *testing.T) {
	r := Result{
		Interaction: "test",
		Failures: []Failure{
			{Field: "nfr.max_round_trip_ms", Message: "exceeded", Severity: SeverityError},
		},
	}
	r.Passed = !hasErrors(r.Failures)
	if r.Passed {
		t.Error("expected Passed=false when error present")
	}
}

func TestSeverityMixedFails(t *testing.T) {
	r := Result{
		Interaction: "test",
		Failures: []Failure{
			{Field: "nfr.max_response_bytes", Message: "exceeded", Severity: SeverityWarning},
			{Field: "nfr.max_round_trip_ms", Message: "exceeded", Severity: SeverityError},
		},
	}
	r.Passed = !hasErrors(r.Failures)
	if r.Passed {
		t.Error("expected Passed=false when mix of warnings and errors")
	}
}

func TestSendRequestMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	req := contract.Request{Method: "GET", Path: "/health"}
	metrics, err := sendRequest(req, server.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Response == nil {
		t.Fatal("expected non-nil response")
	}
	if metrics.Response.StatusCode != 200 {
		t.Errorf("status = %d, want 200", metrics.Response.StatusCode)
	}
	if metrics.RoundTripMs < 0 {
		t.Errorf("RoundTripMs = %d, want >= 0", metrics.RoundTripMs)
	}
	if metrics.TimeToFirstByteMs < 0 {
		t.Errorf("TimeToFirstByteMs = %d, want >= 0", metrics.TimeToFirstByteMs)
	}
}

func TestCheckNFRBytesExceeded(t *testing.T) {
	var result Result
	nfr := &contract.NFR{
		MaxResponseBytes: &contract.NFRThreshold{Threshold: 10},
	}
	metrics := &RequestMetrics{RoundTripMs: 5, TimeToFirstByteMs: 2}
	checkNFR(&result, nfr, metrics, 100)

	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(result.Failures), result.Failures)
	}
	if result.Failures[0].Severity != SeverityError {
		t.Errorf("expected error severity, got %d", result.Failures[0].Severity)
	}
	if result.Failures[0].Field != "nfr.max_response_bytes" {
		t.Errorf("field = %q, want %q", result.Failures[0].Field, "nfr.max_response_bytes")
	}
}

func TestCheckNFRBytesWithinLimit(t *testing.T) {
	var result Result
	nfr := &contract.NFR{
		MaxResponseBytes: &contract.NFRThreshold{Threshold: 1000},
	}
	metrics := &RequestMetrics{RoundTripMs: 5, TimeToFirstByteMs: 2}
	checkNFR(&result, nfr, metrics, 100)

	if len(result.Failures) != 0 {
		t.Errorf("expected no failures, got: %v", result.Failures)
	}
}

func TestCheckNFRWarningSeverity(t *testing.T) {
	var result Result
	nfr := &contract.NFR{
		MaxResponseBytes: &contract.NFRThreshold{Threshold: 10, Severity: "warning"},
	}
	metrics := &RequestMetrics{RoundTripMs: 5, TimeToFirstByteMs: 2}
	checkNFR(&result, nfr, metrics, 100)

	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Failures))
	}
	if result.Failures[0].Severity != SeverityWarning {
		t.Errorf("expected warning severity, got %d", result.Failures[0].Severity)
	}
}

func TestCheckNFRTimingExceeded(t *testing.T) {
	var result Result
	nfr := &contract.NFR{
		MaxRoundTripMs:       &contract.NFRThreshold{Threshold: 10},
		MaxTimeToFirstByteMs: &contract.NFRThreshold{Threshold: 5},
	}
	metrics := &RequestMetrics{RoundTripMs: 50, TimeToFirstByteMs: 20}
	checkNFR(&result, nfr, metrics, 0)

	if len(result.Failures) != 2 {
		t.Fatalf("expected 2 failures, got %d: %v", len(result.Failures), result.Failures)
	}
}

func TestCheckNFRNilIsNoOp(t *testing.T) {
	var result Result
	metrics := &RequestMetrics{RoundTripMs: 50, TimeToFirstByteMs: 20}
	checkNFR(&result, nil, metrics, 100)

	if len(result.Failures) != 0 {
		t.Errorf("expected no failures for nil NFR, got: %v", result.Failures)
	}
}

func TestVerifyNFRIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Write a body larger than threshold
		w.Write([]byte(`{"data": "this is a fairly long response body string"}`))
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "nfr test",
				Request:     contract.Request{Method: "GET", Path: "/test"},
				Response:    contract.Response{Status: 200},
				NFR: &contract.NFR{
					MaxResponseBytes: &contract.NFRThreshold{Threshold: 10},
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	if results[0].Passed {
		t.Error("expected failure when response exceeds max_response_bytes")
	}
}

func TestVerifyNFRWarningStillPasses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"data": "this is a fairly long response body string"}`))
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "nfr warning test",
				Request:     contract.Request{Method: "GET", Path: "/test"},
				Response:    contract.Response{Status: 200},
				NFR: &contract.NFR{
					MaxResponseBytes: &contract.NFRThreshold{Threshold: 10, Severity: "warning"},
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	if !results[0].Passed {
		t.Errorf("expected pass with warning-only NFR failure, got: %v", results[0].Failures)
	}
	if len(results[0].Failures) == 0 {
		t.Error("expected warning failure to be recorded")
	}
}

func TestSeverityZeroValueIsError(t *testing.T) {
	f := Failure{Field: "status", Message: "mismatch"}
	if f.Severity != SeverityError {
		t.Errorf("zero value Severity = %d, want SeverityError (0)", f.Severity)
	}
}

func TestVerifyWithNewMatchTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"count":      42,
			"score":      7.5,
			"message":    "hello world",
			"created_at": "2024-01-15T10:30:00Z",
			"status":     "active",
			"id":         999,
		})
	}))
	defer server.Close()

	minVal := 1.0
	maxVal := 100.0

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "new match types",
				Request:     contract.Request{Method: "GET", Path: "/data"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"count":      10,
						"score":      5.0,
						"message":    "placeholder",
						"created_at": "2024-01-01T00:00:00Z",
						"status":     "pending",
						"id":         1,
					},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.count":      {Match: "min", Min: &minVal},
					"$.body.score":      {Match: "max", Max: &maxVal},
					"$.body.message":    {Match: "includes", Includes: "world"},
					"$.body.created_at": {Match: "datetime"},
					"$.body.status":     {Match: "enum", Values: []string{"active", "inactive", "pending"}},
					"$.body.id":         {Match: "not_null"},
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	if !results[0].Passed {
		t.Errorf("expected pass with new match types, got failures: %v", results[0].Failures)
	}
}

func TestVerifyNewMatchTypesFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"count":   0,
			"message": "goodbye",
			"status":  "deleted",
		})
	}))
	defer server.Close()

	minVal := 5.0

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "new match types failure",
				Request:     contract.Request{Method: "GET", Path: "/data"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"count":   10,
						"message": "placeholder",
						"status":  "pending",
					},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.count":   {Match: "min", Min: &minVal},
					"$.body.message": {Match: "includes", Includes: "world"},
					"$.body.status":  {Match: "enum", Values: []string{"active", "inactive"}},
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	if results[0].Passed {
		t.Error("expected failure for new match types")
	}
	if len(results[0].Failures) < 3 {
		t.Errorf("expected at least 3 failures, got %d: %v", len(results[0].Failures), results[0].Failures)
	}
}

func TestVerifySpecificIndexOverridesWildcard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"items": []any{
				map[string]any{"id": 42},
				map[string]any{"id": 99},
			},
		})
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "specific index overrides wildcard",
				Request:     contract.Request{Method: "GET", Path: "/items"},
				Response: contract.Response{
					Status: 200,
					Body: map[string]any{
						"items": []any{
							map[string]any{"id": 42},
							map[string]any{"id": 1},
						},
					},
				},
				MatchingRules: contract.MatchingRules{
					"$.body.items[*].id": {Match: "type"},
					"$.body.items[0].id": {Match: "exact"}, // specific index takes priority
				},
			},
		},
	}

	results := Verify(c, server.URL, 30*time.Second)
	// items[0].id should use exact (42 == 42, pass)
	// items[1].id should use wildcard type (99 is number like 1, pass)
	if !results[0].Passed {
		t.Errorf("expected pass, got failures: %v", results[0].Failures)
	}
}

func TestVerifyTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := &contract.Contract{
		Interactions: []contract.Interaction{
			{
				Description: "slow endpoint",
				Request:     contract.Request{Method: "GET", Path: "/slow"},
				Response:    contract.Response{Status: 200},
			},
		},
	}

	results := Verify(c, server.URL, 50*time.Millisecond)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Passed {
		t.Error("expected failure due to timeout")
	}
	if len(results[0].Failures) == 0 {
		t.Fatal("expected at least one failure")
	}
	if results[0].Failures[0].Field != "request" {
		t.Errorf("failure field = %q, want %q", results[0].Failures[0].Field, "request")
	}
}
