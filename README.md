# Accord

File-based contract testing. Consumer-driven contracts using plain YAML files, distributed through existing mechanisms (git, CI artifacts, package registries). Zero external dependencies in CI - just a single binary and contract files.

## Installation

```
go install github.com/andrewh/accord/cmd/accord@latest
```

Or download a binary from the [releases page](https://github.com/andrewh/accord/releases).

## Contract Format

Contracts are YAML files describing interactions between a consumer and provider:

```yaml
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

### Matching Rules

Matching rules control how response fields are compared. Without a rule, fields are compared by exact value.

| Type    | Behaviour                         | Extra fields |
|---------|-----------------------------------|--------------|
| `exact` | Exact value match (default)       | none         |
| `type`  | Any value of the same JSON type   | none         |
| `regex` | Value matches regular expression  | `regex`      |

### Matching Rule Paths

Simple dot-notation paths into the response: `$.body.field.nested`, `$.headers.Content-Type`, `$.status`. Paths must start with `$.`.

## Commands

### `accord lint <files...>`

Validates contract files and reports errors and warnings with file, line, and column numbers.

```
$ accord lint contracts/user.yaml
contracts/user.yaml:12:5: error: request.method is required
contracts/user.yaml:25:9: warning: matching rule path "body.id" should start with "$."
```

Lint rules:
- Required fields: `accord`, `consumer.name`, `provider.name`, at least one interaction
- Each interaction: `description`, `request.method`, `request.path`, `response.status`
- Valid HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- Valid status code (100-599)
- No duplicate interaction descriptions
- Matching rule paths start with `$.`
- Matching rule `match` value is a recognised type
- If `match: regex`, `regex` field must be present and compilable

Exit codes: 0 = all checks passed, 1 = errors found, 2 = usage error.

### `accord verify <files...> --provider-url <url>`

Sends contract interactions to a running provider and verifies responses match.

```
$ accord verify contracts/user.yaml --provider-url http://localhost:8080
Verifying contracts/user.yaml (order-service -> user-service)
  PASS  get a user by ID
  PASS  create a user

All interactions passed.
```

For each interaction, the verifier:
1. Builds an HTTP request from the contract (method, path, headers, query, body)
2. Sends it to the provider at the given base URL
3. Compares status, headers (only those in the contract), and body fields
4. Applies matching rules where specified, exact match otherwise
5. Collects all failures per interaction (does not stop at the first)

Exit codes: 0 = all interactions passed, 1 = failures found, 2 = usage error.

### `accord version`

Prints version information.

## Architecture

```
accord/
  cmd/accord/main.go            CLI entrypoint
  internal/
    contract/
      contract.go                Contract types and YAML parsing
      matching.go                Path resolution and matchers
    lint/
      linter.go                  Lint engine: runs rules, collects diagnostics
      rules.go                   Individual lint rule implementations
    verify/
      verifier.go                HTTP client, response comparison
    cli/
      root.go                    Root cobra command
      lint.go                    lint subcommand
      verify.go                  verify subcommand
      version.go                 version subcommand
  testdata/
    valid/                       Valid contract fixtures
    invalid/                     Invalid contract fixtures
  e2e_test.go                    End-to-end tests
```

### Dependencies

| Dependency               | Purpose                          |
|--------------------------|----------------------------------|
| `gopkg.in/yaml.v3`      | YAML parsing with node positions |
| `github.com/spf13/cobra`| CLI framework                    |

Everything else is standard library.

### Contract Parsing

Two-pass YAML parsing:
1. **Structured parse** into Go types for working with contract data
2. **Node parse** (`yaml.Node` tree) preserved alongside, for lint diagnostics with line/column numbers

### Matching

Path resolution parses `$.body.x.y` into segments, walks the response body (which is `any` from YAML/JSON unmarshal) to extract the value, then applies the rule. Numeric types (int, float) are treated as equivalent for type matching.

## Development

```bash
go test ./...          # run all tests
go build ./cmd/accord/ # build the binary
```

## Licence

MIT
