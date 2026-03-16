# AC: resolves-ac-specific

**Status:** planned
**Feature:** [testing-framework/test-runner](../README.md)

## Description

Given a step that references specific named ACs (not wildcard), the runner resolves
only those ACs from the feature's `_acs/` directory, executes them in the order
listed in the table, and does not execute other ACs in the same directory.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario with specific AC references |
| binary_path | Yes | Path to the compiled `synchestra` binary |
| expected_ac_name | Yes | Name of the AC that should be executed |
| excluded_ac_name | Yes | Name of an AC in _acs/ that should NOT be executed |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?

# Verify the expected AC was executed
echo "$output" | jq -e ".steps[].acs[]? | select(.name == \"$expected_ac_name\")" > /dev/null || {
  echo "Expected AC '$expected_ac_name' not found in results"
  exit 1
}

# Verify the excluded AC was NOT executed
match=$(echo "$output" | jq -r ".steps[].acs[]? | select(.name == \"$excluded_ac_name\") | .name")
test -z "$match" || {
  echo "AC '$excluded_ac_name' should not have been executed but was"
  exit 1
}
```

## Scenarios

(None yet.)
