# pkg/cli/agent

Implements the `synchestra agent` CLI command group: fenced worktree-claim operations and audited messaging, per `spec/features/agent-coordination` and task-3 of `spec/plans/synchestra-coordination-foundation.md`.

## Commands

| Command | Domain call | Description |
|---|---|---|
| `agent claim` | `state.WorktreeStore.Claim` | Claim exclusive write ownership of a repository/branch/worktree; reclaims an expired, un-released claim instead of conflicting. |
| `agent list` | `state.WorktreeStore.List` | List worktree claims with optional `--repository`/`--run`/`--active-only` filters. |
| `agent renew` | `state.WorktreeStore.Renew` | Extend a claim's lease without changing ownership. |
| `agent release` | `state.WorktreeStore.Release` | Explicitly end a claim. |
| `agent handoff` | `state.WorktreeStore.Handoff` | Explicit sequential handoff to a successor run under a freshly minted fence. |
| `agent message send` | `state.MessageStore.Send` | Send a typed, audited message (`--kind coordination.request/proposal/counterexample/decision.accepted`, `--evidence` refs). |
| `agent message list` | `state.MessageStore.Thread` | List every message in a thread, in delivery order. |
| `agent message ack` | `state.MessageStore.Acknowledge` | Acknowledge a message as a recipient. |
| `agent run start` | `state.RunStore.Start` | Start a run, recording its nullable model identity and provenance. |
| `agent run correct` | `state.RunStore.Correct` | Append an audited correction to a run's model identity without rewriting the original event. |

`claim`/`list`/`renew`/`release`/`handoff`/`message send`/`message list`/`message ack` are exactly the verbs task-3's brief names. `run start`/`run correct` are the one addition beyond that list: `agent-coordination#ac:optional-model-provenance-is-correctable` — one of task-3's four `Verifies` ACs — explicitly needs a CLI path for the correction workflow, so `run.go` adds the minimal surface that requires (not full effort/run CRUD). `WorktreeClaimParams.RunID` and `MessageSendParams.SenderRunID` remain plain caller-supplied identifiers otherwise (the same pattern `pkg/cli/task/claim.go`'s `--run` flag already uses) — claiming or messaging never requires a prior `run start`.

## Conventions

Follows `pkg/cli/task`'s established shape rather than introducing new ones: one file per verb, a shared `resolveStore`/`mapStoreError` pair (`resolve.go`), `--project`/`--sync`/`--format` flags where applicable, and YAML-default output (`format.go`) with a dedicated wire type per domain struct (`claimOutput`/`messageOutput`), matching `pkg/cli/task/format.go`'s `taskOutput`. `--project` is optional: if omitted, it is derived from the current directory's `origin` Git remote (`resolve.go`'s `resolveProjectID`), since `state-store/topology` scopes every agent-coordination record to a `ProjectID` that `Task()`/`Chat()` do not require.

`--fence-epoch`/`--fence-token` (`fence.go`) are the CLI-visible form of `state.LeaseFence`: a caller learns them from the JSON/YAML a prior `claim`/`renew`/`handoff` call returned and passes them back to prove current authority, mirroring how a real agent (e.g. Workbench) would persist them in its local Work Log between calls.

## Close and the batching window

Every command calls `state.Store.Close` (via `resolve.go`'s `closeStore`) before returning, per `state-store/journal-batching`'s "a caller that owns a Store's lifecycle... must call Close before process exit." `message send` and `run start` additionally use `state.CloseAfter` to avoid waiting out the group-commit window entirely: `Message().Send`/`Run().Start` each issue exactly one journal `Append`, preconditioned only by the journal's O(1) `Head()` read — the narrow contract `CloseAfter` requires (see its doc comment in `pkg/state/closeafter.go`).

Every OTHER mutating verb — `claim`/`renew`/`release`/`handoff`, `message ack`, `run correct` — does NOT use `CloseAfter`, for two distinct reasons `CloseAfter`'s doc comment covers in detail:

- `claim`/`renew`/`release`/`handoff`: `state.WorktreeStore`'s mutating methods each issue TWO sequential `Append`s internally (a lease operation, then a worktree follow-up event — see `pkg/state/agentstore/worktree.go`); racing a forced `Close` against a two-`Append` sequence can permanently fail the second `Append` with `ErrJournalClosed` instead of merely delaying it.
- `message ack`/`run correct`: `Message().Acknowledge`/`Run().Correct` each read the FULL journal history first (via the journal's `After()`, to find the message/run being acted on) before their single `Append` — unlike `Head()`, that scan's duration grows with journal history and is not bounded by `CloseAfter`'s fixed grace period, so it can (and, against a real Git-backed journal during task-3's own testing, did) fire `Close` before the `Append` was even attempted, permanently failing it the same way.

These verbs therefore still pay up to the full configured batching window (default 1000ms) per call in the absence of concurrent traffic — a known, documented trade-off, not an oversight; see `pkg/state/agentstore/README.md`'s Open Questions.

## Spec

See `spec/features/agent-coordination/` (including `cross-harness-conformance/` for the typed negotiation message vocabulary) and `spec/features/cli/README.md` for the shared CLI exit-code contract.

## Open Questions

- No `spec/features/cli/agent/` subtree exists yet describing this command group's surface as a dedicated CLI feature (unlike `spec/features/cli/task/`); this package currently traces only to `agent-coordination`'s behavioral spec. Authoring that CLI-feature spec (verb-by-verb `_args/` documentation, exit-code table, examples) is left to a follow-up rather than expanding task-3's scope into a separate spec-authoring pass.
- `--project` autodetection shells out to `git remote get-url origin`; a repository with no `origin` remote (or a non-standard remote name) has no fallback today besides the explicit `--project` flag. Whether to also try `resolve.StateRepoPath`'s own project-config sources is left open.
- Output format support is YAML/JSON only (no `csv`/`md`, unlike `pkg/cli/task/format.go`); add them if an actual consumer needs tabular claim/message views.
