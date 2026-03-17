# AC: detects-include-cycles

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

When a scenario includes a sub-flow that directly or transitively includes itself
(A → B → A), the runner detects the cycle at validation time (before any execution)
and exits with a non-zero code and an error message naming the files involved in
the cycle.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario that includes a sub-flow with a circular reference |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" 2>&1)
rc=$?

# Should fail
test $rc -ne 0 || { echo "Expected non-zero exit for circular include"; exit 1; }

# Error should mention cycle/circular
echo "$output" | grep -qiE 'circular|cycle' || {
  echo "Error should mention circular/cycle, got:"
  echo "$output"
  exit 1
}
```

## Scenarios

(None yet.)
