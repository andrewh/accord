// Verification engine: sends requests to a provider and compares responses against contracts.
package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrewh/accord/internal/contract"
)

// Result holds the outcome of verifying a single interaction.
type Result struct {
	Interaction string
	Passed      bool
	Failures    []Failure
}

// Failure describes a single verification mismatch.
type Failure struct {
	Field    string
	Expected string
	Actual   string
	Message  string
}

func (f Failure) String() string {
	return fmt.Sprintf("%s: %s (expected %s, got %s)", f.Field, f.Message, f.Expected, f.Actual)
}

// Verify runs all interactions in a contract against a live provider.
func Verify(c *contract.Contract, providerURL string) []Result {
	var results []Result
	for _, ix := range c.Interactions {
		results = append(results, verifyInteraction(ix, providerURL))
	}
	return results
}

func verifyInteraction(ix contract.Interaction, providerURL string) Result {
	result := Result{Interaction: ix.Description}

	resp, err := sendRequest(ix.Request, providerURL)
	if err != nil {
		result.Failures = append(result.Failures, Failure{
			Field:   "request",
			Message: fmt.Sprintf("failed to send request: %v", err),
		})
		return result
	}
	defer resp.Body.Close()

	// Compare status
	if ix.Response.Status != 0 {
		statusRule, hasRule := ix.MatchingRules["$.status"]
		if hasRule {
			if err := contract.ApplyRule(statusRule, ix.Response.Status, resp.StatusCode); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:    "status",
					Expected: fmt.Sprintf("%d", ix.Response.Status),
					Actual:   fmt.Sprintf("%d", resp.StatusCode),
					Message:  err.Error(),
				})
			}
		} else if resp.StatusCode != ix.Response.Status {
			result.Failures = append(result.Failures, Failure{
				Field:    "status",
				Expected: fmt.Sprintf("%d", ix.Response.Status),
				Actual:   fmt.Sprintf("%d", resp.StatusCode),
				Message:  "status code mismatch",
			})
		}
	}

	// Compare headers (only those specified in the contract)
	for key, expected := range ix.Response.Headers {
		actual := resp.Header.Get(key)
		rulePath := "$.headers." + key
		rule, hasRule := ix.MatchingRules[rulePath]
		if hasRule {
			if err := contract.ApplyRule(rule, expected, actual); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:    "headers." + key,
					Expected: expected,
					Actual:   actual,
					Message:  err.Error(),
				})
			}
		} else if actual != expected {
			result.Failures = append(result.Failures, Failure{
				Field:    "headers." + key,
				Expected: expected,
				Actual:   actual,
				Message:  "header value mismatch",
			})
		}
	}

	// Compare body
	if ix.Response.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			result.Failures = append(result.Failures, Failure{
				Field:   "body",
				Message: fmt.Sprintf("failed to read response body: %v", err),
			})
		} else {
			var actualBody any
			if err := json.Unmarshal(bodyBytes, &actualBody); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:   "body",
					Message: fmt.Sprintf("failed to parse response body as JSON: %v", err),
				})
			} else {
				compareBody(&result, ix.Response.Body, actualBody, ix.MatchingRules, "body")
			}
		}
	}

	result.Passed = len(result.Failures) == 0
	return result
}

// compareBody recursively compares expected and actual body fields.
func compareBody(result *Result, expected, actual any, rules contract.MatchingRules, path string) {
	expectedMap, isMap := expected.(map[string]any)
	if !isMap {
		// Scalar or array comparison at this level
		rulePath := "$." + path
		rule, hasRule := rules[rulePath]
		if hasRule {
			if err := contract.ApplyRule(rule, expected, actual); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:    path,
					Expected: fmt.Sprintf("%v", expected),
					Actual:   fmt.Sprintf("%v", actual),
					Message:  err.Error(),
				})
			}
		} else if err := contract.MatchExact(expected, actual); err != nil {
			result.Failures = append(result.Failures, Failure{
				Field:    path,
				Expected: fmt.Sprintf("%v", expected),
				Actual:   fmt.Sprintf("%v", actual),
				Message:  err.Error(),
			})
		}
		return
	}

	actualMap, isActualMap := actual.(map[string]any)
	if !isActualMap {
		result.Failures = append(result.Failures, Failure{
			Field:    path,
			Expected: "object",
			Actual:   fmt.Sprintf("%T", actual),
			Message:  "expected object, got different type",
		})
		return
	}

	for key, expectedVal := range expectedMap {
		fieldPath := path + "." + key
		actualVal, exists := actualMap[key]
		if !exists {
			result.Failures = append(result.Failures, Failure{
				Field:    fieldPath,
				Expected: fmt.Sprintf("%v", expectedVal),
				Actual:   "<missing>",
				Message:  "field not present in response",
			})
			continue
		}

		rulePath := "$." + fieldPath
		rule, hasRule := rules[rulePath]
		if hasRule {
			if err := contract.ApplyRule(rule, expectedVal, actualVal); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:    fieldPath,
					Expected: fmt.Sprintf("%v", expectedVal),
					Actual:   fmt.Sprintf("%v", actualVal),
					Message:  err.Error(),
				})
			}
		} else {
			// Recurse for nested objects, exact match for scalars
			_, isNestedMap := expectedVal.(map[string]any)
			if isNestedMap {
				compareBody(result, expectedVal, actualVal, rules, fieldPath)
			} else if err := contract.MatchExact(expectedVal, actualVal); err != nil {
				result.Failures = append(result.Failures, Failure{
					Field:    fieldPath,
					Expected: fmt.Sprintf("%v", expectedVal),
					Actual:   fmt.Sprintf("%v", actualVal),
					Message:  err.Error(),
				})
			}
		}
	}
}

func sendRequest(req contract.Request, providerURL string) (*http.Response, error) {
	fullURL := strings.TrimRight(providerURL, "/") + req.Path

	if len(req.Query) > 0 {
		params := url.Values{}
		for k, v := range req.Query {
			params.Set(k, v)
		}
		fullURL += "?" + params.Encode()
	}

	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(httpReq)
}
