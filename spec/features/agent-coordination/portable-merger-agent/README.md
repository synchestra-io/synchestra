---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Portable Merger Agent

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/portable-merger-agent?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/portable-merger-agent?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/portable-merger-agent?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/portable-merger-agent?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

Defines a harness-neutral, WB-managed integration role that batches compatible
branches, validates the combined target tree, lands work, observes CI, and
closes every branch/worktree claim. Synchestra owns the audited assignment and
coordination record; Workbench owns deterministic local Git, validation, and
cleanup mechanics; Claude Code, Codex, and Copilot contribute thin launch
adapters and AI skills over the same role contract.

## Problem

A primary agent that writes features, reviews specialists, resolves merge
queues, waits for CI, and cleans every checkout becomes the delivery
bottleneck. Parallel writers can finish faster than that coordinator can land
one branch per CI run, leaving a growing queue of apparently finished branches
that are still work in progress.

The existing Claude Code `merger` agent is a strong baseline: it refreshes the
target, batches compatible branches, runs a fast suite after every merge and a
full suite before push, checks index/working-tree parity, fixes mechanical
conflicts, and watches CI inline. Its contract is currently harness-specific
and main-only, however, and it does not close WB/Synchestra claims or require
source worktree/branch cleanup. It also permits a raw `git worktree` fallback,
which bypasses the Work Log exactly when lifecycle evidence is most important.

## End-to-End Journey

> “Several agents have prepared compatible changes. I hand their run IDs to a
> dedicated merger. It shows the exact batch and target, integrates and tests
> the combined tree once, lands it, watches CI, then archives and removes every
> finished task asset. If a real product disagreement remains, it returns one
> precise audited question without pretending the work is done.”

| Stage | Observable good result |
|---|---|
| Assign | Synchestra creates one `merger` run with the target, immutable source heads, expected base, landing authority, and linked source runs; every participant sees who now owns integration. |
| Inventory | WB and Synchestra agree on source branches/worktrees, merge targets, dependency order, overlapping files, readiness, Work Log durability, and any existing cleanup backlog. Unknown or changing heads block the plan rather than being guessed. |
| Prepare | WB creates or reuses one explicitly claimed integration worktree, fetches the remote, and fast-forwards to the current target before applying any source. No canonical checkout is modified. |
| Integrate | Compatible sources join one batch in dependency order. After each source, the repository's fast validation identifies which addition caused any regression; behavioural conflicts become audited messages to the owners. |
| Validate | The full repository-defined validation runs against the exact committed integration tree. WB records command, exit status, tree SHA, and bounded evidence without secrets. |
| Land | WB re-verifies target/base and source heads, applies the configured direct-push or forge merge policy, immediately pushes the exact integrated target SHA to `origin`, and records the remote target receipt. A task lands on its remote feature branch; a feature lands on `origin/main`. |
| Observe | The merger follows provider checks to a terminal result. A mechanical red result is fixed forward; a policy/product failure remains active and is handed back with evidence. |
| Close | WB seals and archives each source and merger Work Log, removes or explicitly recycles every related worktree, deletes every retired branch, and proves Synchestra active claims and cleanup backlog are both zero. |

**Divergent epilogues.** A successful batch reaches the declared target and
closes every asset. A blocked batch keeps its integration claim and resumable
Work Log, sends a typed escalation naming the exact conflicting behaviors and
commits, and leaves every source run visibly `awaiting_merge` rather than
silently abandoning the queue.

## Behavior

### Product ownership and portable role

Synchestra owns assignment, run/claim state, messages, approvals, and the
repository/project view. Workbench owns filesystem placement, Git operations,
refresh, validation execution, landing evidence, Work Log sealing, branch
deletion, and worktree removal/recycle. The harness only launches an agent with
the shared role instructions and makes its identity/usage available; it does
not reimplement merge policy.

The canonical role is versioned independently from its adapters. Each adapter
declares the same role version and capability manifest. Claude Code, Codex, and
Copilot skills may differ in invocation syntax, but their normative stages,
stop conditions, output schema, and WB/Synchestra commands are identical. A
capability-delivery matrix must show runtime, `--help`, AI-skill, and test
coverage for every merger command before an adapter claims Full support.

The default dispatcher sends mechanical integration to this role once two or
more compatible branches are ready, or whenever the coordinator requests it.
The primary agent remains responsible for architecture and product decisions;
it does not duplicate the merger's local Git/CI loop while a merger run owns
the batch.

### Deterministic Workbench workflow

Workbench exposes one machine-readable merge workflow, provisionally
`wb merge plan|apply|status|resume`, with explicit effort/run IDs, target ref,
source run/branch IDs, expected SHAs, validation profile, and `--format json`.
`plan` is read-only and produces a content hash. `apply` requires that hash and
rejects drift before mutation. Every subsequent step is resumable and
idempotent from the Work Log; a process crash cannot make a later agent guess
which merge, push, or cleanup actions already occurred.

The merger invokes WB for managed lifecycle operations rather than issuing
ad-hoc Git worktree/branch deletion commands. WB may use Git internally. If the
Synchestra server is down, WB uses the Git-backed coordination fallback and
local Work Log outbox. If WB cannot durably record a mutation, the merger stops;
it never bypasses WB with an unaudited worktree fallback.

### Readiness, freezing, and batching

A source run publishes `ready_for_integration` with its exact head, target,
validation evidence, and Work Log archive state. That head becomes immutable
input to the plan. A new source commit invalidates the plan and requires a new
readiness record; source agents do not continue writing behind the merger.

The merger derives dependency/containment order, shared files and declared
scope overlap. It batches all compatible branches so one full validation and
one CI run can drain parallel work. It splits a batch only for an explicit
behavioral conflict, dependency release boundary, protected-risk policy, or
materially different target. It never serializes merely because branches were
authored separately.

The integration worktree starts from a fetched and fast-forwarded remote target
on every attempt. The merger never force-pushes or rewrites published history.
After each source it runs the configured fast profile; before landing it commits
all mechanical resolutions, proves the working tree and index equal the tested
tree, then runs the full profile once.

### Conflicts and audited communication

Mechanical conflicts are resolved by retaining compatible additions,
completing renames, honoring intentional deletion, and updating generated
artifacts through their owner tool. Any resolution that changes behavior,
wire/storage format, an approved decision, or test coverage is not mechanical.
The merger sends a typed `integration.decision_required` message with both
alternatives, commits/files, failed evidence, and its suggested resolution;
the relevant owners reply through the same thread.

The accepted answer becomes `integration.decision.accepted` and is referenced
by the resumed plan. A blocked or failed merger never reports a branch as done
and never releases its claim without a handoff, discard authorization, or safe
cleanup disposition.

### Landing, CI, and terminal cleanup

Immediately before landing, WB verifies the remote target still equals the
planned base (or deterministically refreshes/replans), all source heads still
match, validation evidence names the integration tree, and required approvals
remain valid. Provider-specific direct-push/PR/merge and CI adapters return a
provider-neutral receipt whose target ref equals the integrated tree SHA. Every
local target merge is pushed to `origin` immediately. GitHub is the first
provider; absence of a supported receipt keeps the effort `awaiting_merge`, and
a local merge whose push failed is explicitly `awaiting_push`. Neither state is
eligible for cleanup.

CI is observed to terminal state by the merger run. A mechanical failure is
fixed forward on a new audited attempt. A target with red required checks never
starts the next batch. Queued checks are distinguished from running or failed
checks, and a bounded timeout returns resumable state rather than success.

After verified landing, WB processes assets in delivery order. Task sources are
merged to and removed after their feature target; the feature integration asset
is removed only after the feature reaches `main`. Each retired branch is
deleted, each Work Log is sealed into the durable local archive with optional
remote archival outbox, and each worktree is removed or explicitly recycled
under project policy. A successful merger report is impossible while any
linked active claim or cleanup backlog entry remains.

## Acceptance Criteria

### AC: adapter-contract-is-harness-neutral

**Given** Claude Code, Codex, and Copilot merger adapters at the same role
version
**When** each receives the same synthetic batch
**Then** only launch/attachment metadata differs; plan fields, WB/Synchestra
operations, stop conditions, audit events, terminal report, and lifecycle
outcome are equivalent.

### AC: compatible-branches-land-as-one-batch

**Given** three ready source runs with compatible changes and one declared
target
**When** the merger plans and applies them
**Then** it refreshes the target first, runs the fast profile after each source,
runs the full profile against the committed combined tree, and creates one
landing/CI batch rather than three serialized batches.

### AC: drift-invalidates-the-plan

**Given** a hashed merge plan with exact source and target SHAs
**When** either a source agent commits again or the remote target advances
**Then** `apply` rejects the stale plan before mutation and returns the changed
identity required for deterministic replan/refresh.

### AC: behavioral-conflict-is-audited-not-guessed

**Given** two branches disagree on behavior or a wire format
**When** integration reaches that conflict
**Then** the merger sends one correlated decision request with both alternatives
and evidence, keeps the batch active, and resumes only from an accepted audited
decision; it does not silently pick one side or delete coverage.

### AC: tested-tree-is-the-landed-tree

**Given** a mechanical conflict required edits after Git staged a merge result
**When** full validation passes
**Then** WB proves the committed tree SHA equals the validated and subsequently
landed tree SHA; unstaged/staged drift blocks landing.

### AC: every-target-merge-is-pushed-immediately

**Given** a validated task→feature or feature→main target merge
**When** the merger advances the local target ref
**Then** the same operation immediately pushes the exact SHA to
`origin/<target>` and verifies it; a failed or deferred push leaves the run
`awaiting_push`, retains all claims/assets, and cannot be reported as landed.

### AC: success-requires-zero-related-assets

**Given** task sources targeting a feature branch and a feature integration
targeting `main`
**When** the merger reports success
**Then** verified receipts prove both delivery levels, every retired source and
integration branch is deleted, every Work Log is sealed/archived, every
worktree is removed or explicitly recycled, and linked active-claim and cleanup
backlog counts are zero.

### AC: wb-outage-cannot-be-bypassed

**Given** Synchestra server outage with a healthy Git fallback, or a WB
persistence failure
**When** the merger needs to mutate lifecycle state
**Then** server outage continues through WB's Git/Work Log path, while WB
durability failure stops with resumable evidence; no raw-worktree fallback
creates an untracked branch or checkout.

## Open Questions

- Should the public command group be named `wb merge`, `wb land`, or
  `wb integrate`? The behavior and capability IDs should be frozen before the
  first released AI-skill adapters.
- Which protected-branch providers beyond GitHub must the first provider-neutral
  landing/CI receipt contract support?

---
*This document follows the https://specscore.md/feature-specification*
