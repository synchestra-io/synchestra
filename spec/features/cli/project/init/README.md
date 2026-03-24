# Command: `synchestra project init`

**Parent:** [project](../README.md)

## Synopsis

```
synchestra project init [--title <title>] [--branch <name>] [--no-push]
```

## Description

Initializes Synchestra embedded state in the current git repository. Creates an orphan branch for state management and sets up a git worktree at `.synchestra/`. This is the zero-friction alternative to `synchestra project new` — no separate repos, no YAML linking, just one command in any existing git repo.

See [Embedded State](../../../../features/embedded-state/README.md) for the full design.

## Parameters

| Parameter | Required | Default | Description |
|---|---|---|---|
| `--title` | No | Derived from repo name or README heading | Project title |
| `--branch` | No | `synchestra-state` | Name of the orphan branch for state |
| `--no-push` | No | `false` | Skip pushing the branch to the remote (local-only mode) |

## Exit Codes

| Exit code | Meaning |
|---|---|
| `0` | Initialization complete (or already initialized — idempotent) |
| `2` | Invalid arguments |
| `3` | Not a git repository |
| `10+` | Unexpected error |

## Behaviour

1. Verify current directory is inside a git repository; exit `3` if not
2. Check if `.synchestra/` worktree already exists and is valid
   - If valid: print status, exit `0` (idempotent)
   - If stale (pruned): clean up and continue
3. Check if the orphan branch exists on the remote (`origin/{branch}`)
   - **Yes:** fetch and create worktree from existing branch (joining existing project)
   - **No:** continue to step 4
4. Check if the orphan branch exists locally
   - **Yes:** create worktree from local branch
   - **No:** continue to step 5
5. Create orphan branch:
   - `git checkout --orphan {branch}`
   - Remove all tracked files (`git rm -rf .`)
   - Write `synchestra-state.yaml` with project config
   - Write `tasks/README.md` with empty task board
   - Write root `README.md` (auto-generated project overview)
   - Commit: `"Initialize Synchestra state"`
   - Return to the original branch
6. Create worktree: `git worktree add .synchestra {branch}`
7. Add `.synchestra` to `.gitignore` if not already present (on the current branch)
8. Write `synchestra.yaml` marker to repo root (on the current branch) if not present
9. Unless `--no-push`: push the orphan branch to origin
10. Print summary: branch name, worktree path, sync status

## Examples

```bash
# Quick start in any repo
cd my-project
synchestra project init

# With a custom title
synchestra project init --title "My Awesome Project"

# Local-only (no remote push)
synchestra project init --no-push
```

## Relationship to `synchestra project new`

| | `project init` | `project new` |
|---|---|---|
| Use case | Single repo, quick start | Multi-repo, full setup |
| State location | Orphan branch in same repo | Dedicated state repo |
| Repos required | 1 (current) | 3+ (spec, state, code) |
| Config files | `synchestra.yaml` + state on orphan branch | `synchestra-spec-repo.yaml`, `synchestra-state-repo.yaml`, `synchestra-code-repo.yaml` |

## Outstanding Questions

- Should `init` detect an existing `synchestra-spec-repo.yaml` (from `project new`) and warn/fail to prevent mixing modes?
- Should there be a `--force` flag to re-initialize even when a valid worktree exists?
