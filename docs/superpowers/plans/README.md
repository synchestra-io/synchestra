# Implementation Plans

**Moved to `spec/plans/`**

All implementation plans are now created as formal plans in the `spec/plans/` directory and follow the [Plan specification](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md).

Plans:
- Start in `draft` status and progress through review to `approved`
- Are mutable; snapshots (git hash + action + comment) capture reference points
- Support recursive task decomposition with dependencies and acceptance criteria
- Generate executable tasks in the state store

To create or manage plans, see:
- [Plan specification](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md#behavior) — structure and format
- [CLI commands](../cli/plan.md) — `synchestra plan create`, `submit`, `approve`, etc.

## Outstanding Questions

None at this time.
