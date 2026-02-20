# CLI Reference

## `accord lint <files...>`

Validates contract files and reports errors and warnings with file, line, and
column numbers.

```
$ accord lint contracts/user.yaml
contracts/user.yaml:12:5: error: request.method is required
contracts/user.yaml:25:9: warning: matching rule path "body.id" should start with "$."
```

### Lint rules

- Required fields: `accord`, `consumer.name`, `provider.name`, at least one
  interaction
- Each interaction: `description`, `request.method`, `request.path`,
  `response.status`
- Valid HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- Valid status code (100–599)
- No duplicate interaction descriptions
- Matching rule paths start with `$.`
- Matching rule `match` value is a recognised type
- If `match: regex`, `regex` field must be present and compilable

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed |
| 1 | Errors found |
| 2 | Usage error |

## `accord verify <files...> --provider-url <url>`

Sends contract interactions to a running provider and verifies the responses
match.

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

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--provider-url` | (required) | Base URL of the running provider |
| `--timeout` | `30` | HTTP request timeout in seconds |

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | All interactions passed |
| 1 | Failures found |
| 2 | Usage error |

## `accord generate <openapi-spec>`

Reads an OpenAPI specification and generates starter Accord contract files with
sensible defaults and matching rules.

```
$ accord generate api/openapi.yaml --consumer order-service
wrote ./order-service--user-service.yaml
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--consumer` | `my-service` | Consumer service name |
| `--endpoints` | (all) | Glob pattern to filter endpoint paths |
| `--output-dir` | `.` | Output directory for generated files |
| `--dry-run` | `false` | Print generated contracts to stdout instead of writing files |

## `accord convert <pact-files...>`

Reads Pact v2 or v3 JSON contract files and converts them to Accord YAML
format.

```
$ accord convert pacts/order-user.json --output-dir contracts/
wrote contracts/order-service--user-service.yaml
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output-dir` | `.` | Output directory for converted files |
| `--dry-run` | `false` | Print converted contracts to stdout instead of writing files |

## `accord version`

Prints version information.

```
$ accord version
accord v0.1.0
```
