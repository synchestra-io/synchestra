# AC: runs-setup-before-steps

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

The Setup block runs before all named steps. If Setup fails (non-zero exit),
no named steps execute, but Teardown still runs.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_pass_path | Yes | Path to a scenario where Setup succeeds and sets an env var used by steps |
| scenario_fail_path | Yes | Path to a scenario where Setup fails (exit 1) |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
# Test 1: Setup succeeds — steps should run
output=$("$binary_path" test run "$scenario_pass_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Expected pass scenario to succeed"; echo "$output"; exit 1; }

step_count=$(echo "$output" | jq '[.steps[] | select(.status == "passed")] | length')
test "$step_count" -gt 0 || { echo "No steps ran after successful Setup"; exit 1; }

# Test 2: Setup fails — steps should NOT run, Teardown should run
output=$("$binary_path" test run "$scenario_fail_path" --format json 2>&1)
rc=$?
test $rc -ne 0 || { echo "Expected fail scenario to fail"; exit 1; }

skipped=$(echo "$output" | jq '[.steps[] | select(.status == "skipped")] | length')
test "$skipped" -gt 0 || { echo "Steps should be skipped when Setup fails"; exit 1; }

teardown_ran=$(echo "$output" | jq -r '.teardown.status')
test "$teardown_ran" = "passed" -o "$teardown_ran" = "failed" || {
  echo "Teardown should run even when Setup fails, got status: $teardown_ran"
  exit 1
}
```

## Scenarios

(None yet.)
