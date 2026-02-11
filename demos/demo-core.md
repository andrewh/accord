# Accord: File-Based Contract Testing

*2026-02-11T10:51:02Z*

Accord is a contract testing tool that uses plain YAML files instead of requiring a broker server. This demo walks through the two core commands: `lint` and `verify`.

## Contract Format

A contract describes the interactions a consumer expects from a provider. Here's an example:

```bash
cat testdata/valid/user_service.yaml
```

```output
accord: "0.1"

consumer:
  name: "order-service"

provider:
  name: "user-service"

interactions:
  - description: "get a user by ID"
    request:
      method: GET
      path: /users/123
      headers:
        Accept: application/json
      query:
        include: "email"

    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        id: 123
        name: "Jane Doe"
        email: "jane@example.com"

    matching_rules:
      "$.body.id":
        match: type
      "$.body.name":
        match: type
      "$.body.email":
        match: regex
        regex: "^[^@]+@[^@]+$"
```

The matching rules control how response fields are compared. `type` accepts any value of the same JSON type, `regex` matches against a pattern, and `exact` (the default) requires an identical value.

## Linting

`accord lint` validates contract files and reports errors with file, line, and column numbers.

Linting a valid contract produces no output and exits 0:

```bash
./accord lint testdata/valid/user_service.yaml testdata/valid/minimal.yaml && echo 'exit code: 0'
```

```output
exit code: 0
```

Linting an invalid contract reports each problem with its location:

```bash
./accord lint testdata/invalid/missing_fields.yaml; echo 'exit code:' $?
```

```output
testdata/invalid/missing_fields.yaml:1:1: error: provider.name is required
testdata/invalid/missing_fields.yaml:4:3: error: consumer.name is required
testdata/invalid/missing_fields.yaml:7:5: error: description is required
testdata/invalid/missing_fields.yaml:8:5: error: request.method is required
testdata/invalid/missing_fields.yaml:11:7: error: response.status is required
exit code: 1
```

Lint catches structural issues, invalid HTTP methods, bad status codes, duplicate descriptions, and matching rule problems. Here's a contract with a bad matching rule:

```bash
./accord lint testdata/invalid/bad_matching.yaml; echo 'exit code:' $?
```

```output
testdata/invalid/bad_matching.yaml:19:7: warning: matching rule path "body.id" should start with "$."
testdata/invalid/bad_matching.yaml:19:7: error: unknown match type "fuzzy"
exit code: 1
```

## Verification

`accord verify` sends requests from the contract to a running provider and checks the responses. Let's start a simple test server and verify against it.

With a test server running on port 9876 that returns user data (id: 42, name: "Alice Smith", email: "alice@example.com"), we verify the contract. The contract expects exact values for status and headers, but uses `type` matching for id and name, and `regex` for email - so different values of the right shape will pass:

```bash
./accord verify --provider-url http://localhost:9876 testdata/valid/user_service.yaml
```

```output
Verifying testdata/valid/user_service.yaml (order-service -> user-service)
  PASS  get a user by ID

All interactions passed.
```

The contract expected id: 123 and name: "Jane Doe", but the server returned id: 42 and name: "Alice Smith". The verification still passed because the matching rules only check type (both are numbers, both are strings) and the email matched the regex pattern.

When a response doesn't match, accord reports each failure with expected and actual values:

```bash
./accord verify --provider-url http://localhost:9876 testdata/exact_match.yaml; echo 'exit code:' $?
```

```output
Verifying testdata/exact_match.yaml (strict-client -> user-service)
  FAIL  get a specific user
        body.id: expected 123 (int), got 42 (float64) (expected 123, got 42)
        body.name: expected Jane Doe (string), got Alice Smith (string) (expected Jane Doe, got Alice Smith)
        body.email: expected jane@example.com (string), got alice@example.com (string) (expected jane@example.com, got alice@example.com)
exit code: 1
```

Without matching rules, every field is compared by exact value. The same contract with `type` matching rules passes because the shapes match even though the values differ.

## Test Suite

All tests pass across four packages - unit tests, integration tests, and end-to-end tests:

```bash
go test ./... -count=1
```

```output
ok  	github.com/andrewh/accord	1.744s
?   	github.com/andrewh/accord/cmd/accord	[no test files]
?   	github.com/andrewh/accord/internal/cli	[no test files]
ok  	github.com/andrewh/accord/internal/contract	0.369s
ok  	github.com/andrewh/accord/internal/lint	0.631s
ok  	github.com/andrewh/accord/internal/verify	1.234s
```
