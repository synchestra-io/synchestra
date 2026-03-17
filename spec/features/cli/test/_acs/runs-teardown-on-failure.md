# AC: runs-teardown-on-failure

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

When a step fails (non-zero exit), subsequent steps continue executing (unless
`--fail-fast`), and Teardown always runs regardless of step failures. The Teardown
block's own exit code does not mask the step failure in the final result.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario where a middle step fails but Teardown writes a marker file |
| binary_path | Yes | Path to the compiled `synchestra` binary |
| marker_file | Yes | Path where Teardown writes a file to prove it ran |

## Verification

```bash
rm -f "$marker_file"

output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?

# Scenario should report failure (step failed)
test $rc -ne 0 || { echo "Expected non-zero exit due to step failure"; exit 1; }

# Teardown should have run (marker file exists)
test -f "$marker_file" || {
  echo "Teardown did not run — marker file '$marker_file' not found"
  exit 1
}

# Verify the overall result reflects the step failure, not Teardown status
failed_steps=$(echo "$output" | jq '[.steps[] | select(.status == "failed")] | length')
test "$failed_steps" -gt 0 || { echo "No failed steps in report"; exit 1; }
```

## Scenarios

(None yet.)
