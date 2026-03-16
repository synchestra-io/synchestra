# AC: executes-sequential-steps

**Status:** planned
**Feature:** [testing-framework/test-runner](../README.md)

## Description

Given a scenario with multiple steps (none marked Parallel), each step executes
only after the previous step completes. The output log shows steps in file order
and no two steps overlap in execution time.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario with sequential steps that write timestamps |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Scenario failed with exit $rc"; echo "$output"; exit 1; }

# Verify steps completed in order by checking result ordering
first_step=$(echo "$output" | jq -r '.steps[0].name')
second_step=$(echo "$output" | jq -r '.steps[1].name')
third_step=$(echo "$output" | jq -r '.steps[2].name')

test "$first_step" = "step-a" || { echo "Expected step-a first, got $first_step"; exit 1; }
test "$second_step" = "step-b" || { echo "Expected step-b second, got $second_step"; exit 1; }
test "$third_step" = "step-c" || { echo "Expected step-c third, got $third_step"; exit 1; }
```

## Scenarios

(None yet.)
