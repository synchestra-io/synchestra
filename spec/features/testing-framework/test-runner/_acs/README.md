# Acceptance Criteria: testing-framework/test-runner

Acceptance criteria for the [test runner](../README.md).

| AC | Description | Status |
|---|---|---|
| [parses-valid-scenario](parses-valid-scenario.md) | Valid scenario file parsed into structured result | planned |
| [rejects-malformed-scenario](rejects-malformed-scenario.md) | Malformed scenario rejected with line-number error | planned |
| [executes-sequential-steps](executes-sequential-steps.md) | Steps execute in file order by default | planned |
| [executes-parallel-group](executes-parallel-group.md) | Consecutive Parallel: true steps run concurrently | planned |
| [resolves-ac-wildcard](resolves-ac-wildcard.md) | Wildcard (*) resolves all ACs in feature _acs/ directory | planned |
| [resolves-ac-specific](resolves-ac-specific.md) | Named AC references resolve to correct _acs/ files | planned |
| [runs-setup-before-steps](runs-setup-before-steps.md) | Setup block runs before all steps | planned |
| [runs-teardown-on-failure](runs-teardown-on-failure.md) | Teardown runs even when steps fail | planned |
| [propagates-context-outputs](propagates-context-outputs.md) | Context-scoped outputs accessible to subsequent steps | planned |
| [reports-pass-fail-exit-code](reports-pass-fail-exit-code.md) | Exit 0 on all pass, non-zero on any failure | planned |
| [detects-include-cycles](detects-include-cycles.md) | Circular includes rejected at validation | planned |

## Outstanding Questions

None at this time.
