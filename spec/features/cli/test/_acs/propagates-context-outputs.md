# AC: propagates-context-outputs

**Status:** planned
**Feature:** [cli/test](../README.md)

## Description

When a step declares an output with Store=`context`, the extracted value is
available to all subsequent steps via `${{ context.name }}` substitution. The
runner resolves the variable reference before executing the step's code block.

## Inputs

| Name | Required | Description |
|---|---|---|
| scenario_path | Yes | Path to a scenario where step-a outputs a value to context and step-b reads it |
| binary_path | Yes | Path to the compiled `synchestra` binary |
| expected_value | Yes | The value step-a produces that step-b should receive |

## Verification

```bash
output=$("$binary_path" test run "$scenario_path" --format json 2>&1)
rc=$?
test $rc -eq 0 || { echo "Scenario failed with exit $rc"; echo "$output"; exit 1; }

# Verify step-b received the context value from step-a
step_b_stdout=$(echo "$output" | jq -r '.steps[] | select(.name == "read-value") | .stdout')
echo "$step_b_stdout" | grep -q "$expected_value" || {
  echo "Expected step-b to receive '$expected_value', got: $step_b_stdout"
  exit 1
}
```

## Scenarios

(None yet.)
