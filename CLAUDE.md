# CLAUDE.md - accord

Consumer-driven contract testing using plain YAML files.

## Core Standards

- **Module**: `github.com/andrewh/accord`
- **Binary**: `build/accord` (always build to `build/` directory)
- **Go**: Modern constructs (`any`, `context.Context`)
- **Error Handling**: Wrap errors with `%w`, use sentinel errors
- **Tone**: Professional, concise, no emojis
- **Commits**: Single task per commit, no Claude attribution

## Quick Commands

- **Build**: `make build`
- **Test**: `make test`
- **Lint**: `make lint`

## File Paths & Structure

```
build/              # Built binaries
cmd/accord/         # CLI entry point
internal/
  cli/              # Cobra commands (root, lint, verify, convert, generate, version)
  contract/         # Contract types, YAML parsing, path resolution, matchers
  lint/             # Lint engine and rule implementations
  verify/           # HTTP client and response comparison
  convert/          # Pact v2/v3 to Accord conversion
  generate/         # OpenAPI to Accord contract generation
  openapi/          # OpenAPI spec parsing
docs/               # Getting started guide, CLI reference
testdata/           # Valid and invalid contract fixtures
```

## Code Quality

- Use constants for often-used string values to prevent typos
- No magic numbers; use named constants or variables
- No descriptive single-line comments
- Always ensure tests pass before committing
- YAML parsing uses two passes: structured (Go types) and node (`yaml.Node` for line/column positions)
