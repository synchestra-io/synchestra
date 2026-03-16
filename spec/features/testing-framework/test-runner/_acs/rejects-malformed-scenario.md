# AC: rejects-malformed-scenario

**Status:** planned
**Feature:** [testing-framework/test-runner](../README.md)

## Description

Given a scenario file with structural errors (missing title, duplicate step names,
step with neither code block nor Include, or a Depends on referencing a non-existent
step), the runner exits with a non-zero code and the error message includes the
line number where the problem was detected.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a malformed scenario `.md` file |
| binary_path | Yes | Path to the compiled `synchestra` binary |
| expected_error | Yes | Substring expected in the error output |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" 2>&1)
rc=$?
test $rc -ne 0 || { echo "Expected non-zero exit, got 0"; exit 1; }

echo "$output" | grep -q "$expected_error" || {
  echo "Expected error containing '$expected_error', got:"
  echo "$output"
  exit 1
}

# Verify line number is present in error output
echo "$output" | grep -qE 'line [0-9]+' || {
  echo "Error output missing line number"
  echo "$output"
  exit 1
}
```

## Scenarios

(None yet.)
