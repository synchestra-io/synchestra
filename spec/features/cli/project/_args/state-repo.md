# --state-repo

Reference to the project's state repository.

| Detail | Value |
|---|---|
| Type | String (repo reference) |
| Required | Yes (`new`), No (`set`) |
| Default | — |

## Supported by

| Command |
|---|
| [`project new`](../new/README.md) |
| [`project set`](../set/README.md) |

## Description

Identifies the state repository for the project. The state repo holds coordination data — tasks, claims, workflow artifacts — and is written to primarily by the Synchestra CLI and agents.

Accepts a full git URL (`https://github.com/org/repo`, `git@github.com:org/repo`) or a short path (`github.com/org/repo`). Both forms resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk.

For `project new`, this is the repository where `synchestra-state.yaml` will be created. For `project set`, this re-points the project to a different state repo — if the new state repo has no `synchestra-state.yaml`, one is created; if it has one pointing to a different spec repo, the command fails with exit code `1`.

## Examples

```bash
# Create project
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --target-repo github.com/acme/acme-api

# Change state repo
synchestra project set --state-repo github.com/acme/acme-synchestra-v2
```

## Outstanding Questions

None at this time.
