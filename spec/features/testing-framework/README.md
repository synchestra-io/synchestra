# Feature: Testing Framework

**Status:** Conceptual

## Summary

Synchestra's testing framework turns specifications into executable verification — without leaving markdown. Acceptance criteria define what "correct" means for each feature. Test scenarios compose those criteria into multi-step workflows. The runner executes everything and reports results.

The full specification for this feature lives in the [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/) repository, where it is developed as an independent product. Synchestra integrates Rehearse as its testing framework.

## Sub-features

The testing framework defines two sub-features, both fully specified in the Rehearse repository:

| Sub-feature | Description | Full Specification |
|---|---|---|
| test-scenario | The markdown scenario format: steps, outputs, AC references, includes | [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-scenario/) |
| test-runner | The Go execution engine: parsing, AC resolution, shell execution, reporting | [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-runner/) |

### test-scenario

Defines the markdown scenario format — a human-readable `.md` file with named steps, dependency declarations, input/output passing, AC references, and sub-flow includes. Steps execute sequentially by default with opt-in parallel groups. Every scenario doubles as documentation: a product owner reads the step descriptions; the runner executes the bash blocks. Same file, two audiences.

### test-runner

The execution engine that brings scenarios to life. Parses scenario markdown, resolves AC verification scripts from feature `_acs/` directories, executes bash steps, and produces structured pass/fail reports for both humans and CI. Self-contained Go package (`pkg/testscenario/`) with no Synchestra-specific dependencies — give it a spec root path and it handles the rest.

## Synchestra Integration

Synchestra exposes the testing framework through its CLI:

```
synchestra test run [path]                  — run scenario file or directory
synchestra test run --tag e2e               — filter by tag
synchestra test run --format json           — machine-readable output
synchestra test run --run-manual-tests      — include scenarios tagged 'manual'
synchestra test list                        — list available scenarios
synchestra test list --tag e2e              — list filtered by tag
```

These commands delegate to the Rehearse test runner under the hood. The spec root is resolved from `project_dirs.specifications` in `synchestra-spec.yaml` (default: `spec`).

## Self-Testing

The testing framework tests itself — the runner's own acceptance criteria and dogfood scenarios are executed by the runner it verifies. Synchestra inherits this capability through its integration with Rehearse.

See [`cli/test`](../cli/test/README.md) for how to run Synchestra's own test scenarios and the runner's self-tests via the `synchestra test` command.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Acceptance Criteria](../acceptance-criteria/README.md) | ACs are the atomic verification units that scenarios compose. The runner resolves and executes their verification scripts. |
| [Feature](../feature/README.md) | Features gain `_tests/` directories for feature-scoped test scenarios. |
| [Development Plan](../development-plan/README.md) | Plan step ACs can reference feature ACs; scenarios verify both during and after implementation. |
| [CLI](../cli/README.md) | New `synchestra test` command group: `run`, `list`. |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should the framework support test fixtures or shared setup beyond sub-flows (e.g., a `spec/tests/fixtures/` directory for static test data)?
- Should there be a `spec/tests/config.yaml` for framework-level settings (default timeouts, parallelism limits, reporter format)?
