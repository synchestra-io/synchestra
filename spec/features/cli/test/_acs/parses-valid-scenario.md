# AC: parses-valid-scenario

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

Given a well-formed scenario markdown file with a title, description, tags, Setup,
named steps (with outputs and AC references), and Teardown, the runner parses it
without error and the `--dry-run` output (or JSON report) reflects the correct
structure: scenario name, step names in file order, declared outputs, and AC references.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a valid scenario `.md` file |
| binary_path | Yes | Path to the compiled `synchestra` binary |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Expected exit 0, got $rc"; echo "$output"; exit 1; }

# Verify scenario name is present
echo "$output" | jq -e '.scenario.name' > /dev/null || { echo "Missing scenario name"; exit 1; }

# Verify steps are present and ordered
step_count=$(echo "$output" | jq '.steps | length')
test "$step_count" -gt 0 || { echo "No steps found"; exit 1; }
```

## Scenarios

(None yet.)
