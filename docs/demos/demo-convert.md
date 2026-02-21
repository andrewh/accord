# Converting Pact Contracts to Accord

*2026-02-12T13:22:15Z*

Teams migrating from Pact to Accord would need to rewrite every contract by hand. The `convert` command reads Pact v2 or v3 JSON files and produces valid Accord YAML, preserving interactions, matching rules, and query parameters.

## A Pact v2 Contract

Here's a typical Pact v2 contract with matching rules and a query string:

```bash
cat testdata/pact/v2_with_matching.json
```

```output
{
  "consumer": {
    "name": "web-app"
  },
  "provider": {
    "name": "user-api"
  },
  "interactions": [
    {
      "description": "a request for users",
      "request": {
        "method": "GET",
        "path": "/users",
        "query": "page=1&size=10",
        "headers": {
          "Accept": "application/json"
        }
      },
      "response": {
        "status": 200,
        "headers": {
          "Content-Type": "application/json"
        },
        "body": {
          "id": 1,
          "name": "Jane Doe",
          "email": "jane@example.com"
        },
        "matchingRules": {
          "$.body.id": {
            "match": "type"
          },
          "$.body.name": {
            "match": "type"
          },
          "$.body.email": {
            "match": "regex",
            "regex": "^[^@]+@[^@]+$"
          },
          "$.header.Content-Type": {
            "match": "regex",
            "regex": "application/json.*"
          }
        }
      }
    }
  ],
  "metadata": {
    "pactSpecification": {
      "version": "2.0.0"
    }
  }
}
```

## Converting to Accord

Use `--dry-run` to preview the output without writing files:

```bash
./accord convert --dry-run testdata/pact/v2_with_matching.json
```

```output
# web-app--user-api.yaml
accord: "0.1"
consumer:
    name: web-app
provider:
    name: user-api
interactions:
    - description: a request for users
      request:
        method: GET
        path: /users
        headers:
            Accept: application/json
        query:
            page: "1"
            size: "10"
        body: null
      response:
        status: 200
        headers:
            Content-Type: application/json
        body:
            email: jane@example.com
            id: 1
            name: Jane Doe
      matching_rules:
        $.body.email:
            match: regex
            regex: ^[^@]+@[^@]+$
        $.body.id:
            match: type
        $.body.name:
            match: type
        $.headers.Content-Type:
            match: regex
            regex: application/json.*

```

Several things happened automatically: the v2 query string `page=1&size=10` was split into a map, the flat matching rule paths were preserved (`$.body.id` stays as-is), and the header path `$.header.Content-Type` was translated to Accord's `$.headers.Content-Type` convention.

## Pact v3 with Categorised Matching Rules

Pact v3 organises matching rules into categories (`body`, `header`, etc.) instead of flat paths. The converter handles both formats:

```bash
./accord convert --dry-run testdata/pact/v3_with_matching.json
```

```output
# web-app--user-api.yaml
accord: "0.1"
consumer:
    name: web-app
provider:
    name: user-api
interactions:
    - description: a request for a user
      request:
        method: GET
        path: /users/1
        headers:
            Accept: application/json
        query: {}
        body: null
      response:
        status: 200
        headers:
            Content-Type: application/json
        body:
            active: true
            email: jane@example.com
            id: 1
            name: Jane Doe
      matching_rules:
        $.body.email:
            match: regex
            regex: ^[^@]+@[^@]+$
        $.body.id:
            match: type
        $.body.name:
            match: type
        $.headers.Content-Type:
            match: regex
            regex: application/json.*

```

The v3 categorised paths (`body` + `$.id`) are merged into Accord's flat format (`$.body.id`), and `header` + `Content-Type` becomes `$.headers.Content-Type`. The output is identical regardless of whether the input is v2 or v3.

## Warnings for Unsupported Features

Pact features like provider states, generators, and messages have no Accord equivalent. The converter skips them and prints warnings to stderr:

```bash
./accord convert --dry-run testdata/pact/v3_unsupported.json 2>&1 | head -5
```

```output
warning: testdata/pact/v3_unsupported.json: Pact messages are not supported and were skipped
warning: testdata/pact/v3_unsupported.json: interaction "a request with provider states": providerStates are not supported and were skipped
warning: testdata/pact/v3_unsupported.json: interaction "a request with provider states": generators are not supported and were skipped
# web-app--user-api.yaml
accord: "0.1"
```

The contract is still generated - only the unsupported features are dropped. This lets you migrate the bulk of your contracts and handle the gaps manually.

## Writing Files

Without `--dry-run`, the command writes files to disk. The filename follows the `{consumer}--{provider}.yaml` convention:

```bash
mkdir -p /tmp/accord-convert-demo && ./accord convert --output-dir /tmp/accord-convert-demo testdata/pact/v2_basic.json && ls /tmp/accord-convert-demo/
```

```output
wrote /tmp/accord-convert-demo/web-app--user-api.yaml
web-app--user-api.yaml
```

## Round-Trip: Convert then Lint

Converted contracts are valid by construction. We can prove this by linting the output:

```bash
mkdir -p /tmp/accord-convert-demo2 && ./accord convert --output-dir /tmp/accord-convert-demo2 testdata/pact/v2_with_matching.json && ./accord lint /tmp/accord-convert-demo2/web-app--user-api.yaml && echo 'exit code: 0'
```

```output
wrote /tmp/accord-convert-demo2/web-app--user-api.yaml
exit code: 0
```

No lint errors - the converted contract passes all validation rules. From here you can edit the contract to adjust matching rules, add NFR thresholds, or remove interactions you don't need.

## Multiple Files

You can convert several Pact files in one invocation:

```bash
mkdir -p /tmp/accord-convert-demo3 && ./accord convert --output-dir /tmp/accord-convert-demo3 testdata/pact/v2_basic.json testdata/pact/v3_with_matching.json && ls /tmp/accord-convert-demo3/
```

```output
wrote /tmp/accord-convert-demo3/web-app--user-api.yaml
wrote /tmp/accord-convert-demo3/web-app--user-api.yaml
web-app--user-api.yaml
```

Both files had the same consumer and provider names, so the second write overwrote the first. In practice, each Pact file typically represents a different consumer-provider pair and will produce a separate output file.
