// Verification engine: sends requests to a provider and compares responses against contracts.
package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/andrewh/accord/internal/contract"
)

var indexPattern = regexp.MustCompile(`\[\d+\]`)

// Severity indicates whether a verification failure is an error or warning.
type Severity int

const (
	SeverityError   Severity = iota
	SeverityWarning
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
	Severity Severity
}

// hasErrors returns true if any failure has error severity.
func hasErrors(failures []Failure) bool {
	for _, f := range failures {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
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

	metrics, err := sendRequest(ix.Request, providerURL)
	if err != nil {
		result.Failures = append(result.Failures, Failure{
			Field:   "request",
			Message: fmt.Sprintf("failed to send request: %v", err),
		})
		return result
	}
	resp := metrics.Response
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

	// Read response body (needed for both body comparison and NFR byte count)
	bodyBytes, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		result.Failures = append(result.Failures, Failure{
			Field:   "body",
			Message: fmt.Sprintf("failed to read response body: %v", bodyErr),
		})
	}

	// Compare body
	if ix.Response.Body != nil && bodyErr == nil {
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

	// Check non-functional requirements
	var responseBytes int64
	if bodyErr == nil {
		responseBytes = int64(len(bodyBytes))
	}
	checkNFR(&result, ix.NFR, metrics, responseBytes)

	result.Passed = !hasErrors(result.Failures)
	return result
}

// checkNFR compares measured values against NFR thresholds and appends failures.
func checkNFR(result *Result, nfr *contract.NFR, metrics *RequestMetrics, responseBytes int64) {
	if nfr == nil {
		return
	}

	checkThreshold := func(t *contract.NFRThreshold, field string, actual int64) {
		if t == nil {
			return
		}
		if actual <= int64(t.Threshold) {
			return
		}
		sev := SeverityError
		if t.Severity == "warning" {
			sev = SeverityWarning
		}
		result.Failures = append(result.Failures, Failure{
			Field:    "nfr." + field,
			Expected: fmt.Sprintf("<= %d", t.Threshold),
			Actual:   fmt.Sprintf("%d", actual),
			Message:  "threshold exceeded",
			Severity: sev,
		})
	}

	checkThreshold(nfr.MaxResponseBytes, "max_response_bytes", responseBytes)
	checkThreshold(nfr.MaxTimeToFirstByteMs, "max_time_to_first_byte_ms", metrics.TimeToFirstByteMs)
	checkThreshold(nfr.MaxRoundTripMs, "max_round_trip_ms", metrics.RoundTripMs)
}

// indexToWildcard replaces numeric array indices with wildcards in a path.
// e.g. "$.body.users[0].email" -> "$.body.users[*].email"
func indexToWildcard(path string) string {
	return indexPattern.ReplaceAllString(path, "[*]")
}

// lookupRule finds a matching rule for a path, falling back to a wildcard
// variant if no exact match exists. Specific indices take priority.
func lookupRule(rules contract.MatchingRules, path string) (contract.MatchingRule, bool) {
	if rule, ok := rules[path]; ok {
		return rule, true
	}
	wildcard := indexToWildcard(path)
	if wildcard != path {
		if rule, ok := rules[wildcard]; ok {
			return rule, true
		}
	}
	return contract.MatchingRule{}, false
}

// compareBody recursively compares expected and actual body fields.
func compareBody(result *Result, expected, actual any, rules contract.MatchingRules, path string) {
	switch expectedTyped := expected.(type) {
	case map[string]any:
		compareBodyMap(result, expectedTyped, actual, rules, path)
	case []any:
		compareBodyArray(result, expectedTyped, actual, rules, path)
	default:
		compareBodyScalar(result, expected, actual, rules, path)
	}
}

func compareBodyMap(result *Result, expectedMap map[string]any, actual any, rules contract.MatchingRules, path string) {
	actualMap, ok := actual.(map[string]any)
	if !ok {
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
		rule, hasRule := lookupRule(rules, rulePath)
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
			compareBody(result, expectedVal, actualVal, rules, fieldPath)
		}
	}
}

func compareBodyArray(result *Result, expectedArr []any, actual any, rules contract.MatchingRules, path string) {
	actualArr, ok := actual.([]any)
	if !ok {
		result.Failures = append(result.Failures, Failure{
			Field:    path,
			Expected: "array",
			Actual:   fmt.Sprintf("%T", actual),
			Message:  "expected array, got different type",
		})
		return
	}

	if len(expectedArr) != len(actualArr) {
		result.Failures = append(result.Failures, Failure{
			Field:    path,
			Expected: fmt.Sprintf("array of length %d", len(expectedArr)),
			Actual:   fmt.Sprintf("array of length %d", len(actualArr)),
			Message:  "array length mismatch",
		})
		return
	}

	for i := range expectedArr {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		compareBody(result, expectedArr[i], actualArr[i], rules, elemPath)
	}
}

func compareBodyScalar(result *Result, expected, actual any, rules contract.MatchingRules, path string) {
	rulePath := "$." + path
	rule, hasRule := lookupRule(rules, rulePath)
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
}

// RequestMetrics holds the HTTP response and timing measurements.
type RequestMetrics struct {
	Response          *http.Response
	RoundTripMs       int64
	TimeToFirstByteMs int64
}

func sendRequest(req contract.Request, providerURL string) (*RequestMetrics, error) {
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

	var ttfb time.Duration
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(httpReq.Context(), trace))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	roundTrip := time.Since(start)
	return &RequestMetrics{
		Response:          resp,
		RoundTripMs:       roundTrip.Milliseconds(),
		TimeToFirstByteMs: ttfb.Milliseconds(),
	}, nil
}
