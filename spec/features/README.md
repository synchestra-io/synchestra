# Features: Synchestra

Feature specifications for the Synchestra project, managed by Synchestra.

## Index

| Feature | Status | Description |
|---|---|---|
| [micro-tasks](micro-tasks/README.md) | Conceptual | Pre/post prompt micro-task chains and background automation |
| [cross-repo-sync](cross-repo-sync/README.md) | Conceptual | Cross-repository branching, task coordination, and merge strategy |
| [model-selection](model-selection/README.md) | Conceptual | Smart model routing based on task complexity and configuration |
| [conflict-resolution](conflict-resolution/README.md) | Conceptual | AI-powered merge conflict detection and resolution |
| [outstanding-questions](outstanding-questions/README.md) | Conceptual | Question lifecycle management linked to tasks and features |
| [proposals](proposals/README.md) | Conceptual | Non-normative change requests attached to features with review status and optional tracker linkage |
| [ui](ui/README.md) | Conceptual | Human-facing interfaces for project navigation, proposals, tasks, and workers across web and terminal surfaces |
| [claim-and-push](claim-and-push/README.md) | Conceptual | Distributed task claiming via git push-based optimistic locking |
| [task-status-board](task-status-board/README.md) | Conceptual | Markdown task board in task directory READMEs for at-a-glance status visibility |
| [agent-skills](agent-skills/README.md) | In Progress | Dedicated, focused skills that AI agents use to interact with Synchestra |
| [cli](cli/README.md) | In Progress | The `synchestra` CLI — primary interface for agents and humans |

## Feature Summaries

### [Micro-Tasks](micro-tasks/README.md)

Small, automated steps that run before, after, or in the background of a user's prompt — formatting, validation, cross-reference updates, link checks. They keep the project consistent without burning tokens from the main task's context window. Configured per-project or per-module as pre/post/background chains, modeled after GitHub Actions workflow jobs.

### [Cross-Repo Sync](cross-repo-sync/README.md)

Coordinates changes that span multiple repositories. When a task requires edits across repos (e.g., API spec + backend + frontend), Synchestra decomposes the work into sub-tasks, reserves matching branch names across all affected repos, manages dependency order, and handles the integration merge lifecycle.

### [Model Selection](model-selection/README.md)

Routes tasks to the minimal viable model to avoid wasting expensive tokens on mechanical work. Three levels of precedence: user override (CLI/API/UI), configuration rules (`model_class` mapping to platform-specific models), and dynamic assessment where a small model classifies task complexity before routing.

### [Conflict Resolution](conflict-resolution/README.md)

When git merge conflicts occur between concurrent agent operations, Synchestra launches a specialized sub-agent to analyze and resolve the conflict. Three tiers: auto-merge via git rebase, AI-assisted merge that understands change intent from task descriptions, and human escalation with a confidence threshold for ambiguous cases.

### [Outstanding Questions](outstanding-questions/README.md)

Every document maintains a structural "Outstanding Questions" section with a full lifecycle: open → linked (to a task) → resolved → recently resolved → archived. When a linked task completes, a sub-agent evaluates whether the output actually answers the question and resolves it automatically.

### [Proposals](proposals/README.md)

Proposals attach non-normative change requests directly to a feature without changing the feature's current specification. Each proposal has its own status lifecycle, can link to a GitHub issue for MVP, and is excluded from default current-state understanding unless explicitly requested.

### [UI](ui/README.md)

UI defines the human-facing product surfaces for Synchestra. It starts from a project list, then provides project-level navigation into Features, Tasks, and Workers, with MVP flows for feature proposal creation and task creation/enqueueing across both the web app and terminal UI.

### [Claim-and-Push](claim-and-push/README.md)

Distributed task claiming through git's push semantics. Agents claim tasks by committing a status change and pushing — if the push fails, another agent got there first. No central lock server needed. The protocol relies on frequent commits to minimize conflict windows and provide granular audit trail.

### [Task Status Board](task-status-board/README.md)

A markdown table in task directory READMEs that serves as both the visibility layer and the claim mechanism. The board is the source of truth for task state — agents claim tasks by updating a row and pushing. Conflicts on the same row indicate a claim collision; the CLI parses diffs by task ID to distinguish collisions from unrelated changes.

### [Agent Skills](agent-skills/README.md)

A set of dedicated, focused skills that AI agents use to interact with Synchestra — claiming tasks, reporting status, updating progress. Each skill wraps a single CLI command with clear trigger conditions, parameters, and exit code handling. Skills are distributed via CLI, MCP server, or direct file access.

### [CLI](cli/README.md)

The `synchestra` command-line interface. Follows a `synchestra <resource> <action>` pattern with consistent exit codes, atomic git commit-and-push for mutations, and both query and update modes. Defines the task status model, valid transitions, and the `abort_requested` flag. Commands are organized as `cli/task/claim/`, `cli/task/status/`, etc.

## Feature dependency graph

```
claim-and-push ← conflict-resolution
       ↑                ↑
cross-repo-sync ────────┘
       ↑
micro-tasks (independent)
model-selection (independent)
outstanding-questions (independent)
proposals (independent)
proposals ← ui → cli
```

`claim-and-push` is foundational — most concurrent features depend on it.

## Outstanding Questions

- Are there features missing from this list that are already described in `docs/features/` but not yet tracked here?
- **Suggested build order:** claim-and-push first (foundational), then outstanding-questions and model-selection (independent, high value), then proposals, then UI once CLI and proposal flows are ready enough to expose, then conflict-resolution, then micro-tasks and cross-repo-sync. Does this align with project priorities?

### Features with outstanding questions:

- [micro-tasks](micro-tasks/README.md): 4 outstanding questions
- [cross-repo-sync](cross-repo-sync/README.md): 4 outstanding questions
- [model-selection](model-selection/README.md): 4 outstanding questions
- [conflict-resolution](conflict-resolution/README.md): 3 outstanding questions
- [outstanding-questions](outstanding-questions/README.md): 3 outstanding questions
- [claim-and-push](claim-and-push/README.md): 3 outstanding questions
- [task-status-board](task-status-board/README.md): 4 outstanding questions
- [agent-skills](agent-skills/README.md): 3 outstanding questions
- [cli](cli/README.md): 3 outstanding questions
