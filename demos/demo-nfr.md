# Non-Functional Requirements in Contracts

*2026-02-11T20:56:30Z*

Accord contracts can now enforce non-functional requirements (NFRs) per interaction. Each NFR sets a threshold for response size or timing, with configurable severity that controls whether a breach is an error or a warning.

## Contract Format

The `nfr` block sits alongside `request` and `response` in each interaction. Three threshold types are supported: `max_response_bytes`, `max_time_to_first_byte_ms`, and `max_round_trip_ms`. Each takes a `threshold` (the not-to-exceed value) and an optional `severity` (defaults to `"error"`).

```bash
cat testdata/valid/nfr_example.yaml
```

```output
accord: "0.1"
consumer:
  name: "order-service"
provider:
  name: "user-service"
interactions:
  - description: "get user with NFR constraints"
    request:
      method: GET
      path: /users/123
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 4096
        severity: warning
      max_time_to_first_byte_ms:
        threshold: 200
      max_round_trip_ms:
        threshold: 500
```

Here, `max_response_bytes` is a warning (a breach won't fail verification), while the two timing thresholds default to error severity.

## Linting

`accord lint` validates NFR fields. Thresholds must be positive and severity must be `"error"` or `"warning"`.

```bash
./accord lint testdata/valid/nfr_example.yaml && echo 'exit code: 0'
```

```output
exit code: 0
```

A contract with a zero threshold and an invalid severity gets flagged:

```bash
cat > /tmp/bad_nfr.yaml << 'YAML'
accord: "0.1"
consumer:
  name: "a"
provider:
  name: "b"
interactions:
  - description: "bad nfr"
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 0
        severity: fatal
YAML
./accord lint /tmp/bad_nfr.yaml; echo "exit code: $?"
```

```output
/tmp/bad_nfr.yaml:15:9: error: threshold must be > 0 for max_response_bytes
/tmp/bad_nfr.yaml:16:9: error: invalid severity "fatal" for max_response_bytes (must be "error" or "warning")
exit code: 1
```

## Verification

During `accord verify`, the tool measures response size and timing, then checks them against NFR thresholds. Let's start a test server and see the three possible outcomes.

### PASS: All thresholds met

A small, fast response easily stays within generous limits:

```bash
python3 -c "
import sys; sys.path.insert(0, \"demos\")
from nfr_server import run_verify
run_verify(\"\"\"accord: \\\"0.1\\\"
consumer:
  name: test
provider:
  name: test
interactions:
  - description: small fast response
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 1000
      max_round_trip_ms:
        threshold: 5000
\"\"\")" 2>&1
```

```output
Verifying /dev/stdin (test -> test)
  PASS  small fast response

All interactions passed.
```

### WARN: Warning threshold breached

When a warning-severity threshold is exceeded, the interaction shows WARN but verification still passes (exit 0):

```bash
python3 -c "
import sys; sys.path.insert(0, \"demos\")
from nfr_server import run_verify
run_verify(\"\"\"accord: \\\"0.1\\\"
consumer:
  name: test
provider:
  name: test
interactions:
  - description: oversized response (warning)
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 5
        severity: warning
\"\"\")" 2>&1
```

```output
Verifying /dev/stdin (test -> test)
  WARN  oversized response (warning)
        [warning] nfr.max_response_bytes: threshold exceeded (expected <= 5, got 10)

All interactions passed.
```

The response was 10 bytes, exceeding the 5-byte warning threshold. The `[warning]` prefix makes it clear this is advisory.

### FAIL: Error threshold breached

When an error-severity threshold is exceeded (the default), verification fails with exit code 1:

```bash
python3 -c "
import sys; sys.path.insert(0, \"demos\")
from nfr_server import run_verify
run_verify(\"\"\"accord: \\\"0.1\\\"
consumer:
  name: test
provider:
  name: test
interactions:
  - description: oversized response (error)
    request:
      method: GET
      path: /test
    response:
      status: 200
    nfr:
      max_response_bytes:
        threshold: 5
\"\"\", \"exit code: EXIT\")" 2>&1
```

```output
Verifying /dev/stdin (test -> test)
  FAIL  oversized response (error)
        nfr.max_response_bytes: threshold exceeded (expected <= 5, got 10)
exit code: 1
```

Without a `severity: warning` override, the same threshold breach becomes a hard failure.

## Test Suite

All tests pass, including the new NFR unit, integration, and end-to-end tests:

```bash
go test ./... -count=1 2>&1 | sed "s/[0-9]*\.[0-9]*s/(cached)/g"
```

```output
ok  	github.com/andrewh/accord	(cached)
?   	github.com/andrewh/accord/cmd/accord	[no test files]
?   	github.com/andrewh/accord/internal/cli	[no test files]
ok  	github.com/andrewh/accord/internal/contract	(cached)
ok  	github.com/andrewh/accord/internal/lint	(cached)
ok  	github.com/andrewh/accord/internal/verify	(cached)
```
