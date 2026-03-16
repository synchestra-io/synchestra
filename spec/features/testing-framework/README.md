# Feature: Testing Framework

**Status:** Conceptual

## Summary

A markdown-native testing framework for composing [acceptance criteria](../acceptance-criteria/README.md) into multi-step integration and E2E test flows. The framework defines the scenario format, provides a Go-based test runner, and integrates with the CLI for execution.

## Contents

| Directory | Description |
|---|---|
| [test-scenario](test-scenario/README.md) | The markdown scenario format: steps, outputs, AC references, includes |
| [test-runner](test-runner/README.md) | The Go execution engine: parsing, AC resolution, shell execution, reporting |

### test-scenario

Defines the markdown-native scenario format — a human-readable `.md` file with named steps, dependency declarations, input/output passing, AC references, and sub-flow includes. Steps execute sequentially by default with opt-in parallel groups. The format is designed so that scenarios double as documentation: readable by humans, executable by the runner.

### test-runner

The Go package (`pkg/testscenario/`) that parses scenario files, resolves AC verification scripts from feature `_acs/` directories, executes bash steps, and reports results. The runner has no dependencies on Synchestra-specific code — it receives a configurable spec root path and resolves everything from the filesystem.

## Problem

Synchestra has feature specs with behavioral descriptions and development plans with acceptance criteria, but no structured way to:

- **Verify features work end-to-end.** Individual Go tests validate package-level behavior, but there is no harness for black-box testing of the compiled CLI binary across a multi-step workflow (create project → add repo → start container → verify → shut down → remove).
- **Reuse verification logic.** The same assertion ("deleted project is not in list") appears in multiple contexts — a feature-scoped test, an E2E lifecycle test, a regression suite. Without a reusable unit, each test re-implements the check. [Acceptance criteria](../acceptance-criteria/README.md) provide the reusable unit; this framework composes them into test flows.
- **Connect specs to tests.** Acceptance criteria in plans and features are prose today. There is no executable link from "this feature should do X" to "this script verifies X."

## Behavior

### File locations

| Location | Purpose |
|---|---|
| `spec/tests/` | Cross-feature E2E scenarios |
| `spec/tests/flows/` | Reusable sub-flows |
| `spec/features/{feature}/_tests/` | Feature-scoped integration tests |

Cross-feature scenarios test workflows that span multiple features (e.g., project lifecycle: create → configure → verify → remove). Feature-scoped tests focus on a single feature's behavior in isolation.

### CLI commands

```
synchestra test run [path]       — run scenario file or directory
synchestra test run --tag e2e    — filter by tag
synchestra test list             — list available scenarios
synchestra test list --tag e2e   — list filtered by tag
```

These follow the existing `synchestra <resource> <action>` command pattern.

### Configurable spec root

The spec root directory name is configurable via `project_dirs.specifications` in `synchestra-spec.yaml` (default: `spec`). All path resolution — scenario discovery, AC resolution, sub-flow includes — uses this configured root.

### Design principles

1. **Scenarios are documentation.** A scenario file should be readable as a step-by-step guide, not just machine input.
2. **ACs are the reusable unit.** Scenarios compose ACs; they don't re-implement assertions.
3. **No custom DSL.** The format is markdown with conventions, not a new language.
4. **Sequential by default.** Parallelism is opt-in and explicit.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Acceptance Criteria](../acceptance-criteria/README.md) | ACs are the atomic verification units that scenarios compose. The runner resolves and executes AC verification scripts. |
| [Feature](../feature/README.md) | Features gain `_tests/` directories for feature-scoped scenarios. |
| [Development Plan](../development-plan/README.md) | Plan step ACs can reference feature ACs; the test runner verifies both. |
| [CLI](../cli/README.md) | New `synchestra test` command group: `run`, `list`. |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should the framework support test fixtures or shared setup beyond sub-flows?
- Should there be a `spec/tests/config.yaml` for framework-level settings (timeouts, parallelism limits)?
