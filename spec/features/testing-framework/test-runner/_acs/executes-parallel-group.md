# AC: executes-parallel-group

**Status:** planned
**Feature:** [testing-framework/test-runner](../README.md)

## Description

Given a scenario where two consecutive steps are marked `Parallel: true`, both
steps start executing before either completes, and the next sequential step waits
for both to finish. Verified by comparing execution timestamps: parallel steps
overlap, the sequential step starts after both parallel steps end.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario with a parallel group (steps that sleep briefly) |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Scenario failed with exit $rc"; echo "$output"; exit 1; }

# Parallel steps should have overlapping start/end times
# The total duration should be less than the sum of individual step durations
# (i.e., parallel steps ran concurrently, not sequentially)
total_duration=$(echo "$output" | jq '.duration_ms')
step_b_duration=$(echo "$output" | jq '.steps[] | select(.name == "parallel-b") | .duration_ms')
step_c_duration=$(echo "$output" | jq '.steps[] | select(.name == "parallel-c") | .duration_ms')

sum_parallel=$((step_b_duration + step_c_duration))
test "$total_duration" -lt "$sum_parallel" || {
  echo "Steps did not run in parallel: total=${total_duration}ms, sum=${sum_parallel}ms"
  exit 1
}
```

## Scenarios

(None yet.)
