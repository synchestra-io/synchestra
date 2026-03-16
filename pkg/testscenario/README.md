# pkg/testscenario

Markdown-native test scenario runner. Parses `.md` scenario files into structured
step sequences, resolves acceptance criteria references, executes steps with
input/output passing, and reports results.

This package has no dependencies on Synchestra-specific code. It receives a
configurable spec root path and resolves AC references from the filesystem.

## Spec

See `docs/superpowers/specs/2026-03-16-e2e-testing-and-acceptance-criteria-design.md`

## Running Tests

### Go unit tests

```bash
go test ./pkg/testscenario/...
```

This includes `TestParseScenario_runnerCoreDogfood` which parses the actual
dogfood scenario and verifies all steps, nested code fences, and
Setup/Teardown are parsed correctly.

### Dogfood scenario

The test runner tests itself via the scenario at
`spec/features/testing-framework/test-runner/_tests/runner-core.md`.

```bash
go run . test run spec/features/testing-framework/test-runner/_tests/runner-core.md
```

See the
[Test Runner feature README](../../spec/features/testing-framework/test-runner/README.md#dogfooding)
for background on the bootstrap strategy.

### Demo scenarios (manual)

These are tagged `manual` and skipped during normal test runs. Run them
directly by path to see the live progress reporter in action:

```bash
# 4-second sleep step — watch the real-time progress indicator
go run . test run spec/features/testing-framework/test-runner/_tests/progress-demo.md

# Step failure — shows how errors are reported live
go run . test run spec/features/testing-framework/test-runner/_tests/error-demo.md
```

To include manual scenarios when running a directory:

```bash
go run . test run spec/features/testing-framework/test-runner/_tests/ --run-manual-tests
```

### CLI options

| Flag | Default | Description |
|---|---|---|
| `--format` | `text` | Output format: `text` (styled with live progress) or `json` |
| `--spec-root` | `spec` | Override the spec root directory |
| `--tag` | | Filter scenarios by tag (repeatable) |
| `--run-manual-tests` | `false` | Include scenarios tagged `manual` |

## Outstanding Questions

None at this time.
