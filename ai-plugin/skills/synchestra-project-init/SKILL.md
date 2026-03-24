---
name: synchestra-project-init
description: Initializes embedded Synchestra state in the current git repo. Use when setting up a new project for task coordination or when a user wants to start using Synchestra in an existing repo.
---

# Skill: synchestra-project-init

Initialize Synchestra embedded state in the current git repository. Creates an orphan branch for state management and sets up a git worktree at `.synchestra/`. This is the zero-friction way to start using Synchestra — one command, no separate repos.

**CLI reference:** [synchestra project init](../../spec/features/cli/project/init/README.md)

## When to use

- A user wants to start using Synchestra in an existing git repository
- You need task coordination in a project that doesn't have Synchestra set up yet
- You detect `synchestra` CLI is available but no `synchestra.yaml` marker exists in the repo
- A user asks about persistent task state, cross-session continuity, or multi-agent coordination

## Command

```bash
synchestra project init \
  [--title <title>] \
  [--branch <name>] \
  [--no-push]
```

## Parameters

| Parameter | Required | Default | Description |
|---|---|---|---|
| `--title` | No | Derived from README heading or directory name | Project title |
| `--branch` | No | `synchestra-state` | Name of the orphan branch for state storage |
| `--no-push` | No | `false` | Skip pushing the branch to the remote (local-only mode) |

## Exit codes

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Initialization complete (or already initialized — idempotent) | Proceed with task management |
| `1` | Conflict — repo has a dedicated project setup (`synchestra-spec-repo.yaml`) | Use the existing dedicated setup instead |
| `2` | Invalid arguments | Check parameter values and retry |
| `3` | Not a git repository | Navigate to a git repository first |
| `10+` | Unexpected error | Log the error and escalate |

## Examples

### Quick start in any repo

```bash
cd my-project
synchestra project init
```

### With a custom title

```bash
synchestra project init --title "My Awesome Project"
```

### Local-only (no remote push)

```bash
synchestra project init --no-push
```

### Custom state branch name

```bash
synchestra project init --branch my-state
```

## What it creates

| What | Where | Impact on your code |
|---|---|---|
| State branch | `synchestra-state` (orphan) | No shared history with your code branches |
| Worktree | `.synchestra/` | Added to `.gitignore` automatically |
| Marker file | `synchestra.yaml` in repo root | 3-line YAML on your main branch |
| Task board | `.synchestra/tasks/README.md` | On the state branch only |

## Notes

- **Idempotent:** Running `init` when already initialized prints the current status and exits `0`. Safe to run multiple times.
- **Joins existing projects:** If the state branch exists on the remote (e.g., a teammate already ran `init`), it fetches and connects to the existing state rather than creating a new one.
- **Conflict detection:** If the repo already has a `synchestra-spec-repo.yaml` (from `synchestra project new`), `init` exits with an error — embedded and dedicated modes cannot coexist in the same repo.
- **Reversible:** To remove Synchestra, run `git worktree remove .synchestra && git branch -D synchestra-state && rm synchestra.yaml`.

## After initialization

Once initialized, use the task management skills:
- `synchestra-task-new` — create tasks
- `synchestra-task-enqueue` — make tasks available for agents
- `synchestra-claim-task` — claim a task before starting work
- `synchestra-task-start` — begin work on a claimed task
- `synchestra-task-complete` — mark a task as done
- `synchestra-task-list` — view all tasks and their status
