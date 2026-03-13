# Project Definition

How Synchestra discovers and configures projects.

## Repository Types

Synchestra operates with three kinds of repositories. Each has a distinct role:

| Repository type | What it holds | Naming convention | Example |
|---|---|---|---|
| **State repository** | Tasks, claims, coordination state, workflow artifacts | `{project}-synchestra` | `acme-synchestra` |
| **Spec repository** | Requirements, architecture, documentation, `synchestra-project.yaml` | User's choice | `acme`, `acme-spec` |
| **Code repository** (one or more) | Implementation and source code | User's choice | `acme-api`, `acme-web` |

The **spec repository** and **code repositories** can be combined into a single repo (common for smaller projects), but the **state repository** should always be a dedicated, separate repo. The state repo has a very different commit cadence — high-frequency machine commits from agents claiming tasks, updating statuses, and pushing coordination artifacts — and keeping it separate avoids polluting the project's code history.

The naming convention `{project}-synchestra` (suffix style) groups the state repo alongside its sibling repos in alphabetical listings (e.g., `acme-api`, `acme-synchestra`, `acme-web`).

## Project File

Every project managed by Synchestra has a `synchestra-project.yaml` file as its entry point. This file lives in the **spec repository** (or the combined spec+code repository) and references the state repository.

### Mandatory fields

| Field | Description |
|---|---|
| `title` | Human-readable project name |
| `state_repo` | URL of the state repository (e.g., `https://github.com/org/acme-synchestra`) |

### Optional fields

| Field | Default | Description |
|---|---|---|
| `repos` | — | List of target repository URLs (code repos that agents work in) |
| `project_dirs.specifications` | `spec` | Directory for technical specifications (features, architecture, etc.) |
| `project_dirs.documents` | `docs` | Directory for user-facing documentation |

## Repository Layouts

Synchestra supports two layouts for where project files live within the spec repository, depending on whether it manages one or multiple projects.

### Multi-project layout (default)

For spec repositories that manage multiple projects, project files live under the `synchestra/projects/` directory:

```
synchestra/
  projects/
    {project_id}/
      synchestra-project.yaml   # Project configuration
      README.md                 # Project overview

spec/                           # Specifications (default, configurable)
  features/
    ...

docs/                           # Documentation (default, configurable)
  ...
```

The project entry point is `synchestra/projects/{project_id}/synchestra-project.yaml`.

#### Example

```yaml
title: My Service
state_repo: https://github.com/org/my-service-synchestra
repos:
  - https://github.com/org/my-service-api
  - https://github.com/org/my-service-web
```

### Dedicated spec repository layout

For spec repositories dedicated to a single project, project files live at the repository root:

```
synchestra-project.yaml         # Project configuration (at root)
README.md                       # Project overview (at root)
LICENSE
spec/                           # Specifications
  features/
    ...
docs/                           # Documentation
  ...
```

The project entry point is `synchestra-project.yaml` at the repository root.

This layout is appropriate when the entire repository exists to specify one project. There is no `synchestra/projects/` nesting — the repository itself is the project directory.

#### Example

```yaml
title: Synchestra
state_repo: https://github.com/synchestra-io/synchestra-state
repos:
  - https://github.com/synchestra-io/synchestra-go
  - https://github.com/synchestra-io/synchestra-app
```

## State Repository Structure

The state repository contains only Synchestra operational data — no specs, docs, or source code:

```
synchestra-project.yaml         # Minimal project config (title + back-reference to spec repo)
README.md                       # Auto-generated project overview
tasks/                          # Task queue
  task-1/
    README.md                   # Task description, status, assignment
    subtask-1/
      README.md
  task-2/
    README.md
```

## State Repository README

The root `README.md` of a state repository follows this template:

```markdown
# {project_title} — Synchestra State

[Open in Synchestra](https://synchestra.io/app/project?id={state_repo_id})

State repository for the [{project_title}]({spec_repo_url}) project.

This repo is managed by [Synchestra](https://github.com/synchestra-io/synchestra) — it tracks task status, coordination state, and workflow artifacts. For specifications, architecture, and documentation, see the [{project_title}]({spec_repo_url}) repository.
```

### Template variables

| Variable | Source | Example |
|---|---|---|
| `{project_title}` | `title` field from `synchestra-project.yaml` | `Synchestra` |
| `{spec_repo_url}` | URL of the spec repository hosting this project's `synchestra-project.yaml` | `https://github.com/synchestra-io/synchestra` |
| `{state_repo_id}` | GitHub identifier of the state repo (org/repo format) | `github.com/synchestra-io/synchestra-state` |

### Example (for Synchestra itself)

```markdown
# Synchestra — Synchestra State

[Open in Synchestra](https://synchestra.io/app/project?id=github.com/synchestra-io/synchestra-state)

State repository for the [Synchestra](https://github.com/synchestra-io/synchestra) project.

This repo is managed by [Synchestra](https://github.com/synchestra-io/synchestra) — it tracks task status, coordination state, and workflow artifacts. For specifications, architecture, and documentation, see the [Synchestra](https://github.com/synchestra-io/synchestra) repository.
```

### How Synchestra determines the layout

Synchestra checks for a `synchestra-project.yaml` at the repository root. If found, the repository is treated as a dedicated spec repository. Otherwise, it looks under `synchestra/projects/` for the multi-project layout.

## Outstanding Questions

- Should there be an explicit field in `synchestra-project.yaml` to declare the layout, or is auto-detection (root file presence) sufficient?
- Should `synchestra-project.yaml` in the state repo contain a `spec_repo` back-reference field, or is the link only from spec repo → state repo?
- Should code repositories also contain a lightweight `synchestra.yaml` pointer to the state repo for CLI auto-discovery?
