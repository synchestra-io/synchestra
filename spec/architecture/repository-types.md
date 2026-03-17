# Repository Types

Synchestra operates with three kinds of repositories. Each has a distinct role, commit cadence, and audience. Understanding the separation is key to working with Synchestra effectively.

## Overview

| Repository type | What it holds | Who writes to it | Commit cadence |
|---|---|---|---|
| **Spec repository** (one or more) | Requirements, architecture, documentation, `synchestra-spec.yaml` | Humans, agents (reviewed) | Low — deliberate, reviewed changes |
| **State repository** | Tasks, claims, coordination state, workflow artifacts | Synchestra CLI, agents (automated) | High — frequent machine commits |
| **Code repository** (one or more) | Implementation and source code, `synchestra-code.yaml` | Developers, agents | Medium — feature branches, PRs |

## Spec Repository

The spec repository is the **source of truth for what should be built**. It contains:

- **Feature specifications** — what the system should do, acceptance criteria, design decisions
- **Architecture documents** — how the system is structured, trade-offs, constraints
- **Product documentation** — user-facing explanations, API guides, tutorials
- **Project configuration** — [`synchestra-spec.yaml`](../features/project-definition/README.md), which defines the project and references the state and code repositories

The spec repo is where humans and agents collaborate on *decisions*. Changes are deliberate and typically reviewed. The directory structure mirrors the product's feature tree — agents navigate `spec/features/` to understand requirements before starting work.

### Why it's separate

Specifications have a fundamentally different lifecycle from both code and coordination state. They change when the product direction changes, not when an agent claims a task or a build completes. Keeping specs in their own repo (or combined with code for smaller projects) ensures that the product definition isn't buried under machine-generated commits.

### Naming convention

User's choice. Common patterns: `{project}`, `{project}-spec`, or combined with code in a single repo.

### Example structure

```
acme/
  synchestra-spec.yaml         # Project config → references state repo + code repos
  README.md
  spec/
    features/
      user-auth/
        README.md
      payment-flow/
        README.md
    architecture/
      ...
  docs/
    ...
```

## State Repository

The state repository is the **coordination hub**. It contains only Synchestra operational data:

- **Task queue** — task descriptions, statuses, assignments, sub-task hierarchies
- **Coordination state** — which agent claimed what, when, what's blocked
- **Workflow artifacts** — sync markers, conflict resolution records

The state repo is written to primarily by the Synchestra CLI and agents. Every `synchestra task claim`, `synchestra task complete`, and similar command results in an atomic commit-and-push to this repo. This means it accumulates commits rapidly — dozens or hundreds per day on active projects.

### Why it's separate

This is the most important separation. The state repo **must** be dedicated because:

1. **Commit noise.** Machine-generated coordination commits (task claims, status transitions) would drown out meaningful human commits in a code or spec repo's git history.
2. **Conflict surface.** Agents competing for tasks push frequently. Keeping this traffic in a dedicated repo avoids merge conflicts with code or spec changes.
3. **Access patterns.** The state repo is pulled and pushed constantly by the CLI. Code and spec repos follow normal development workflows (feature branches, PRs). Mixing them would create unnecessary contention.
4. **Permissions.** Agents need write access to the state repo to claim and update tasks. You may not want those same agents to have unrestricted write access to your production code or specifications.

### Naming convention

`{project}-synchestra` — suffix style groups the state repo alongside its sibling repos in alphabetical listings (e.g., `acme-api`, `acme-synchestra`, `acme-web`).

For Synchestra's own development, the state repo is named `synchestra-state` as a special case to avoid the awkward `synchestra-synchestra`.

### Example structure

```
acme-synchestra/
  synchestra-state.yaml            # Lists spec repos managed by this state repo
  README.md                        # Auto-generated project overview
  tasks/
    implement-auth/
      README.md                   # Task description, status, assignment
      add-jwt-middleware/
        README.md
      write-login-endpoint/
        README.md
    fix-payment-bug/
      README.md
```

## Code Repository

Code repositories hold the **implementation** — source code, tests, configuration, infrastructure definitions. There can be one or many per project.

Agents work in code repos to fulfill tasks defined in the state repo, following specifications from the spec repo. The Synchestra CLI uses a consistent branch naming convention (`synchestra/{task-slug}`) across all affected repositories, making it easy to trace related changes.

### Why it's separate

Code repos already exist — they're the user's actual project. Synchestra doesn't dictate their structure beyond requiring agents to use the `synchestra/{task-slug}` branch naming convention for coordinated work. Keeping code repos independent means:

- Existing CI/CD pipelines, branch protections, and review workflows continue unchanged
- Multiple code repos can be part of the same Synchestra project (frontend, backend, infrastructure)
- Teams keep full control over their code organization

### Naming convention

User's choice. Synchestra doesn't impose naming on code repos.

### Example structure

```
acme-api/
  synchestra-code.yaml           # Lists spec repos this code repo implements
  src/
    ...
```

For a project with multiple code repos:

```
acme-api/          # Backend code repo
acme-web/          # Frontend code repo
acme-infra/        # Infrastructure code repo
acme/              # Spec repo (with synchestra-spec.yaml)
acme-synchestra/   # State repo (tasks, coordination)
```

## How They Connect

The **spec repository** is the anchor. Its `synchestra-spec.yaml` defines the project and references the other repositories:

```yaml
title: Acme Platform
state_repo: https://github.com/acme/acme-synchestra
repos:
  - https://github.com/acme/acme-api
  - https://github.com/acme/acme-web
  - https://github.com/acme/acme-infra
```

The **state repository** contains `synchestra-state.yaml` listing all spec repos that share this state repo:

```yaml
spec_repos:
  - https://github.com/acme/acme
  - https://github.com/acme/acme-rehearse
```

**Code repositories** contain `synchestra-code.yaml` listing all spec repos they implement:

```yaml
spec_repos:
  - https://github.com/acme/acme
```

Spec and code repos have a **many-to-many** relationship: one spec can be implemented by multiple code repos, and one code repo can implement multiple specs. Similarly, multiple spec repos can share a single state repo.

### Typical workflow

See [Spec-to-Execution Pipeline](spec-to-execution.md) for the full lifecycle with diagrams.

1. Human or agent reads a **spec** from the spec repo to understand what to build
2. Agent claims a **task** in the state repo via `synchestra task claim`
3. Agent creates a `synchestra/{task-slug}` branch in the relevant **code repo(s)**
4. Agent implements the feature, commits, and pushes
5. Agent marks the task complete in the **state repo** via `synchestra task complete`

## Combining Repositories

For smaller projects, the spec and code repos can be combined into a single repository. The `synchestra-spec.yaml` lives at the root alongside `spec/`, `docs/`, and the source code. The state repo remains separate.

```
acme/                             # Combined spec + code repo
  synchestra-spec.yaml
  synchestra-code.yaml
  spec/
    features/
      ...
  docs/
    ...
  src/                            # Source code
    ...

acme-synchestra/                  # State repo (always separate)
  tasks/
    ...
```

What **cannot** be combined: the state repo. Even for the simplest projects, coordination state belongs in its own dedicated repository.

## Outstanding Questions

- How should the CLI resolve which project a code repo belongs to if the developer is working in the code repo and hasn't explicitly set `--project`? (The code repo's `synchestra-code.yaml` lists spec repos, which in turn reference the state repo.)
