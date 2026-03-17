# --code-repo

Reference to a code repository.

| Detail | Value |
|---|---|
| Type | String (repo reference), repeatable |
| Required | Yes (`new`, `code add`, `code remove`) |
| Default | — |

## Supported by

| Command |
|---|
| [`project new`](../new/README.md) |
| [`project code add`](../code/add/README.md) |
| [`project code remove`](../code/remove/README.md) |

## Description

Identifies a code repository — where agents create branches and push implementation changes. A project must have at least one code repo.

Accepts a full git URL (`https://github.com/org/repo`, `git@github.com:org/repo`) or a short path (`github.com/org/repo`). Both forms resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk.

This flag is repeatable — pass it multiple times to specify multiple code repos in a single command.

Values are stored as origin URLs in the `repos` list of `synchestra-spec.yaml`.

## Examples

```bash
# Multiple code repos at project creation
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --code-repo github.com/acme/acme-api \
  --code-repo github.com/acme/acme-web

# Add a code repo
synchestra project code add --code-repo github.com/acme/acme-infra

# Remove a code repo
synchestra project code remove --code-repo github.com/acme/acme-web
```

## Outstanding Questions

None at this time.
