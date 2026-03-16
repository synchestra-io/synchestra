# Implementation Plans

**Moved to `spec/plans/`**

All development and implementation plans are now created as formal development plans in the `spec/plans/` directory and follow the [Development Plan specification](../../features/development-plan/README.md).

Development plans:
- Start in `draft` status and progress through review to `approved`
- Are immutable once approved — changes require creating a new superseding plan
- Support complex task decomposition with dependencies and acceptance criteria
- Generate executable tasks in the state store

To create or manage plans, see:
- [Development Plan specification](../../features/development-plan/README.md#behavior) — structure and format
- [CLI commands](../cli/plan.md) — `synchestra plan create`, `submit`, `approve`, etc.

## Outstanding Questions

None at this time.
