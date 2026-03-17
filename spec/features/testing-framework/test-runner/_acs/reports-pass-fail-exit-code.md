# AC: reports-pass-fail-exit-code

**Status:** planned
**Feature:** [testing-framework/test-runner](../README.md)

## Description

The runner exits 0 when all steps and ACs pass. It exits non-zero when any step
or AC fails. The text report includes per-step and per-AC pass/fail status. The
JSON report includes structured result data with step names, statuses, durations,
and AC results.

## Inputs

| Name | Required | Description |
|---|---|---|
| passing_scenario_path | Yes | Path to a scenario where all steps pass |
| failing_scenario_path | Yes | Path to a scenario where at least one step fails |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
# Test 1: All-pass scenario → exit 0
output=$("$binary_path" test run "$passing_scenario_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Expected exit 0 for passing scenario, got $rc"; exit 1; }

result=$(echo "$output" | jq -r '.result')
test "$result" = "passed" || { echo "Expected result 'passed', got '$result'"; exit 1; }

# Test 2: Failing scenario → exit non-zero
output=$("$binary_path" test run "$failing_scenario_path" --format json 2>&1)
rc=$?
test $rc -ne 0 || { echo "Expected non-zero exit for failing scenario"; exit 1; }

result=$(echo "$output" | jq -r '.result')
test "$result" = "failed" || { echo "Expected result 'failed', got '$result'"; exit 1; }

# Verify per-step results are present
step_count=$(echo "$output" | jq '.steps | length')
test "$step_count" -gt 0 || { echo "No step results in JSON output"; exit 1; }
```

## Scenarios

(None yet.)
