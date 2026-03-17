# --spec-repo

Reference to the project's spec repository.

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

Identifies the spec repository for the project. The spec repo holds `synchestra-spec.yaml` (the full project definition), feature specifications, architecture documents, and product documentation.

Accepts a full git URL (`https://github.com/org/repo`, `git@github.com:org/repo`) or a short path (`github.com/org/repo`). Both forms resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk.

For `project new`, this is the repository where `synchestra-spec.yaml` will be created. For `project set`, this re-points the project to a different spec repo.

## Examples

```bash
# Full URL
synchestra project new --spec-repo https://github.com/acme/acme-spec \
  --state-repo https://github.com/acme/acme-synchestra \
  --code-repo https://github.com/acme/acme-api

# Short path
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --code-repo github.com/acme/acme-api

# Re-point spec repo
synchestra project set --spec-repo github.com/acme/acme-spec-v2
```

## Outstanding Questions

None at this time.
