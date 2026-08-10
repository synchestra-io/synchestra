# Plan: Physical State Store MVP

**Status:** draft
**Features:**
  - [state-store](../../features/state-store/README.md)
  - [agent-coordination](../../features/agent-coordination/README.md)
**Source type:** feature
**Source:** [State Store](../../features/state-store/README.md), [Agent Coordination](../../features/agent-coordination/README.md)

## End-to-End Journey

> “A Codex CLI run and a Claude Code CLI run own separate Fair Split Relay
> worktrees, negotiate the cents policy in one audited thread, and a human can
> see the active/recent effort, recover from a server outage through Git, and
> prove that every branch and worktree was merged then removed or recycled.”

| Stage | Observable good result |
|---|---|
| Start | An effort, runs, and exclusive worktree claims exist in the active store and Git mirror, with current epoch/fence evidence. |
| Coordinate | The durable thread contains `message.requested`, `message.proposed`, `message.counterexample`, and `decision.accepted` with one correlation ID; both runs see acknowledgements. |
| Replicate/fallback | Either Git or SQLite may be active; the other reaches the same ordered checksum cursor. With the server unavailable, Git remotely stores an immutable communication envelope in a separate inbox; reconciliation into authority is Planned. |
| Finish | The library task reaches the feature branch, the feature reaches `main`, and each claim records removal or explicit recycle before it is completed. |

## Validated locally, not merged

| Item | Evidence | Verifies |
|---|---|---|
| Backend-neutral topology and ordered journal contract | `pkg/state/replication`, commit `bdf27b1` | one active, Git-required, epoch/sequence/checksum/idempotency rules |
| Physical Git/inGitDB ↔ SQLite/DALgo replication | `dal_journal_physical_test.go` | both active→mirror directions, restart persistence, exact lag/divergence fencing, Git remote receipt and fresh-clone parity |
| Git adapter rollback-safe multi-record transaction | `github.com/ingitdb/dalgo2ingitdb` commits `b093a18`, `27d3f4d` | failed domain+journal+outbox write leaves no changed state; caller index and readers remain isolated |

## Remaining Tasks

### 1. Wire the physical store into CLI/server configuration

Implement `synchestra state topology`, `synchestra state status`,
`synchestra state replicate`, `synchestra state wait`, and
`synchestra state reconcile`. `status --format json` returns active endpoint,
authority epoch, each replica cursor/lag/last error, and cleanup backlog.

Before implementing the command group, add a machine-readable capability
matrix to `spec/features/cli/` and test it against Cobra help. It is the
single traceability source for CLI help, plugin skills, and implementation
support state:

| Capability ID | Command | Required support state | Help anchor | Skill anchor |
|---|---|---|---|---|
| `state.topology` | `synchestra state topology` | Git/SQLite roles and replica config | `state topology --help` | `skills/state/references/topology.md` |
| `state.status` | `synchestra state status --format json` | authority, cursor, lag/error, active/recent, cleanup backlog | `state status --help` | `skills/state/references/status.md` |
| `state.replicate` | `synchestra state replicate` | ordered/idempotent outbox drain | `state replicate --help` | `skills/state/references/topology.md` |
| `state.reconcile` | `synchestra state reconcile` | Git fallback/repository-ref import | `state reconcile --help` | `skills/state/references/fallback.md` |
| `state.wait` | `synchestra state wait --cursor` | mirror durability barrier | `state wait --help` | `skills/state/references/topology.md` |
| `claim.worktree` | `synchestra claim worktree` | fenced one-writer claim/handoff | `claim worktree --help` | `skills/state/references/claim.md` |
| `message.thread` | `synchestra message send/ack` | typed audited thread envelope | `message --help` | `skills/state/references/message.md` |

The test asserts each capability ID has one implemented command and stable
flags, one help anchor, one skill reference, and a non-ambiguous support state
(`implemented`, `planned`, or `blocked`). A command or skill cannot ship
without this row.

**Verifies:** A SQLite-active project and a Git-active project both expose the
same cursor after `state replicate`; an unavailable mirror reports nonzero lag
and its error without changing the active result.

### 2. Implement claims, threads, and Git fallback over the journal

Add Effort, Run, Worktree Claim, lease/fence, message acknowledgement, and
fallback-envelope import records. The physical Git fallback inbox accepts only
the v1 communication allowlist and has remote receipt/fresh-clone proof; its
reconciliation into the active authority remains Planned. Task/claim mutation
still needs explicit epoch-fenced promotion.

**Verifies:** A stale run cannot mutate a claim, and a server-down message is
auditable in Git then imported once after reconciliation.

### 3. Ship the agent-facing skill contract before the CLI commands ship

The owning plugin path is
[`ai-plugin-synchestra/skills/state/SKILL.md`](https://github.com/synchestra-io/ai-plugin-synchestra/tree/main/skills/state/SKILL.md).
Create it as the resource index, with progressive-disclosure references:

- `references/topology.md` — configure/select active and replica roles;
- `references/status.md` — inspect active/recent agents, lag, cleanup backlog;
- `references/claim.md` — claim/handoff/complete only with merge and asset disposition;
- `references/message.md` — send/acknowledge typed audited messages and scope overlap decisions;
- `references/fallback.md` — Git fallback, reconcile, and no split-brain mutation;
- `references/fair-split-relay.md` — executable two-agent conformance journey.

Codex and Claude Code wrappers call the same CLI and use the same IDs,
cursors, and error meanings. Copilot CLI is the next adapter; desktop wrappers
come later. Skills contain workflow selection and interpretation only, never
their own storage mutation implementation.

**Verifies:** Each new CLI command is reachable from the state skill, and an
agent can complete the Fair Split Relay run using only the documented
references and CLI output.

### 4. Build Fair Split Relay cross-CLI conformance harness

Create the tiny Go library and task worktrees. The library-owner and
CLI/test-owner exchange the typed thread above, accept the deterministic policy
`€10 / Alice, Bob, Carol = €3.34, €3.33, €3.33`, and attach commit/worktree
evidence to their Work Logs. Run the scenario twice: normal SQLite-active and
server-down Git-fallback.

**Verifies:** The harness asserts ordered messages, correlation/evidence in
both Work Logs, Git↔SQLite parity, merge task→feature→main, and zero abandoned
branches/worktrees after the audited cleanup operation.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/plan-specification*
