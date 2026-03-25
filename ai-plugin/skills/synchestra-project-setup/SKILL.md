---
name: synchestra-project-setup
description: Guides a user through setting up a new Synchestra project. Use when a user wants to initialize Synchestra, configure project topology, or asks how to get started.
---

# Skill: synchestra-project-setup

Guide a user through configuring and initializing a Synchestra project. This is a conversational skill — gather information through questions, then execute the appropriate CLI commands.

**Feature spec:** [project-setup](../../spec/features/agent-skills/project-setup/README.md)

## When to use

- A user asks to set up or initialize Synchestra in a project
- A user asks how to get started with Synchestra
- You detect no Synchestra configuration in the current repository and the user wants task management
- A user asks about project configuration, topology, or repo layout

## Workflow

### Step 1: Gather project information

Ask the user the following questions. Accept defaults when the user does not have a preference.

1. **Project title** — What should the project be called?
   - Default: derived from the repository's README heading or directory name
   - Used as the display name in Synchestra state

2. **Spec directory** — Where do specification files live?
   - Default: `spec/`
   - This is the root for feature specs, plans, and other structured documents

3. **Source directories** — Which directories contain implementation code?
   - Default: the repository root
   - Examples: `src/`, `lib/`, `pkg/`, `cmd/`

4. **Additional repositories** — Are there other git repos that should be linked as code repos?
   - Default: none
   - Only relevant for multi-repo projects

5. **State storage preference** — How should Synchestra store coordination state?
   - **Option A: Embedded (recommended)** — State lives on an orphan branch in this repo, checked out as a `.synchestra/` worktree. Single repo, zero config files required. Best for most projects.
   - **Option B: Dedicated state repo** — State lives in a separate git repository. Best for multi-repo projects or when state needs different access controls.
   - Default: Option A

### Step 2: Execute setup

Based on the gathered information, run the appropriate commands.

**For Option A (embedded state):**

```bash
synchestra project init --title "<title>"
```

This creates the orphan branch, worktree, and config file. Task commands work immediately — no further configuration needed ([config-less mode](../../spec/features/embedded-state/README.md#config-less-mode)).

**For Option B (dedicated state repo):**

```bash
synchestra project new --title "<title>" --spec-root "<spec_root>"
```

Then, for each additional repository:

```bash
synchestra project code add <repo-url>
```

### Step 3: Verify

Run `synchestra project info` to confirm the project is set up correctly. Check:

- Exit code is `0`
- Project title matches what the user specified
- State store is accessible
- Spec root is correctly configured

Report the result to the user.

## Exit codes

These are the exit codes from the underlying CLI commands:

| Exit code | Meaning | What to do |
|---|---|---|
| `0` | Success (or already initialized — idempotent) | Confirm to user and suggest next steps |
| `1` | Conflict (e.g., embedded init in a repo with dedicated config) | Explain the conflict and ask the user how to proceed |
| `2` | Invalid arguments | Fix the arguments and retry |
| `3` | Not a git repository | Ask the user to navigate to a git repository first |
| `10+` | Unexpected error | Report the error to the user |

## After setup

Suggest these next steps to the user:

1. **Create a task:** Use `synchestra-task-new` to create the first task
2. **Explore features:** Use `synchestra-feature-list` to see existing feature specs
3. **Check status:** Use `synchestra-task-list` to view the task board

## Notes

- **Idempotent:** Both `project init` and `project new` are safe to run multiple times. If the project is already set up, they detect the existing configuration and exit cleanly.
- **Config-less mode:** After Option A setup, task commands work without any config file on the main branch. The CLI detects the `.synchestra/` worktree automatically.
- **Upgrade path:** Users who start with Option A can later switch to a dedicated state repo. The embedded state can be extracted using `synchestra project extract-state` (future command).
- **Do not skip questions.** Even though all questions have defaults, asking them ensures the user understands what is being configured. For experienced users who say "just use defaults," run Option A with derived defaults.
