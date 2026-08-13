# Research

Recorded research artifacts — benchmarks, comparisons, and other measured
evidence — that back a Feature's Acceptance Criterion where the AC itself
asks for measured evidence rather than only pass/fail behavior. This is
distinct from `spec/decisions/` (ADRs recording a design choice and its
rationale) and from `spec/features/*/_verify`/`_recap` (SpecScore's own
per-commit AC verification reports): a research record here is a point-in-time
measurement, explicitly expected to be re-recorded when the measured
behavior could plausibly have changed.

Each research artifact lives in its own directory (`<topic>/README.md`) and
states, at minimum: which Feature/AC it backs, the exact reproduction
command, and the machine context the numbers were recorded on.

## Contents

| Directory | Description |
|---|---|
| [state-store-backend-comparison/](state-store-backend-comparison/README.md) | Git-active vs. SQLite-active append/replication-lag/recovery-time comparison backing `state-store/topology#ac:backend-comparison-is-equivalent` |

### state-store-backend-comparison

Benchmarks the two initial supported topologies (Git active/SQLite mirror,
and SQLite active/Git mirror) against real backends, reporting append
throughput (batched and unbatched), replication lag drain, restore time, and
Git repository growth for both directions.

## Open Questions

None at this time.

---
*This directory follows repository conventions from AGENTS.md; entries here
are not SpecScore Feature/Idea/Plan artifacts.*
