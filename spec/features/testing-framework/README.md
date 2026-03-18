# Feature: Testing Framework

**Status:** Conceptual

## Summary

Synchestra's testing framework turns specifications into executable verification — without leaving markdown. Acceptance criteria define what "correct" means for each feature. Test scenarios compose those criteria into multi-step workflows. The runner executes everything and reports results.

The full specification for this feature lives in the [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/) repository, where it is developed as an independent product. Synchestra integrates Rehearse as its testing framework.

## Sub-features

The test-scenario format and test-runner engine are defined and developed in [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/). Synchestra does not duplicate those sub-features — see the canonical specs:

- [test-scenario](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-scenario/) — the markdown scenario format
- [test-runner](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-runner/) — the Go execution engine

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

These commands delegate to the Rehearse test runner under the hood. The spec root is resolved from `project_dirs.specifications` in `synchestra-spec-repo.yaml` (default: `spec`).

## Self-Testing

The `synchestra test` command's own [acceptance criteria](../cli/test/_acs/) and [dogfood scenarios](../cli/test/_tests/) are executed by the runner it wraps.

Run the self-tests:

```bash
# Dogfood scenario — exercises parsing, execution, outputs, AC resolution
go run . test run spec/features/cli/test/_tests/runner-core.md

# All test scenarios including demos
go run . test run spec/features/cli/test/_tests/ --run-manual-tests
```

See [`cli/test`](../cli/test/README.md) for the full `synchestra test` command reference.

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
