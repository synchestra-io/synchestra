# Scenario: Runner core behaviors

**Description:** Integration test of the test runner's core capabilities — parsing, sequential execution, parallel groups, context output propagation, AC resolution (wildcard and specific), Setup/Teardown lifecycle, and error reporting. This scenario is executed by the runner itself (dogfooding).
**Tags:** integration, runner, dogfood

## Setup

```bash
export BINARY_PATH="${BINARY_PATH:-$(go env GOPATH)/bin/synchestra}"
export SPEC_ROOT="$(git rev-parse --show-toplevel)/spec"
export FIXTURE_DIR=$(mktemp -d)

# Create a minimal feature with _acs/ for AC resolution tests
mkdir -p "$FIXTURE_DIR/features/test-fixture/_acs"

cat > "$FIXTURE_DIR/features/test-fixture/_acs/README.md" << 'ACREADME'
# Acceptance Criteria: test-fixture
| AC | Description | Status |
|---|---|---|
| [always-pass](always-pass.md) | Always passes | implemented |
| [check-input](check-input.md) | Checks an input var | implemented |
ACREADME

cat > "$FIXTURE_DIR/features/test-fixture/_acs/always-pass.md" << 'AC1'
# AC: always-pass
**Status:** implemented
**Feature:** [test-fixture](../README.md)
## Description
Always passes.
## Inputs
| Name | Required | Description |
|---|---|---|
## Verification
```bash
exit 0
```
## Scenarios
(None yet.)
AC1

cat > "$FIXTURE_DIR/features/test-fixture/_acs/check-input.md" << 'AC2'
# AC: check-input
**Status:** implemented
**Feature:** [test-fixture](../README.md)
## Description
Verifies that a specific input variable is set.
## Inputs
| Name | Required | Description |
|---|---|---|
| test_value | Yes | A value that should equal "hello" |
## Verification
```bash
test "$test_value" = "hello"
```
## Scenarios
(None yet.)
AC2
```

## build-binary

```bash
cd "$(git rev-parse --show-toplevel)"
go build -o "$BINARY_PATH" .
```

## parse-valid

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [parses-valid-scenario]($SPEC_ROOT/features/testing-framework/test-runner/_acs/parses-valid-scenario.md) |

```bash
# Create a valid scenario fixture
cat > "$FIXTURE_DIR/valid-scenario.md" << 'SCENARIO'
# Scenario: Valid test

**Description:** A minimal valid scenario.
**Tags:** fixture

## do-something

```bash
echo "hello"
```
SCENARIO

"$BINARY_PATH" test run "$FIXTURE_DIR/valid-scenario.md" --dry-run --format json
```

## reject-malformed

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [rejects-malformed-scenario]($SPEC_ROOT/features/testing-framework/test-runner/_acs/rejects-malformed-scenario.md) |

```bash
# Create a malformed scenario (duplicate step names)
cat > "$FIXTURE_DIR/malformed-scenario.md" << 'SCENARIO'
# Scenario: Bad test

**Description:** Has duplicate step names.

## same-name

```bash
echo "first"
```

## same-name

```bash
echo "duplicate"
```
SCENARIO

# Should fail — pass expected_error for the AC
export expected_error="duplicate"
export scenario_path="$FIXTURE_DIR/malformed-scenario.md"
export binary_path="$BINARY_PATH"
! "$BINARY_PATH" test run "$FIXTURE_DIR/malformed-scenario.md" 2>&1 | grep -qi "duplicate"
```

## test-sequential

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| sequential_order | context | `cat $STEP_STDOUT` |

```bash
# Create a scenario with 3 sequential steps that append to a file
cat > "$FIXTURE_DIR/sequential-scenario.md" << 'SCENARIO'
# Scenario: Sequential order

**Description:** Verifies sequential execution order.

## step-a

```bash
echo -n "A" >> "$FIXTURE_DIR/order.txt"
```

## step-b

```bash
echo -n "B" >> "$FIXTURE_DIR/order.txt"
```

## step-c

```bash
echo -n "C" >> "$FIXTURE_DIR/order.txt"
cat "$FIXTURE_DIR/order.txt"
```
SCENARIO

rm -f "$FIXTURE_DIR/order.txt"
"$BINARY_PATH" test run "$FIXTURE_DIR/sequential-scenario.md"
result=$(cat "$FIXTURE_DIR/order.txt")
test "$result" = "ABC" || { echo "Expected ABC, got $result"; exit 1; }
echo "$result"
```

## test-context-outputs

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [propagates-context-outputs]($SPEC_ROOT/features/testing-framework/test-runner/_acs/propagates-context-outputs.md) |

```bash
# Create a scenario where step-a writes to context, step-b reads it
cat > "$FIXTURE_DIR/context-scenario.md" << 'SCENARIO'
# Scenario: Context propagation

**Description:** Tests context output passing.

## write-value

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| test_value | context | `echo hello` |

```bash
echo "producing value"
```

## read-value

```bash
echo "received: ${{ context.test_value }}"
```
SCENARIO

export expected_value="hello"
export scenario_path="$FIXTURE_DIR/context-scenario.md"
"$BINARY_PATH" test run "$FIXTURE_DIR/context-scenario.md" --format json
```

## test-ac-wildcard

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [resolves-ac-wildcard]($SPEC_ROOT/features/testing-framework/test-runner/_acs/resolves-ac-wildcard.md) |

```bash
# Create a scenario that uses wildcard AC reference against our fixture feature
cat > "$FIXTURE_DIR/wildcard-scenario.md" << 'SCENARIO'
# Scenario: Wildcard ACs

**Description:** Tests wildcard AC resolution.

## run-with-wildcard

**ACs:**

| Feature | ACs |
|---|---|
| [test-fixture]($FIXTURE_DIR/features/test-fixture/) | * |

```bash
export test_value="hello"
echo "running step"
```
SCENARIO

export feature_path="$FIXTURE_DIR/features/test-fixture"
"$BINARY_PATH" test run "$FIXTURE_DIR/wildcard-scenario.md" --format json
```

## test-teardown-on-failure

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [runs-teardown-on-failure]($SPEC_ROOT/features/testing-framework/test-runner/_acs/runs-teardown-on-failure.md) |

```bash
export marker_file="$FIXTURE_DIR/teardown-ran.marker"

cat > "$FIXTURE_DIR/teardown-scenario.md" << 'SCENARIO'
# Scenario: Teardown on failure

**Description:** Verifies teardown runs when steps fail.

## failing-step

```bash
exit 1
```

## Teardown

```bash
touch "$FIXTURE_DIR/teardown-ran.marker"
```
SCENARIO

export scenario_path="$FIXTURE_DIR/teardown-scenario.md"
# Run and expect failure, but teardown should still run
"$BINARY_PATH" test run "$FIXTURE_DIR/teardown-scenario.md" --format json || true
test -f "$marker_file" || { echo "Teardown did not run"; exit 1; }
```

## test-exit-codes

**ACs:**

| Feature | ACs |
|---|---|
| [testing-framework/test-runner]($SPEC_ROOT/features/testing-framework/test-runner/) | [reports-pass-fail-exit-code]($SPEC_ROOT/features/testing-framework/test-runner/_acs/reports-pass-fail-exit-code.md) |

```bash
# Create passing scenario
cat > "$FIXTURE_DIR/pass-scenario.md" << 'SCENARIO'
# Scenario: All pass

**Description:** All steps pass.

## pass-step

```bash
exit 0
```
SCENARIO

# Create failing scenario
cat > "$FIXTURE_DIR/fail-scenario.md" << 'SCENARIO'
# Scenario: One fails

**Description:** One step fails.

## fail-step

```bash
exit 1
```
SCENARIO

export passing_scenario_path="$FIXTURE_DIR/pass-scenario.md"
export failing_scenario_path="$FIXTURE_DIR/fail-scenario.md"
export binary_path="$BINARY_PATH"

# Passing: exit 0
"$BINARY_PATH" test run "$FIXTURE_DIR/pass-scenario.md" --format json
pass_rc=$?
test $pass_rc -eq 0 || { echo "Expected exit 0 for passing, got $pass_rc"; exit 1; }

# Failing: exit non-zero
"$BINARY_PATH" test run "$FIXTURE_DIR/fail-scenario.md" --format json || true
fail_rc=$?
# Note: the || true above means we need to capture differently
"$BINARY_PATH" test run "$FIXTURE_DIR/fail-scenario.md" --format json; fail_rc=$?
test $fail_rc -ne 0 || { echo "Expected non-zero for failing"; exit 1; }
echo "Exit codes correct: pass=$pass_rc, fail=$fail_rc"
```

## Teardown

```bash
rm -rf "$FIXTURE_DIR"
```
