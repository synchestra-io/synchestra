# Feature: CLI / test

**Status:** Conceptual

## Summary

The `synchestra test` command group runs and lists markdown-native test scenarios. It delegates to the [Rehearse](https://github.com/synchestra-io/rehearse) test runner under the hood, resolving the spec root from `project_dirs.specifications` in `synchestra-spec.yaml`.

## Commands

### `synchestra test run`

Executes one or more test scenarios.

```
synchestra test run [path]                  — run scenario file or directory
synchestra test run --tag e2e               — filter by tag
synchestra test run --format json           — machine-readable output
synchestra test run --run-manual-tests      — include scenarios tagged 'manual'
synchestra test run --spec-root ./my-spec   — override spec root directory
```

| Flag | Default | Description |
|---|---|---|
| `--format` | `text` | Output format: `text` (styled with live progress) or `json` |
| `--spec-root` | from `synchestra-spec.yaml` | Override the spec root directory |
| `--tag` | | Filter scenarios by tag (repeatable) |
| `--run-manual-tests` | `false` | Include scenarios tagged `manual` in directory scans |

When no path is given, defaults to `{spec_root}/tests/`.

### `synchestra test list`

Lists available scenarios without executing them.

```
synchestra test list                        — list all scenarios
synchestra test list --tag e2e              — list filtered by tag
```

## Running Synchestra's Own Tests

Synchestra uses [Rehearse](https://github.com/synchestra-io/rehearse) to test itself. The test runner is the same `pkg/testscenario` package that Rehearse provides.

### Run all test scenarios

```bash
go run . test run spec/tests/
```

### Run tests for the `test` command itself

The test runner's own acceptance criteria and dogfood scenarios verify the `synchestra test` command:

```bash
# Run the runner's self-test scenario (dogfood) — 8 steps, 8 ACs
go run . test run spec/features/testing-framework/test-runner/_tests/runner-core.md

# Run all test-runner scenarios including demos
go run . test run spec/features/testing-framework/test-runner/_tests/ --run-manual-tests
```

The runner tests itself — if it can parse and execute its own dogfood scenario, that is direct evidence of correctness.

### JSON output for CI

```bash
go run . test run spec/tests/ --format json
```

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Testing Framework](../../testing-framework/README.md) | `synchestra test` is the CLI entry point for the testing framework. |
| [CLI](../README.md) | Parent — follows the `synchestra <resource> <action>` pattern. |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should `synchestra test run` without arguments default to `spec/tests/` or the current directory?
- Should there be a `synchestra test init` command to scaffold example scenarios?
