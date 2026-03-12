# Project Definition

How Synchestra discovers and configures projects.

## Project File

Every project managed by Synchestra has a `synchestra-project.yaml` file as its entry point.

### Mandatory fields

| Field | Description |
|---|---|
| `title` | Human-readable project name |
| `repo` | Repository URL (can include subpath for monorepos) |

### Optional fields

| Field | Default | Description |
|---|---|---|
| `project_dirs.specifications` | `spec` | Directory for technical specifications (features, architecture, etc.) |
| `project_dirs.documents` | `docs` | Directory for user-facing documentation |

## Repository Layouts

Synchestra supports two layouts for where project files live, depending on whether the repository is shared or dedicated to a single project.

### Multi-project layout (default)

For repositories that manage multiple projects (including the main `synchestra` repo itself), project files live under the `synchestra/projects/` directory:

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
repo: https://github.com/org/monorepo/services/my-service
```

### Dedicated project repository layout

For repositories dedicated to a single project, project files live at the repository root:

```
synchestra-project.yaml         # Project configuration (at root)
README.md                       # Project overview (at root)
LICENSE
...
```

The project entry point is `synchestra-project.yaml` at the repository root.

This layout is appropriate when the entire repository exists to manage one project. There is no `synchestra/projects/` nesting — the repository itself is the project directory.

#### Example

```yaml
title: Synchestra
repo: https://github.com/synchestra-io/synchestra
```

### How Synchestra determines the layout

Synchestra checks for a `synchestra-project.yaml` at the repository root. If found, the repository is treated as a dedicated project repo. Otherwise, it looks under `synchestra/projects/` for the multi-project layout.

## Outstanding Questions

- Should there be an explicit field in `synchestra-project.yaml` to declare the layout, or is auto-detection (root file presence) sufficient?
- For dedicated repos, should `spec/` and `docs/` directories be co-located in the same repo or always referenced from the main synchestra repo?
