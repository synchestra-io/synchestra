---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Plan Overlap Coordination

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/plan-overlap-coordination?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/plan-overlap-coordination?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/plan-overlap-coordination?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/agent-coordination/plan-overlap-coordination?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

Detects shared capabilities, repository collision surfaces, integration order,
and conflicting scope across reviewable plans before implementation. Plan
owners coordinate through an audited thread, nominate one provider for shared
work, and turn every other plan into an explicit reuse dependency instead of
paying twice to implement and later reconcile the same functionality.

## Problem

Two master agents can independently produce valid feature plans that both need
to extend the same library, schema, algorithm, CLI, or repository. Without a
cross-plan review, each plan looks internally correct. The duplication becomes
visible only after both worker fleets have spent tokens implementing it and a
merger must decide which version survives.

Even plans that do not duplicate functionality can target the same repository.
They need an explicit answer about whether their changes are independent, what
their integration points are, which order should land, and when each agent must
refresh its base. A single coordinator may overlook the relationship; separate
people cannot rely on shared conversational context at all.

## End-to-End Journey

> “Two agents draft different feature plans. Before I approve either one, the
> review shows that both touch the same repository and both propose the same
> allocation primitive. The owners agree that one task provides it and the
> other reuses it. Independent tasks continue in parallel, the consumer waits
> only at the real dependency, and integration has one implementation rather
> than two competing versions.”

| Stage | Observable good result |
|---|---|
| Draft | Each plan revision yields a compact intent manifest naming repositories, capabilities provided/required, touched modules/paths/contracts, produced artifacts, targets, and task dependencies. The full plans need not be loaded by every agent. |
| Submit for review | A mandatory incremental scan compares the submitted revision with In Review, Approved, Executing, and Blocked plan intents, plus Implemented provider artifacts. Same-repository work is visible even when no deeper overlap is yet proven. |
| Classify | Deterministic identifiers find exact shared capabilities and artifacts; path/module/contract intersections find integration risk; semantic matching proposes likely duplicates with confidence and evidence but does not silently block. |
| Coordinate | Plan owners receive one correlated thread showing exact task pairs. They confirm independent work, partition scope, order coupled changes, nominate one provider and consumers, extract a shared prerequisite plan, reuse an existing artifact, or request a behavioral decision. |
| Revise | The accepted resolution is reflected in new immutable plan revisions: duplicate provider work disappears, consumer tasks gain explicit dependencies and integration checkpoints, and the overlap finding points to the accepted resolution. |
| Approve | A synchronous policy check permits approval only when deterministic blocking findings are resolved against the exact revision. Advisory semantic findings remain visible with their disposition. |
| Execute | Synchestra schedules unrelated DAG tasks concurrently. The shared provider has one owner; consumers cannot claim the dependent task until the provider's remote target/artifact receipt is verified, but may continue all other ready tasks. |
| Integrate | Repository views show the agreed order and integration points. Base-advance notifications reach consumers when the provider lands, and the Portable Merger Agent follows the recorded dependency rather than rediscovering it from a conflict. |

**Divergent epilogues.** If same-repository plans are independent, both owners
record `independent_after_review` with non-overlapping scope, target, landing
order, and refresh checkpoint, then proceed in parallel. If they share a
capability, exactly one provider remains and consumers wait/reuse. If they
disagree about behavior, approval remains blocked on an audited decision rather
than letting two implementations become an expensive voting mechanism.

## Behavior

### Candidate plans and plan-intent claims

A **Plan Intent** is a non-exclusive, revision-bound declaration of expected
work. It is deliberately distinct from a fenced Worktree Claim: several plans
may honestly consider the same repository, but only execution acquires a writer
claim. An intent stores no original prompt and contains only compact planning
metadata safe for project collaborators.

The mandatory candidate set contains plans in `In Review`, `Approved`,
`Executing`, or `Blocked`. A direct Draft→Approved transition runs the same
check before approval. Authors may scan Draft plans on demand or opt them into
early sharing. `Implemented` plans are indexed as potential providers/reuse
evidence, not live conflicts. `Rejected`, `Withdrawn`, `Superseded`, and
`Deprecated` plans are excluded from active collision checks but remain audit
history.

Every intent is keyed by plan identity plus content revision/hash. Changing a
plan invalidates prior findings and resolutions for changed tasks. Stale
results cannot approve a new revision.

### Compact intent manifest

The canonical derived JSON manifest contains, per task:

- plan/task ID, revision, status, owner, source Feature, and target branch;
- canonical repository IDs and project/module IDs;
- `provides`: stable capability/contract identifiers and expected version;
- `requires`: capability/artifact identifiers and task/plan references;
- `touches`: Features, modules/packages, path prefixes, public symbols/APIs,
  schemas/wire/storage formats, migrations, and generated artifacts;
- `produces`: named artifacts already supported by SpecScore Plans;
- planned integration points, ordering constraints, and evidence provenance;
  and
- when a task is a migration/cutover, its mode (`full_cutover`,
  `staged_dual_run`, or `scoped_pilot`), complete consumer inventory, retained
  legacy consumers, rollback control, and removal deadline; and
- authored versus mechanically derived versus semantic-suggested markers with
  confidence. A semantic suggestion never masquerades as authored metadata.

SpecScore should define generic task fields such as `Repositories`, `Provides`,
`Requires`, and `Touches`; they are useful to any orchestrator and do not name
Synchestra. Synchestra derives and stores the queryable intent projection. Until
those fields ship, a Synchestra namespaced sidecar may carry the same data, but
it must reference the plan revision and must not become a competing Plan format.

Agents normally read their plan plus the small set of matched manifests and
task excerpts, not every occurrence or full plan in the project. The index is
incremental and caches semantic results by revision so unchanged plans do not
consume repeated model tokens.

### Detection levels

Detection is ordered from cheap/deterministic to assisted:

1. **Repository co-location.** Any shared canonical repository creates a
   `review_required` relationship. It is not itself a conflict.
2. **Exact capability/artifact identity.** Two providers for one capability,
   or a provider duplicating an Implemented artifact, is a blocking duplicate
   until resolved.
3. **Structural intersection.** Shared Features, modules, path prefixes,
   symbols, schemas, migrations, generated outputs, or target branches produce
   typed conflict/integration findings.
4. **Dependency graph.** Existing `Depends on`, `Requires`, and `Produces`
   relationships find missing providers, cycles, and ordering opportunities.
5. **Semantic suggestion.** A model compares compact unmatched task summaries
   and proposes likely shared intent with cited phrases and confidence. Owners
   must confirm or dismiss it; low-confidence prose alone cannot block approval.

Each finding names both plan revisions, exact task pairs, repositories, match
keys/evidence, suggested classifications, and the minimum excerpts required to
decide. Repository-wide “these plans overlap” prose without task evidence is
not actionable output.

### Ambiguous migration and cutover scope

The planner and policy checker treat “migrate”, “move”, “port”, “replace”,
“consolidate”, and “retire” as cutover-intent signals. The default is
`full_cutover`: inventory every consumer/surface/caller, replace all of them,
remove the old implementation, and verify no legacy caller remains. A route or
screen mentioned as an example is not a scope boundary unless the owner says it
is.

If an agent believes coexistence is useful for rollback, feature flags, A/B
testing, or a pilot, it does not choose that product policy. Before approval it
presents the complete consumer inventory and asks the owner to select
`full_cutover`, `staged_dual_run`, or `scoped_pilot`. Staged/pilot modes require
an explicit retained-consumer list, one authoritative switching mechanism,
telemetry/success gate, rollback rule, owner, and removal deadline. A plan that
uses a cutover verb but neither proves full scope nor records an approved
exception is a deterministic blocking finding.

Consumer discovery uses repository/module graphs, symbol/reference searches,
routes/screens, configuration and generated-contract ownership before semantic
assistance. The manifest records evidence and blind spots. If consumers lie in
repositories outside the plan's authority, the plan gains provider/consumer
tasks or asks for expanded authority; it does not quietly leave those consumers
on the legacy path.

### Required resolution vocabulary

The owners choose one audited, typed resolution:

- `reuse_existing`: verify the existing provider's artifact/version and remove
  planned reimplementation;
- `provider_consumer`: designate one provider task/run and add explicit
  consumer dependencies;
- `extract_shared`: move reusable cross-feature work to a dedicated Feature/
  Plan in its owning repository and make both plans consumers;
- `partition_scope`: assign disjoint modules/paths/contracts and their owners;
- `sequence_integration`: retain coupled work but state order, target, handoff
  artifact, base-refresh checkpoint, and merger batch boundary;
- `independent_after_review`: both owners confirm independent scope while still
  recording landing order and integration points;
- `behavioral_decision`: block until one contract/behavior is accepted; or
- `supersede`: retire one task/plan in favor of the other.

Different owners acknowledge the resolution from their own runs. A single
person operating multiple master agents may approve as coordinator, but each
affected agent still receives the revision/dependency through the audited
channel. No resolution exists only in chat prose.

If the provider is already Implemented, the consumer verifies a published
package, remote target SHA, schema, or other declared artifact before claiming
its dependent task. If a provider is Executing, the consumer waits at that DAG
edge and continues unrelated tasks. If the provider aborts/fails, Synchestra
requires an explicit replacement owner, handoff, or plan revision; it does not
quietly let every consumer start its own copy.

### SpecScore extension boundary

The current SpecScore CLI event subscriber system is sufficient for emitting
plan-revision notifications and asynchronously indexing them. It is **not
sufficient for an approval gate**: Exec subscriber stdout is discarded,
subscribers cannot return structured findings, and fan-out succeeds when any
subscriber succeeds. Those one-way delivery semantics must remain appropriate
for events rather than being overloaded into policy.

SpecScore therefore needs a generic synchronous policy-check extension boundary
for lifecycle transitions. A checker receives an immutable artifact envelope on
stdin and returns a versioned findings document with stable finding IDs,
severity, blocking/advisory disposition, evidence references, remediation, and
checked revision. Required checkers fail closed for approval; advisory checkers
cannot block. After a successful transition, the existing durable event/outbox
notifies subscribers. SpecScore knows only this generic contract; Synchestra is
one checker implementation.

SpecScore Studio does not need Synchestra-specific JavaScript. It needs only a
generic rendering of the same versioned findings envelope plus provider links;
Synchestra and WB may render richer project/repository graphs from their own
operational data.

### Project/repository view and execution guard

Project and repository views show upcoming plan-intent relationships alongside
active/recent agent claims, with distinct styling so planned work is not
mistaken for a writer. The graph includes plan/task, provider/consumer,
same-repository, integration-order, message/decision, execution claim, and
landing edges. Filters answer “what else plans to touch this repository or
capability before I approve/start this?”

At task-claim time Synchestra reruns deterministic checks against the approved
revision and current provider state. It rejects a second unresolved provider,
an unmet shared dependency, or a plan whose resolved revision changed. At
ready-for-integration time it verifies the provider artifact/remote target and
recorded sequencing checkpoint, protecting against drift after review.

Metrics count exact duplicates prevented, shared providers reused, plan-time
versus merge-time overlaps, behavioral escalations, and Work Log token/elapsed
usage by outcome. Token savings are reported as measured comparisons or bounded
estimates, never invented from task counts alone.

### Synthetic conformance scenario

The Fair Split fixture begins with two independent master agents:

- the library Feature plan includes a task providing
  `fair-split/ordered-cent-allocation`; and
- the CLI Feature plan initially includes its own rounding/allocation helper in
  the same repository while also needing stable output.

Submitting both for review must expose the shared repository, exact/semantic
capability overlap, and integration point. The owners accept the library task
as provider; the CLI plan removes its duplicate helper, adds a requirement on
the provider artifact, and retains unrelated CLI formatting/test work. Worker
agents then implement exactly one allocator, the consumer resumes after the
remote feature target advances, and the portable merger lands the combined
journey with no duplicate symbol, behavioral conflict, or cleanup backlog.

A paired negative fixture has two plans in the same repository touching
disjoint documentation and benchmark paths. Both owners confirm independence,
record landing order/refresh, and the review does not serialize their work.

## Acceptance Criteria

### AC: same-repository-plans-require-explicit-evaluation

**Given** two In Review plans name the same canonical repository
**When** the overlap check runs
**Then** both owners receive a task-paired repository relationship and must
record independence, partition, sequencing, reuse, or another typed resolution;
repository co-location alone does not falsely claim a code conflict.

### AC: one-provider-replaces-duplicate-implementation

**Given** two plans provide the same stable capability or one proposes a
capability already delivered
**When** owners resolve the finding
**Then** exactly one provider remains, every consumer has an explicit dependency
and artifact/integration point, and an unresolved second provider blocks
approval and task claiming.

### AC: semantic-suggestions-are-advisory-and-cited

**Given** two tasks with different identifiers but similar intent text
**When** semantic detection proposes an overlap
**Then** the finding includes exact plan revisions, task excerpts, confidence,
and an `assisted` provenance marker; it becomes blocking only after deterministic
confirmation or an owner classifies it as shared/conflicting.

### AC: independent-work-remains-parallel

**Given** two plans target the same repository but declare and confirm disjoint
scope
**When** review completes
**Then** both can approve and execute concurrently, while their target, landing
order, integration points, and base-refresh checkpoint remain visible to agents
and the merger.

### AC: consumer-waits-only-at-real-dependency

**Given** a consumer plan has unrelated ready tasks plus one shared-provider
dependency
**When** the provider is still Executing
**Then** Synchestra schedules unrelated tasks, blocks only the dependent claim,
and notifies/resumes that edge after a verified remote provider receipt.

### AC: approval-check-is-revision-bound-and-fail-closed

**Given** an In Review plan with a deterministic unresolved overlap and a
required Synchestra policy checker
**When** approval is requested
**Then** SpecScore rejects the transition with the generic structured finding;
after owners revise and resolve it, only a check of the new exact revision may
permit approval. An unavailable required checker cannot be treated as success.

### AC: migration-mode-covers-every-consumer-or-is-approved

**Given** a plan task asks to migrate a mechanism and names one example screen
**When** review discovers additional screens/routes/callers
**Then** the plan either selects full cutover and owns replacement plus
zero-legacy proof for all consumers, or records the owner's explicit staged or
pilot choice with retained consumers, switch/rollback, telemetry and removal
deadline; the example alone cannot satisfy scope.

### AC: event-hooks-remain-decoupled-from-policy-results

**Given** the same successful approval transition
**When** SpecScore runs policy and notification extensions
**Then** the synchronous generic checker decides the gate, while the durable
event/outbox independently fans out the accepted transition; Synchestra remains
a plugin and SpecScore core contains no Synchestra-specific logic.

### AC: fair-split-plans-produce-one-allocator

**Given** the two Fair Split master-agent plans initially contain overlapping
allocation work
**When** plan review, execution, and integration finish
**Then** the accepted revisions identify one provider and one consumer, source
contains exactly one ordered-cent allocator, audited records show both owners'
agreement, the remote target receipts satisfy dependencies, and no active
claim/branch/worktree cleanup backlog remains.

## Open Questions

- What stable namespace should capability identifiers use across repositories?
  Lean: Feature/GraphSpec/package identifiers first, with project-local aliases
  resolved to canonical URIs in the derived manifest.
- Which semantic model/embedding provider should the MVP use? The stored
  contract should contain only input hashes, bounded excerpts, confidence and
  disposition so provider choice can change without changing plan semantics.

---
*This document follows the https://specscore.md/feature-specification*
