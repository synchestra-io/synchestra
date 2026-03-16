# Synchestra repository instructions

## Build, test, and lint commands

This repository does not define local build, test, or lint commands. There is no `package.json`, `go.mod`, `Makefile`, GitHub Actions workflow, or application source tree here; this repo is the specification and skill-definition source of truth for Synchestra.

Do not invent verification commands for this repo. Validate documentation changes by checking the affected Markdown files against the surrounding specs and conventions.

If you need runnable implementation details, the root `README.md` points to the execution repos:

- `synchestra-go` for the CLI, daemon, server, and HTTP API implementation
- `synchestra-app` for the web UI

## High-level architecture

This repository describes Synchestra more than it implements it. Read it as a layered product definition:

- `spec/` is the technical source of truth. Use it for behavior, data model, task lifecycle, CLI semantics, and repository layout.
- `skills/` packages the CLI into agent-facing skills. Each skill is a concrete wrapper around a single `synchestra` command and links back to the relevant CLI spec.
- `docs/` contains user-facing explanations of the platform and API surface. Use it when you need the conceptual stack or public interface rather than internal feature requirements.

Within that structure, a few files anchor the big picture:

- `README.md` explains Synchestra as a git-backed coordination layer for multi-platform AI agents. It also defines the key ideas: hierarchical task trees, naming conventions as API, git as the database, token-efficient context loading, and claim-and-push concurrency.
- `spec/features/project-definition/README.md` defines the three repository types (state, spec, code), the two supported layouts for spec repositories (dedicated or multi-project), and the `synchestra-spec.yaml` contract. The state repository (`{project}-synchestra`) is always separate and holds only tasks and coordination state.
- `spec/features/cli/README.md` defines the canonical CLI contract. The `synchestra` CLI is the shared interface for both agents and humans, and mutation commands are expected to perform atomic commit-and-push operations.
- `spec/features/agent-skills/README.md` defines how skills are structured and distributed. Skills do not replace the CLI; they standardize when to call it, which parameters to pass, and how to interpret exit codes.
- `spec/features/task-status-board/README.md` defines the markdown table claiming mechanism for optimistic locking, including the conflict resolution protocol for concurrent claims.
- `docs/features/README.md` captures the conceptual feature stack: state synchronization at the base, then agent coordination and progress reporting, then workflow orchestration, with human steering on top.
- `docs/api/README.md` documents the public REST API that mirrors the platform capabilities, even though the implementation lives in `synchestra-go`.

When working in this repo, treat specs as normative and docs as explanatory. If a skill README and a feature spec disagree, reconcile the change against the CLI or feature spec instead of editing one file in isolation.

## Key conventions

Every directory must have a `README.md`. This is a repository-wide rule, not a suggestion. If you create a directory, add its README in the same change.

Every `README.md` must include an `Outstanding Questions` section. If there are no open questions, write `None at this time.`

Any `README.md` that has child directories must summarize each immediate child after its index table. Keep those summaries brief and useful so an agent can understand the tree without opening every child.

The directory tree is part of the model. Nesting is meaningful: feature directories contain sub-features, task directories contain sub-tasks, and the structure itself is how agents navigate the system.

Naming conventions are part of the API. Paths like `spec/features/`, `skills/`, and `synchestra/projects/{project}/tasks/` are not incidental; they are how Synchestra expects humans and agents to discover requirements, skills, and work queues.

Skill directories follow a fixed pattern: one skill per CLI action, stored at `skills/{skill-name}/README.md`. When editing or adding a skill, preserve the established structure: when to use it, command, parameters, exit codes, examples, and notes.

The CLI contract is consistent across commands. Exit codes have shared meanings (`0` success, `1` conflict, `2` invalid arguments, `3` not found, `4` invalid state transition, `10+` unexpected error), and mutation commands are described as atomic commit-and-push operations while read commands pull first for freshness.

Task state names are canonical. Use the statuses defined in `spec/features/cli/README.md`: `planning`, `queued`, `claimed`, `in_progress`, `completed`, `failed`, `blocked`, and `aborted`. `abort_requested` is a flag, not a standalone status.

This repo often describes systems that are implemented elsewhere. Before adding operational details, check whether the content belongs here as specification/documentation or in a runtime repo such as `synchestra-go` or `synchestra-app`.
