# AC: resolves-ac-wildcard

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

Given a step that references `*` for a feature's ACs, the runner discovers all
`.ac.md` files in the feature's `_acs/` directory, extracts their
verification scripts, and executes them in alphabetical order. The report lists
each AC individually with pass/fail status.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario with a wildcard AC reference |
| binary_path | Yes | Path to the compiled `synchestra` binary |
| feature_path | Yes | Path to the feature directory containing `_acs/` with at least 2 AC files |
| spec_root | Yes | Spec root directory for the scenario under test |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --spec-root "$spec_root" --format json 2>&1)
rc=$?

# Count AC files in the feature's _acs/ directory
expected_ac_count=$(find "$feature_path/_acs" -name '*.ac.md' | wc -l | tr -d ' ')

# Count ACs reported in the step results
reported_ac_count=$(echo "$output" | jq '[.steps[].acs[]? | select(.feature)] | length')

test "$reported_ac_count" -eq "$expected_ac_count" || {
  echo "Expected $expected_ac_count ACs, runner reported $reported_ac_count"
  exit 1
}
```

## Scenarios

(None yet.)
