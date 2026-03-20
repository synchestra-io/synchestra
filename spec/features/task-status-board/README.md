# Feature: Task Status Board

**Status:** Conceptual

## Summary

A markdown table embedded in task directory READMEs that provides at-a-glance visibility into task status, ownership, and progress. The board is the source of truth for task state within a Synchestra project.

## Problem

When multiple agents and humans work on a project concurrently, there's no single place to see what's happening. Task state is scattered across git branches, commit messages, and individual task documents. Without a board, answering "what's in progress right now?" requires traversing the entire task tree.

## Location

The board appears in any task directory that contains sub-tasks:

- `synchestra/projects/{project}/tasks/README.md` — root task board
- `synchestra/projects/{project}/tasks/{task}/README.md` — sub-task board (when a task has its own sub-tasks)

The `synchestra/projects/{project}/tasks/` tree is the **source of truth** for task state. Boards may appear elsewhere (e.g., `spec/features/`) for convenience, but those are not authoritative.

## Board Format

### Columns

| Column | Description |
|---|---|
| Task | Title with link to the task directory |
| Status | Emoji + text status (see statuses below) |
| Depends&nbsp;on | Task references that must complete before this task can start (see [Task References](#task-references-depends_on)) |
| Branch | Branch name and/or worktree path |
| Agent | Model name (details like run_id, machine in the task's own README) |
| Requester | Human who requested or is responsible for the task |
| Time | Start timestamp for in-progress; start + duration for terminal states |

### Statuses

| Emoji | Status | Description |
|---|---|---|
| 📋 | `planning` | Task is being defined, requirements are being gathered |
| ⏳ | `queued` | Task is fully defined and ready for an agent to pick up |
| 🔵 | `in_progress` | An agent has claimed and is actively working on the task |
| 🟡 | `blocked` | Was in progress but hit a wall — waiting on human input, external dependency, or other non-task blocker |
| ✅ | `complete` | Task finished successfully |
| ❌ | `failed` | Agent attempted the task but encountered an unrecoverable error |
| ⛔ | `aborted` | Task was deliberately stopped by a human or system decision |

### Status lifecycle

```
planning → queued → in_progress → complete
                        ↓
                      blocked → in_progress  (when unblocked)
                        ↓
                      aborted

              in_progress → failed
              in_progress → aborted
```

Note: `queued` tasks with unfulfilled `depends_on` cannot be claimed — they are implicitly waiting, but their status remains `queued` (not `blocked`). `blocked` is reserved for tasks that were in progress and got stuck for reasons beyond task dependencies.

## Example

### Root board (`tasks/README.md`)

| Task | Status | Depends&nbsp;on | Branch | Agent | Requester | Time |
|---|---|---|---|---|---|---|
| [setup-db](setup-db/) | ✅&nbsp;`complete` | — | `synchestra/setup-db` | Sonnet&nbsp;4.5 | @alex | 2026-03-12<br>10:15 (4m32s) |
| [implement-api](implement-api/) | 🔵&nbsp;`in_progress` | setup-db | `synchestra/implement-api` | Opus&nbsp;4 | @alex | 2026-03-12<br>10:22 |
| [write-tests](write-tests/) | ⏳&nbsp;`queued` | implement-api | — | — | @alex | — |
| [deploy-staging](deploy-staging/) | ⏳&nbsp;`queued` | implement-api,<br>write-tests | — | — | @alex | — |

### Sub-task board (`tasks/implement-api/README.md`)

| Task | Status | Depends&nbsp;on | Branch | Agent | Requester | Time |
|---|---|---|---|---|---|---|
| [define-schema](define-schema/) | ✅&nbsp;`complete` | — | `synchestra/implement-api` | Sonnet&nbsp;4.5 | @alex | 2026-03-12<br>10:22 (2m10s) |
| [endpoints](endpoints/) | 🔵&nbsp;`in_progress` | define-schema | `synchestra/implement-api` | Opus&nbsp;4 | @alex | 2026-03-12<br>10:25 |
| [validation](validation/) | ⏳&nbsp;`queued` | define-schema | — | — | @alex | — |

## Task References (`depends_on`)

Task references in `depends_on` follow a relative-path model — like file paths. Resolution depends on the referencing context:

### Sibling reference (same parent)

A task can reference a sibling by its slug alone:

```
subtask-2 references subtask-1 → depends_on: subtask-1
```

### Cousin reference (different parent, same project)

Use a relative path from the referencing task's parent:

```
task-2/subtask-1 references task-1/subtask-2 → depends_on: task-1/subtask-2
```

### Cross-project reference (external)

Use the fully qualified URL to the task directory in the external project's repo:

```
depends_on: https://github.com/org/repo/synchestra/projects/project-id/tasks/task-1/subtask-2
```

### Resolution rules

1. A bare slug (no `/`) resolves against siblings in the same parent directory.
2. A relative path (contains `/`) resolves from the project's `tasks/` root.
3. A URL resolves as an external cross-project reference.

## Updating the Board

The board should be updated via the [Synchestra CLI](../cli/README.md) or API — not manually edited. Manual edits are possible but not advisable as they may break micro-task workflows that depend on board state transitions.

The [`synchestra task` commands](../cli/task/README.md) handle board updates atomically as part of their commit-and-push flow:

| Board transition | CLI command | Skill |
|---|---|---|
| → `planning` | [`task new`](../cli/task/new/README.md) | [synchestra-task-new](../../../skills/synchestra-task-new/README.md) |
| `planning` → `queued` | [`task enqueue`](../cli/task/enqueue/README.md) (or `task new --enqueue`) | [synchestra-task-enqueue](../../../skills/synchestra-task-enqueue/README.md) |
| `queued` → `in_progress` | [`task claim`](../cli/task/claim/README.md) + [`task start`](../cli/task/start/README.md) | [synchestra-claim-task](../../../skills/synchestra-claim-task/README.md), [synchestra-task-start](../../../skills/synchestra-task-start/README.md) |
| `in_progress` → `complete` | [`task complete`](../cli/task/complete/README.md) | [synchestra-task-complete](../../../skills/synchestra-task-complete/README.md) |
| `in_progress` → `failed` | [`task fail`](../cli/task/fail/README.md) | [synchestra-task-fail](../../../skills/synchestra-task-fail/README.md) |
| `in_progress` → `blocked` | [`task block`](../cli/task/block/README.md) | [synchestra-task-block](../../../skills/synchestra-task-block/README.md) |
| `blocked` → `in_progress` | [`task unblock`](../cli/task/unblock/README.md) | [synchestra-task-unblock](../../../skills/synchestra-task-unblock/README.md) |
| → `aborted` | [`task aborted`](../cli/task/aborted/README.md) | [synchestra-task-aborted](../../../skills/synchestra-task-aborted/README.md) |
| (query) | [`task status`](../cli/task/status/README.md), [`task list`](../cli/task/list/README.md) | [synchestra-task-status](../../../skills/synchestra-task-status/README.md), [synchestra-task-list](../../../skills/synchestra-task-list/README.md) |

The board in the parent README is the **source of truth**. A task may duplicate its own status in its own README for convenience, but on conflict the parent board wins.

## Claiming a Task (Optimistic Locking)

The board doubles as the claim mechanism — no separate lock protocol needed. Agents use the [`synchestra-claim-task`](../../../skills/synchestra-claim-task/README.md) skill (which wraps [`task claim`](../cli/task/claim/README.md)) to handle this flow automatically. The underlying protocol:

1. Agent pulls latest main, scans the board for `queued` tasks with fulfilled `depends_on`.
2. Agent creates a new branch.
3. On that branch, agent updates the board row: status → `in_progress`, fills in Branch, Agent, Time.
4. Agent attempts to merge/push to main.
5. **If the push succeeds** — the task is claimed. Agent proceeds with the work.
6. **If the push fails** — CLI parses the git diff to check whether the conflict involves the claimed task's row (identified by task ID). See [Conflict Resolution on Claim](#conflict-resolution-on-claim).

```
Agent A                          Agent B
   │                                │
   ├─ pull main                     ├─ pull main
   ├─ see task-X queued             ├─ see task-X queued
   ├─ branch: claim-task-X          ├─ branch: claim-task-X
   ├─ update row → in_progress      ├─ update row → in_progress
   ├─ push → SUCCESS ✓              │
   │                                ├─ push → CONFLICT ✗
   ├─ start working                 ├─ discard branch
   │                                ├─ pick next task or exit
```

### Conflict Resolution on Claim

When a push fails, the CLI parses the git diff and checks whether the conflict involves the row for the task being claimed (matched by task ID):

- **Task row is conflicted** — another agent claimed the same task. Discard the branch, move to the next available task or exit.
- **Task row is NOT conflicted** — the conflict is from a different change (another agent claiming a different task, a formatter run, etc.). The CLI auto-resolves the merge and retries the push.

In practice, a claim attempt only changes a single row, so conflicts should almost exclusively come from another agent claiming the same task. But the check is valuable as a safety net — for example, if a formatter ran and reformatted the table, the row may be in conflict even though the task is still claimable. The CLI can auto-resolve formatting-only conflicts and proceed with the claim.

### Why the board is source of truth

Using the parent board as the single claim point means:
- Conflict detection is automatic via git — no external lock service needed.
- All state transitions are visible in one place — humans and agents read the same board.
- The claim-and-push protocol reduces to "edit a markdown table row and push."

## Multi-line cells

Table cells use `<br>` for line breaks where needed (e.g., timestamps, dependency lists). This is supported in GitHub-flavored markdown.

## Recently Finished

Tasks that reach a terminal status (`completed`, `failed`, `aborted`) are automatically moved from the active board to a **Recently Finished** sub-section below it. This keeps the active board focused on work that needs attention.

### Format

The "Recently Finished" section uses the same table columns as the active board:

```markdown
### Recently Finished

| Task | Status | Depends&nbsp;on | Branch | Agent | Requester | Time |
|---|---|---|---|---|---|---|
| [setup-db](setup-db/) | ✅&nbsp;`complete` | — | `synchestra/setup-db` | Sonnet&nbsp;4.5 | @alex | 2026-03-12<br>10:15 (4m32s) |
| [fix-typo](fix-typo/) | ❌&nbsp;`failed` | — | `synchestra/fix-typo` | Haiku&nbsp;4.5 | @alex | 2026-03-12<br>10:20 (1m05s) |
```

### Retention

The number of tasks shown in "Recently Finished" is configurable in project settings. Options:

| Setting | Description | Default |
|---|---|---|
| `board.recently_finished.limit` | Maximum number of tasks to show | `10` |
| `board.recently_finished.hours` | Alternative: show tasks finished within the last N hours | — |

If both are set, `limit` takes precedence. When a task exceeds the retention window, it is removed from the "Recently Finished" section entirely — it remains accessible via its task directory and through [`task list`](../cli/task/list/README.md) / [`task info`](../cli/task/info/README.md).

## Interaction with Development Plans

See [Spec-to-Execution Pipeline](../../architecture/spec-to-execution.md) for the full architectural view of how features, plans, and tasks connect across repository boundaries.

Tasks generated from a [development plan](../development-plan/README.md) appear on the board like any other task. Each task's README carries a back-reference to its plan and plan step, but the board itself is unaware of plans — it tracks task status regardless of how tasks were created.

The development plan feature provides a derived status view (`synchestra plan status`) that reads plan step references from tasks and aggregates board status into a flat, plan-oriented progress report. See [Development Plan: Derived status view](../development-plan/README.md#derived-status-view).

## Outstanding Questions

- What is the exact task directory structure? (e.g., `tasks/{task-slug}/README.md` with YAML frontmatter for machine-readable status?)
- Should the `Requester` field support teams/groups or only individual humans?
- Should the `claimed` status have a separate row state or be folded into the claiming user's identity?
