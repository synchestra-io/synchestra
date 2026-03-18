# Code Repository

Code repositories hold the **implementation** — source code, tests, configuration, infrastructure definitions. There can be one or many per project.

Agents work in code repos to fulfill tasks defined in the state repo, following specifications from the spec repo. The Synchestra CLI uses a consistent branch naming convention (`synchestra/{task-slug}`) across all affected repositories, making it easy to trace related changes.

## Why It's Separate

Code repos already exist — they're the user's actual project. Synchestra doesn't dictate their structure beyond requiring a config file and the `synchestra/{task-slug}` branch naming convention for coordinated work. Keeping code repos independent means:

- Existing CI/CD pipelines, branch protections, and review workflows continue unchanged.
- Multiple code repos can be part of the same Synchestra project (frontend, backend, infrastructure).
- Teams keep full control over their code organization.

## Naming Convention

User's choice. Synchestra does not impose naming on code repos.

For a project with multiple code repos:

```
acme-api/          # Backend code repo
acme-web/          # Frontend code repo
acme-infra/        # Infrastructure code repo
acme/              # Spec repo (with synchestra-spec-repo.yaml)
acme-synchestra/   # State repo (tasks, coordination)
```

## Example Structure

```
acme-api/
  synchestra-code-repo.yaml           # Lists spec repos this code repo implements
  src/
    ...
```

## Rules

The following rules are mandatory for every code repository managed by Synchestra.

1. **Config file** — The code repo root MUST contain `synchestra-code-repo.yaml` listing spec repos this code implements.

2. **Branch naming** — Coordinated work MUST use the `synchestra/{task-slug}` branch naming convention.

3. **No structural requirements** — Synchestra does not dictate internal structure beyond the config file and branch convention. Existing CI/CD, branch protections, and review workflows remain unchanged.

## Outstanding Questions

None at this time.
