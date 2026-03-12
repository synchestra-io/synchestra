# Synchestra Meta

This is the hardcoded `synchestra/` directory — the orchestration layer for all projects managed by this repository.

## Structure

```
synchestra/
  projects/
    {project_id}/
      synchestra-project.yaml  # Project configuration (entry point)
      README.md                # Project overview

spec/                          # Specifications (default, configurable per project)
  features/                    # Feature specifications
    micro-tasks/
    cross-repo-sync/
    ...

docs/                          # User-facing documentation (default, configurable per project)
  features/                    # Product feature docs
    agent-coordination.md
    ...
```

## Project File Format

The project entry point is `synchestra/projects/{project_id}/synchestra-project.yaml`.

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

### Minimal example

```yaml
title: Synchestra
repo: https://github.com/synchestra-io/synchestra
```

### Full example

```yaml
title: Synchestra
repo: https://github.com/synchestra-io/synchestra
project_dirs:
  specifications: spec
  documents: docs
```

### Monorepo example

```yaml
title: My Service
repo: https://github.com/org/monorepo/services/my-service
project_dirs:
  specifications: design
  documents: docs
```

## Outstanding Questions

None at this time.

### Children with outstanding questions:

- [projects/](projects/README.md): 0 outstanding questions
  - [synchestra](projects/synchestra/README.md): 3 outstanding questions
    - [features/](../spec/features/README.md): 3 outstanding questions
      - [micro-tasks](../spec/features/micro-tasks/README.md): 4 outstanding questions
      - [cross-repo-sync](../spec/features/cross-repo-sync/README.md): 4 outstanding questions
      - [model-selection](../spec/features/model-selection/README.md): 4 outstanding questions
      - [conflict-resolution](../spec/features/conflict-resolution/README.md): 3 outstanding questions
      - [outstanding-questions](../spec/features/outstanding-questions/README.md): 3 outstanding questions
      - [claim-and-push](../spec/features/claim-and-push/README.md): 3 outstanding questions
