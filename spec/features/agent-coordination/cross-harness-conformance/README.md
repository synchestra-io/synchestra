---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Cross-Harness Agent Coordination Conformance

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/cross-harness-conformance?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/cross-harness-conformance?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/cross-harness-conformance?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/cross-harness-conformance?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

A small, understandable synthetic journey that proves agents launched by
different harnesses can discover each other, negotiate a shared contract in
both directions, deliver compatible changes in one repository, recover through
the Git fallback, and leave no branch/worktree cleanup backlog.

## Problem

Unit tests can prove that a message API stores envelopes. They cannot prove that
real Codex CLI, Claude Code CLI, GitHub Copilot CLI, Claude Desktop, and Codex
desktop sessions can identify themselves consistently, receive a message,
reason about it, reply through the audited channel, use the decision in their
work, and complete the managed Git lifecycle.

A convincing conformance test needs a task simple enough to explain in one
minute, but coupled enough that two isolated agents must communicate. It must
grade protocol and observable behavior deterministically rather than deciding
whether natural-language conversation “looked collaborative.”

## End-to-End Journey

> “Two agents from different tools work on one Fair Split repository. One owns
> the allocation library and one owns the CLI/test. They discover an API and
> rounding overlap, exchange evidence, agree a deterministic contract, land
> both task branches, and leave no abandoned worktree or branch.”

| Stage | Observable good result |
|---|---|
| Start | The harness creates one feature effort and two fenced task runs in separate WB worktrees/branches of the same fixture repository; the repository view names each runtime and claimed area. |
| Discover | Both runs see the shared `fair-split-contract` overlap before integration and open one correlated coordination thread. |
| Negotiate | The CLI/test owner requests stable output; the library owner proposes a representation/remainder rule; the other run supplies a concrete counterexample; an accepted decision records evidence and is visible in both Work Logs. Messages flow in both directions. |
| Implement | The library and CLI/test changes compile independently against the accepted contract, without either writer modifying the other's worktree. |
| Integrate | A third, harness-portable merger run takes the frozen task heads, uses WB to merge and push both tasks to the remote feature branch and the feature to `origin/main`, and the executable returns the exact expected split. |
| Finish | Both runs are recent/terminal with message and landing evidence, while active claims and cleanup backlog for the fixture are empty. |

**Divergent epilogues.** The normal journey uses the server/SQLite active store
with Git mirror. The outage journey stops the server after the first request;
the reply and decision traverse append-only Git fallback, reconcile after
restart exactly once, and reach the same output and clean terminal lifecycle.

## Behavior

### Fair Split fixture

The fixture is a tiny Go repository with a library and command. Its user-visible
example splits exactly EUR 10.00 among `Alice`, `Bob`, and `Carol`. The accepted
contract uses ordered shares, conserves integer cents, and assigns indivisible
remainder cents in participant-name lexical order. The deterministic output is:

```text
Alice EUR 3.34
Bob EUR 3.33
Carol EUR 3.33
```

The three values MUST total EUR 10.00. Currency arithmetic uses integer cents;
floating-point tolerance is not part of the grader.

The library-owner prompt supplies conservation and library scope. The
CLI/test-owner prompt supplies stable human-readable ordering and CLI scope.
Neither prompt supplies the entire integration answer. Both identify the
Synchestra effort/run and require coordination through Synchestra rather than
direct terminal/stdout relaying.

### Typed negotiation

The thread contains, at minimum, immutable typed records for
`coordination.request`, `coordination.proposal`,
`coordination.counterexample`, and `coordination.decision.accepted`.
Every record carries sender, recipient, effort, thread, correlation ID,
repository/claim context, timestamp, and delivery evidence. The accepted
decision references the counterexample/test evidence and is acknowledged by
both runs.

Natural-language summaries remain useful to humans, but conformance asserts the
typed sequence, bidirectional senders, correlations, acknowledgements, and
evidence references. It MUST NOT grade exact prose or require a vendor-specific
prompt format.

### Harness adapter contract

An adapter declares runtime ID/version, launch command or interactive session
attachment, environment/capability preflight, Synchestra endpoint/fallback
configuration, bounded timeout, and terminal-result capture. Credentials are
provided by the operator's existing harness configuration and never copied into
the fixture, Work Log, message, or Git state.

The automated MVP runs Codex CLI and Claude Code CLI. GitHub Copilot CLI is the
next CLI adapter and the three pairwise combinations reuse the same journey.
Claude Desktop and the Codex desktop app later attach through the same run and
message protocol; an initially human-triggered desktop launch is acceptable,
but grading after attachment remains identical.

An unavailable adapter produces `skipped` with an explicit missing capability;
it never reports the journey passed. A required release matrix treats skipped
as failure according to release policy.

### Isolation and lifecycle

Each writer owns one WB-managed worktree/branch claim. A feature integration run
with role `merger` owns the feature branch and consumes immutable
`ready_for_integration` heads through the Portable Merger Agent contract. Task completion requires merge to that feature branch
and removal or explicit audited recycle of task branches/worktrees. Feature
completion requires merge to `main` and the same disposition for all remaining
assets. The grader compares Git/WB inventory to Synchestra claims and fails on
an orphan, partial multi-repository/run claim set, or `awaiting_cleanup` record.

Recycled fixture worktrees are reset to a new Work Log and claim; only configured
caches may survive. Sealed logs remain available as recent evidence.

### Store and outage evidence

The server-mode run records an authoritative SQLite cursor and verified Git
mirror barrier. The outage run records Git fallback envelopes and
acknowledgements, then a restart/reconciliation cursor. The grader compares
canonical record IDs and payload hashes across stores, rejecting loss,
duplication, reordered thread state, or dual-authority writes.

## Acceptance Criteria

### AC: two-cli-harnesses-negotiate-bidirectionally

**Given** available Codex CLI and Claude Code CLI adapters
**When** the Fair Split journey reaches the contract overlap
**Then** the two runtime identities exchange request, proposal,
counterexample, accepted decision, and acknowledgements in one correlated
thread, with at least one authored message in each direction.

### AC: accepted-contract-produces-exact-split

**Given** both task branches use the accepted ordered-share contract
**When** the integrated command splits EUR 10.00 for Alice, Bob, and Carol
**Then** its three exact lines are EUR 3.34, EUR 3.33, and EUR 3.33 in that
order, their integer cents sum to 1000, and all fixture tests pass on `main`.

### AC: coordination-evidence-is-auditable-not-prose-graded

**Given** two adapters express the negotiation in different natural language
**When** the grader inspects the run
**Then** it decides conformance from typed records, participants, correlation,
acknowledgements, evidence, final behavior and lifecycle, not exact wording.

### AC: server-outage-reconciles-the-same-negotiation

**Given** the server stops after the first coordination request
**When** both runs continue through Git fallback and the server restarts
**Then** the remaining reply/decision/acknowledgement records reconcile exactly
once in thread order, SQLite and Git converge on the same hashes, and neither
run writes through a second authority.

### AC: conformance-leaves-no-cleanup-backlog

**Given** both task changes and the feature change have passed
**When** the journey reports success
**Then** task branches are merged to the feature branch, the feature branch is
merged and pushed to `origin/main`, the remote feature target carries each task
landing, every associated worktree/branch is removed or explicitly
recycled with a reset Work Log, active claim count is zero, and cleanup backlog
count is zero.

### AC: adapter-matrix-reuses-one-scenario

**Given** a registered Codex, Claude Code, Copilot, or desktop adapter pair
**When** the same Fair Split fixture and grader execute
**Then** only launch/attachment mechanics differ; coordination records,
behavioral result, store checks and lifecycle assertions are unchanged.

### AC: portable-merger-drains-the-fixture

**Given** the two task agents have published immutable ready heads
**When** a Claude Code, Codex, or Copilot merger adapter receives the batch
**Then** it uses the same hashed WB merge plan to land task→feature→`main`,
records validation and landing receipts, and closes every source and integration
claim without the coordinating session performing duplicate Git mechanics.

## Open Questions

- What stable automation/attachment API should launch Claude Desktop and the
  Codex desktop app without coupling the protocol to UI automation?
- Should the release-blocking matrix require all three CLI pairs immediately,
  or graduate from Codex CLI ↔ Claude Code CLI after the adapter contract has
  passed repeatedly?

---
*This document follows the https://specscore.md/feature-specification*
