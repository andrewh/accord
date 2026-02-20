# Accord

[![CI](https://github.com/andrewh/accord/actions/workflows/ci.yaml/badge.svg)](https://github.com/andrewh/accord/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/andrewh/accord)](https://goreportcard.com/report/github.com/andrewh/accord)
[![Go Reference](https://pkg.go.dev/badge/github.com/andrewh/accord.svg)](https://pkg.go.dev/github.com/andrewh/accord)

Simple contract testing at scale.

Consumer-driven contracts using plain YAML files, distributed through existing
mechanisms (git, CI artifacts, package registries). Zero external dependencies
in CI — just a single binary and contract files.

## Install

```sh
go install github.com/andrewh/accord/cmd/accord@latest
```

Or download a binary from the [releases page](https://github.com/andrewh/accord/releases).

## Quick start

```yaml
# order-service--user-service.yaml
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

```sh
# Validate the contract
accord lint order-service--user-service.yaml

# Verify against a running provider
accord verify order-service--user-service.yaml --provider-url http://localhost:8080
```

## What it does

Accord reads a YAML contract file describing the interactions a consumer
expects from a provider. It can lint contracts for correctness, verify them
against a running provider, generate contracts from OpenAPI specs, and convert
Pact contracts to Accord format.

Use cases:

- **Catch breaking changes** — verify provider APIs still satisfy consumer
  contracts before shipping
- **Consumer-driven** — each consumer declares exactly what it needs, nothing
  more
- **CI-friendly** — a single binary with no external dependencies; contracts
  are plain files you already know how to distribute
- **Migrate from Pact** — `accord convert` imports Pact v2/v3 JSON contracts
- **Bootstrap from OpenAPI** — `accord generate` creates starter contracts
  from an OpenAPI spec

## Commands

See the [CLI reference](docs/reference/cli.md) for full details. Summary:

| Command | Description |
|---------|-------------|
| `accord lint <files...>` | Validate contract files |
| `accord verify <files...> --provider-url <url>` | Verify contracts against a running provider |
| `accord generate <openapi-spec>` | Generate contracts from an OpenAPI spec |
| `accord convert <pact-files...>` | Convert Pact contracts to Accord format |
| `accord version` | Print version information |

## Contract format

### Matching rules

Matching rules control how response fields are compared. Without a rule, fields
are compared by exact value.

| Type    | Behaviour                         | Extra fields |
|---------|-----------------------------------|--------------|
| `exact` | Exact value match (default)       | none         |
| `type`  | Any value of the same JSON type   | none         |
| `regex` | Value matches regular expression  | `regex`      |

### Matching rule paths

Simple dot-notation paths into the response: `$.body.field.nested`,
`$.headers.Content-Type`, `$.status`. Paths must start with `$.`.

## Documentation

- [Getting started](docs/getting-started.md) — set up contract tests between
  two services
- [CLI reference](docs/reference/cli.md) — all commands, flags, and exit codes

## Development

```sh
make build   # build the binary
make test    # run all tests
make lint    # check formatting and vet
```

## Licence

[Apache 2.0](LICENSE)
