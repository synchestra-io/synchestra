---
format: https://specscore.md/feature-specification
status: Approved
---

# Feature: Agent Coordination

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination?op=request-change) |
**Status:** Approved
**Source Ideas:** —

## Summary

Tracks efforts, agent runs, worktree claims, audited messages, active/recent views, overlap detection, recovery, and cleanup state across repositories.

## Contents

| Child | Description |
|---|---|
| [repository-change-notifications](repository-change-notifications/README.md) | Normalizes agent, Workbench, Git-provider, and reconciliation signals into verified ref-update notifications for affected active agents. |
| [cross-harness-conformance](cross-harness-conformance/README.md) | Proves that independently launched agent harnesses can negotiate, deliver, recover, and clean up through the same audited protocol. |
| [portable-merger-agent](portable-merger-agent/README.md) | Defines a harness-neutral, WB-managed integration role that batches compatible branches, validates the combined target tree, lands work, observes CI, and closes every branch/worktree claim. |
| [plan-overlap-coordination](plan-overlap-coordination/README.md) | Detects shared capabilities and conflicting scope across reviewable plans before implementation, assigns one audited provider, and turns every consumer into an explicit reuse dependency. |

## Problem

An agent's branch or worktree alone does not say who owns it, what user request
started it, whether its work is active, completed, aborted, superseded, or safe
to remove, nor which adjacent effort is changing the same code. This creates
duplicate work, unsafe cleanup, dead sessions that cannot be resumed, and a
fleet of branches whose state must be reconstructed from unreliable Git graph
proxies.

Codex, Claude, and human operators need one audited coordination vocabulary and
one project/repository view while retaining a Git-only recovery path if a server
is unavailable. This Feature intentionally coordinates work; it does not make
SpecScore core depend on Synchestra. SpecScore may emit generic extension events
and supply feature/lesson context, while Synchestra acts as an optional provider
through its CLI/server integration.

## End-to-End Journey

> “I open a project, see who is active and what each person owns; before two
> agents touch the same library they see the overlap and agree who will do it.
> If one dies, another can see what it was doing, resume safely, and later clean
> up only after the work is actually landed or explicitly abandoned.”

| Stage | Observable good result |
|---|---|
| Start — create effort and run | The effort exists with its origin, scope, private Work Log reference, and one named Codex/Claude/human run; both the actor and project view show the same active record. |
| Claim | The run owns one branch and worktree under a fencing lease; any competing writer receives the winner and conflict evidence rather than a guessed outcome. |
| Coordinate | A touched-area declaration reveals relevant active claims. Targeted messages are durable, acknowledged, and visible in each linked run's audit trail. |
| Change and refresh | A verified ref update notifies affected runs. Their view says whether refresh is required, while dirty work is never auto-rebased. |
| Handoff or recovery | A checkpoint identifies the worktree, branch/tips, base, test state, and next action. A successor either accepts a fenced handoff or is prevented from using a still-live claim. |
| Finish and cleanup | A completed run requires its declared merge target plus verified removal/recycle of every claim; an aborted/failed run requires an explicit handoff/removal/recycle disposition. The seven-day recent view preserves the audit summary after active records close. |

**Divergent epilogues.** A completed effort records landing evidence before
cleanup eligibility; an aborted effort remains recent with a resumable reason,
tip, and owner decision. Both epilogues let a human distinguish “safe to remove”
from “needs resumption,” without inferring either from branch age.

## Behavior

### Canonical vocabulary and durable records

An **Effort** is the durable unit of requested work. It has a stable ID,
project/repository IDs, title, declared scope, initiator, creation time, and a
private Work Log reference. The exact original prompt lives in the private
local Work Log and may be copied only to an explicitly configured, authorized,
encrypted private Synchestra server/cloud archive. It MUST NOT enter a public
dashboard, generic operational event, Git state repository, Git fallback,
mirror, or backup. Those surfaces retain only its digest and private-archive
receipt/reference.

A **Run** is one execution attempt within an effort. It records the agent
family (`codex`, `claude`, `human`, or registered implementation), nullable
model identity, role (`primary` or `subagent`), parent run, start/end
timestamps, terminal reason, bounded token/usage counters, and delivery state.
A model value is never guessed: when present it records provenance as
`runtime_observed` or `caller_declared`; when the runtime exposes no identity it
remains null/unknown. A correction appends an audited event naming the
superseded field/value and replacement; it never rewrites the immutable source
event. A subagent is a run, not an invisible annotation, so communication and
cost can be audited. Model-provenance and correction ingestion remain Planned
until the storage/API/CLI paths and recovery tests land.

A **Worktree Claim** binds exactly one concurrent writer run to one canonical
repository identity, local worktree location, local branch, target/base ref,
base SHA, observed head SHA, lease/fence token, and cleanup disposition. It
also records a bounded set of declared path/feature/module areas. The record
stores Git facts and hashes, never credentials or arbitrary workspace contents.

The Workbench-owned Hybrid Work Log is the local recovery counterpart:

- durable private journal/outbox: `~/.wb/worklogs/<effort-id>/`;
- minimal git-ignored projection inside the worktree, pointing back to effort
  and run IDs; and
- Synchestra archival state once delivery succeeds.

The Work Log is sufficient to explain a dead worktree while the server is down;
Synchestra is the queryable, cross-machine audit and coordination layer. A
terminal run follows a two-phase finalization protocol:

1. **seal** the mutable run journal into a content-addressed terminal record
   (including branch/base/tip/checkpoint and delivery/cleanup evidence);
2. **archive** that sealed record in the mandatory durable local archive outside
   the worktree, then enqueue optional Synchestra server/cloud archival; and
3. only after local archival succeeds, **reset/rebind** the git-ignored
   worktree projection to a new effort/run or remove the worktree.

The local archive has configurable retention and is never removed merely because
the worktree is removed/recycled. Remote archival uses a durable outbox/retry
record and an authenticated encrypted private-payload channel; it stores a
receipt plus sealed-content hash when accepted. The generic state journal and
Git replicas receive the receipt/hash, never the prompt-bearing payload. By default a
remote/server outage does not block safe cleanup because the local archive is
already durable; a project may require a remote archive receipt before cleanup.
Archived terminal logs feed recent/history views but never count as active
claims or leases.

### One-writer rule and cooperative work

The MVP permits **one concurrent writer per worktree and branch**. A claim is
acquired and renewed with the current authority epoch and fence token; every
mutation proves both. A run that loses/does not renew its lease must stop
writing and offer a checkpoint/handoff. Read-only observers and helpers that
return a patch, analysis, or message without touching the checkout are allowed.

Sequential cooperation is supported through explicit handoff: outgoing run
checkpoints, releases/transfer is fenced, incoming run accepts, then becomes
the sole writer. Multiple writers in the same checkout are intentionally not an
MVP feature: Git's index, uncommitted files, test outputs, and agent tools
provide no reliable operation-level isolation or attribution. Supporting it
would require a shared file-operation protocol, transactional staging/index
isolation, per-writer rollback, intent locks, and an audited conflict resolver;
until those exist it would make recovery and cleanup less deterministic, not
more collaborative.

### State machine, merge target, and cleanup

Effort and run lifecycle states are `planning`, `active`, `handoff_pending`,
`awaiting_merge`, `awaiting_push`, `awaiting_cleanup`, `completed`, `aborted`, `failed`,
`superseded`, and `archived`. A green test, pushed branch, or opened PR keeps a
run **active**; it is evidence, not completion. A terminal state is not
synonymous with delivered work. Completed runs record one of:

- `landed`: forge PR/merge or verified direct-to-target landing evidence;
- `handoff`: a successor run accepted ownership;
- `not_landed`: explicit remaining work / recovery decision; or
- `discarded`: explicit authorized abandonment.

Every effort declares its delivery level: a **feature effort** is delivered only
when its change is merged and the exact integrated SHA is pushed to
`origin/main`, and every related branch/worktree has been removed or recycled;
a **task effort** is delivered only when its change is merged and pushed to its
remote owning feature branch, and every related branch/worktree has been removed
or recycled. A local target merge is `awaiting_push`, not landing evidence; no
source or integration claim is cleaned until the remote target ref is verified.
A run cannot transition to `completed` before this remote target and all its
claimed assets are verified. An aborted/failed effort likewise has
a terminal audit disposition only after its assets are explicitly handed off,
removed, or recycled; otherwise it remains `awaiting_cleanup` and contributes
to the cleanup backlog.

Cleanup eligibility is derived, not guessed: clean worktree, no active lease,
known terminal/merge disposition, required Work Log durability, and a verified
asset disposition. Landing verification asks the forge for PR state when
available and compares trees/recorded target evidence for direct pushes; it
never declares a branch pending merely because a squash/rebase changed commit
SHAs.

The safe default is **remove**: remove the worktree and delete/archive its
branch only after the above evidence is durable. **Recycle** is optional and
must be enabled per project. It is a distinct audited transition, not leaving a
finished checkout in place: the old claim is sealed and archived, then closes;
Workbench resets the worktree projection before renaming/reassigning it to a
new effort/run; creates a new Work Log and new fenced claim; fetches and
refreshes the configured new base; and records only configured cache paths
retained from the old workspace. Source, generated files, Git index, untracked
work, and private run state are never silently inherited.
An otherwise finished branch/worktree still present without one of these
dispositions is a lifecycle violation and appears in the active cleanup backlog
rather than disappearing from the dashboard.

Cleanup transitions are themselves audited with worktree/branch/tip evidence.
Archiving hides records from default views but preserves searchable metadata and
audit history.

### Audited communication and overlap prevention

Messages are immutable envelopes linked to sender run, recipient run(s), effort,
thread, correlation ID, and optional repository/ref/claim. Recipient
acknowledgements are separate records. Thread membership is explicit and all
deliveries, acknowledgements, and transport evidence are retained under the
project's access policy. The message body is private state; list views expose
only safe metadata/snippets as configured.

Before claiming, a run submits declared scope as repository paths, module IDs,
SpecScore feature references, and optionally symbols. The server computes exact
path intersection plus conservative parent/child and feature dependency overlap.
An overlap is a notification, not an automatic denial: the claimant must either
link an existing coordination thread/accepted handoff, narrow its scope, or
record an explicit “independent after review” rationale. Concurrent writers may
work in separate worktrees/branches on related areas only with that visible
decision.

### Active/recent project and repository views

`synchestra project agents` and `synchestra repository agents` return active
runs and terminal runs from the default last **seven days**, grouped by project
or canonical repository. Output contains effort/run identity, agent/model,
state, parent/child relationship, worktree/branch ownership, scope, base/head
freshness, linked threads, overlap warnings, delivery/cleanup disposition, and
staleness/authority metadata. A graph view renders parent runs, messages,
shared effort, and overlap/hand-off edges. Workbench may present the same data
as an operational repository dashboard; other clients may render it differently
without duplicating coordination truth.

The server view is backed by the active store. Git fallback produces the same
minimum active/recent view from claimed records and envelopes, labelled
`transport: git-fallback` and with its last verified cursor. The recent window
is configurable per project; seven days is the default. Views also expose an
explicit cleanup-backlog count and each item's missing merge/removal/recycle
condition; this is never inferred from branch age or test status.

### Refresh policy

Each claim tracks `target_ref`, last fetched target SHA/time, last integrated
target SHA/time, dirty state, and `refresh_required` reason. Defaults are:

1. fetch target every 60 minutes and when a verified target update arrives;
2. fetch/assess before commit, push, handoff, finalization, and merge;
3. integrate after a clean checkpoint commit and before push/handoff/finalize/
   merge; and
4. never auto-integrate a dirty checkout — record the requirement and notify the
   owner instead.

Projects may configure the interval and trigger set. Unpublished private work
rebases onto target by default. Published/shared work merges target by default;
rewriting it requires explicit `--force-with-lease` authorization and updated
claim evidence. A notification says “base advanced” rather than commanding a
blind rebase, because the run owns the safe integration moment.

## Acceptance Criteria

### AC: one-writer-claim-is-fenced

**Given** two Codex/Claude runs request the same branch/worktree
**When** the active store processes both claims and one run later loses its
lease
**Then** exactly one claim succeeds, every later mutation from the loser is
rejected by epoch/fence, and the winner/claim evidence is visible to both.

### AC: overlap-is-visible-and-coordinated

**Given** an active run declares a shared library path and feature
**When** another run proposes overlapping work in a separate worktree
**Then** both project and repository views show the relationship and the second
run must record a handoff, coordination thread, narrowed scope, or explicit
independent rationale before it becomes active.

### AC: abandoned-run-is-resumable

**Given** a run stops without terminal server contact
**When** an operator opens its worktree or a successor starts
**Then** the Hybrid Work Log plus Synchestra/Git records identify the original
effort, agent/model/parent, branch/base/tip, checkpoint, remaining work, and
whether the old lease is still live; unsafe takeover is rejected.

### AC: optional-model-provenance-is-correctable

**Given** a runtime exposes no model ID, or a caller later proves a declared
model value was wrong
**When** Synchestra creates or corrects the Run record
**Then** it stores null/unknown rather than guessing, labels every non-null
value `runtime_observed` or `caller_declared`, and appends a correction linked
to the superseded event without rewriting audit history.

### AC: completed-and-aborted-are-distinguishable

**Given** one completed run with verified landing and one aborted run with
unlanded work
**When** the seven-day repository view and cleanup check run
**Then** both remain visible with their distinct disposition; only the landed
record can become cleanup-eligible without an explicit discard/hand-off.

### AC: delivery-requires-merge-and-asset-disposition

**Given** a feature effort with green tests and a pushed review branch, plus a
task effort merged to its feature branch with an unremoved worktree
**When** either run attempts `complete`
**Then** both remain active/`awaiting_merge`, `awaiting_push`, or
`awaiting_cleanup` with their missing condition; only the feature merged and
pushed to `origin/main` and task merged and pushed to its remote feature branch
can complete after every claimed branch/worktree is removed or explicitly
recycled.

### AC: local-target-merge-is-not-delivered

**Given** a task or feature change merged into its local target branch
**When** the target SHA is absent from or differs from `origin/<target>`
**Then** the effort is `awaiting_push`, the local target merge is visible as
durability backlog, every related claim/Work Log remains active, and cleanup or
completion is rejected until the exact remote ref is verified.

### AC: recycle-is-explicit-and-isolated

**Given** recycle is enabled and a landed claim is eligible for cleanup
**When** Workbench recycles its worktree to a new effort
**Then** the old claim is sealed into the mandatory local archive, optional
remote archival is queued with receipt/hash tracking, its projection is reset,
and a new run/Work Log/fence and refreshed base are created; only configured
cache paths are retained, and the repository view links rather than conflates
the two efforts.

### AC: offline-archive-does-not-block-default-cleanup

**Given** a sealed terminal run whose server/cloud archive is unavailable
**When** its local durable archive succeeds and the project uses the default
policy
**Then** cleanup/removal or recycle can proceed with a retry-backed remote
archive outbox; the archived log appears in recent history but is not an active
claim. A project requiring a remote receipt instead reports `awaiting_archive`.

### AC: messages-survive-transport-switch

**Given** linked Codex and Claude runs and a server outage
**When** they exchange messages through the Git fallback and the server returns
**Then** each envelope and acknowledgement is auditable exactly once in the
same thread, and no message is mistaken for an authoritative task transition.

## Open Questions

- What finite default retention should apply to sealed private Work Logs once
  automatic garbage collection ships? The MVP retains them indefinitely and
  exposes configurable retention rather than risking automatic evidence loss;
  metadata/hash and any required remote archive receipt may outlive private
  prompt content under a later policy.
- Which key authority should protect optional prompt-bearing remote archives?
  Lean: a per-log data key plus an organization/account key identifier in the
  envelope, allowing founder-hosted server-managed keys first and customer-KMS
  or client-held keys later without changing the sealed archive format.

---
*This document follows the https://specscore.md/feature-specification*
