# Getting Started with Accord

This guide walks through setting up contract tests between two real services. By the end you'll have a contract file that validates your provider's API on every CI run.

## Prerequisites

- A running provider service with an HTTP API
- The `accord` binary ([install instructions](../README.md#installation))

## Step 1: Identify the Interactions You Depend On

Contract testing is consumer-driven: you write contracts for the specific endpoints your consumer actually calls, not the provider's entire API surface.

If your provider has an OpenAPI spec, use it to identify the endpoints, methods, and response shapes your consumer depends on. You don't need to cover every endpoint - only the ones your consumer would break without.

For example, if your consumer is an order service that calls a user service, you might depend on:

- `GET /api/v1/users/{id}` - to look up the customer placing an order
- `GET /api/v1/users/{id}/addresses` - to get the shipping address

Your OpenAPI spec tells you the response schemas for these endpoints. Pull out the fields your consumer actually reads - those are what go in the contract.

## Step 2: Write a Contract

Create a YAML file describing each interaction. Name it after the relationship, e.g. `order-service--user-service.yaml`:

```yaml
accord: "0.1"

consumer:
  name: "order-service"

provider:
  name: "user-service"

interactions:
  - description: "look up customer by ID"
    request:
      method: GET
      path: /api/v1/users/42
      headers:
        Accept: application/json
        Authorization: "Bearer test-token"

    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        id: 42
        name: "Jane Doe"
        email: "jane@example.com"
        active: true

    matching_rules:
      "$.body.id":
        match: type
      "$.body.name":
        match: type
      "$.body.email":
        match: regex
        regex: "^[^@]+@[^@]+$"
      "$.body.active":
        match: type

  - description: "get customer shipping addresses"
    request:
      method: GET
      path: /api/v1/users/42/addresses
      headers:
        Accept: application/json
        Authorization: "Bearer test-token"

    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        addresses:
          - street: "123 Main St"
            city: "London"
            postcode: "EC1A 1BB"

    matching_rules:
      "$.body.addresses":
        match: type
```

### Choosing matching rules

Use your OpenAPI spec to decide which matching strategy fits each field:

| Your consumer needs...            | Use               | Example                              |
|-----------------------------------|--------------------|--------------------------------------|
| This exact value                  | no rule (default)  | A specific status code               |
| Any value of the right type       | `match: type`      | Any integer ID, any string name      |
| A value matching a pattern        | `match: regex`     | Email addresses, UUIDs, ISO dates    |

A good rule of thumb: use `type` for most body fields, `exact` for status codes and content types, and `regex` for fields with a known format. If your OpenAPI spec defines a `pattern` on a field, use that as your regex.

### Request details

Use realistic values in the request. The path, headers, and query parameters are sent literally to the provider during verification:

- **Path**: Use a real or test ID that exists in your test environment
- **Headers**: Include authentication tokens, content types, and any headers your consumer sends
- **Query parameters**: Include pagination, filters, or includes your consumer uses
- **Body**: For POST/PUT/PATCH, include a valid request body

## Step 3: Lint the Contract

Validate the contract before checking it in:

```
$ accord lint order-service--user-service.yaml
```

No output means the contract is valid. If there are problems, you'll see them with file, line, and column numbers:

```
order-service--user-service.yaml:15:7: error: invalid HTTP method: "GETT"
order-service--user-service.yaml:42:9: warning: matching rule path "body.id" should start with "$."
```

Fix any errors and re-lint until clean.

## Step 4: Verify Against the Provider

Start your provider in a test environment (or point at a running staging instance), then verify:

```
$ accord verify order-service--user-service.yaml --provider-url http://localhost:8080
Verifying order-service--user-service.yaml (order-service -> user-service)
  PASS  look up customer by ID
  PASS  get customer shipping addresses

All interactions passed.
```

If verification fails, you'll see exactly which fields didn't match:

```
  FAIL  look up customer by ID
        body.email: value "not-valid" does not match pattern "^[^@]+@[^@]+$" (expected jane@example.com, got not-valid)
        headers.Content-Type: header value mismatch (expected application/json, got text/html)
```

### Authentication

If your provider requires authentication, include the credentials in the contract's request headers. For test environments, use a dedicated test token:

```yaml
request:
  headers:
    Authorization: "Bearer test-token"
```

For CI, you can template the token from an environment variable before running accord, or configure your test environment to accept a fixed test token.

## Step 5: Add to CI

Add contract verification to your provider's CI pipeline. The provider runs verification to confirm it still satisfies its consumers' contracts:

```yaml
# Example GitHub Actions step
- name: Verify contracts
  run: |
    # Start the provider (adjust to your setup)
    ./start-provider.sh &
    sleep 5
    accord verify contracts/*.yaml --provider-url http://localhost:8080
```

### Where to store contracts

Contracts are plain files. Distribute them however you distribute code:

- **Same repo**: If consumer and provider are in a monorepo, keep contracts alongside the code
- **Dedicated repo**: A `contracts` repo that both teams contribute to
- **CI artifacts**: Publish contracts as build artifacts, download them in the provider's pipeline
- **Package registry**: Version contracts as a package that the provider's tests depend on

The key constraint is that the provider's CI can access the consumer's contract files at verification time.

## Step 6: Write Contracts for Each Consumer

Each consumer writes its own contract against the provider. If three services depend on the user service, you end up with three contract files:

```
contracts/
  order-service--user-service.yaml
  billing-service--user-service.yaml
  notification-service--user-service.yaml
```

The provider verifies all of them. If a provider change would break any consumer, verification catches it before it ships.

## Deriving Contracts from OpenAPI

If your provider publishes an OpenAPI spec, use it as a reference when writing contracts:

1. Find the endpoints your consumer calls in the spec
2. Note the response schema - which fields exist, their types, any patterns
3. Write the contract with only the fields your consumer reads
4. Use the OpenAPI `type` to choose matching rules (`string` → `type`, `string` + `pattern` → `regex`)
5. Use the OpenAPI `required` fields to decide which fields to include

You don't need to replicate the entire OpenAPI schema. Contracts are intentionally minimal - they describe what the consumer actually depends on, not everything the provider could return. A provider can add new fields freely; the contract only breaks if a field the consumer uses changes or disappears.
