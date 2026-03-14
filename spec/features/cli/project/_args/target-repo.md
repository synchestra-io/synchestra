# --target-repo

Reference to a target (code) repository.

| Detail | Value |
|---|---|
| Type | String (repo reference), repeatable |
| Required | Yes (`new`, `target add`, `target remove`) |
| Default | — |

## Supported by

| Command |
|---|
| [`project new`](../new/README.md) |
| [`project target add`](../target/add/README.md) |
| [`project target remove`](../target/remove/README.md) |

## Description

Identifies a target (code) repository — where agents create branches and push implementation changes. A project must have at least one target repo.

Accepts a full git URL (`https://github.com/org/repo`, `git@github.com:org/repo`) or a short path (`github.com/org/repo`). Both forms resolve to `{repos_dir}/{hosting}/{org}/{repo}` on disk.

This flag is repeatable — pass it multiple times to specify multiple target repos in a single command.

Values are stored as origin URLs in the `repos` list of `synchestra-spec.yaml`.

## Examples

```bash
# Multiple targets at project creation
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --target-repo github.com/acme/acme-api \
  --target-repo github.com/acme/acme-web

# Add a target repo
synchestra project target add --target-repo github.com/acme/acme-infra

# Remove a target repo
synchestra project target remove --target-repo github.com/acme/acme-web
```

## Outstanding Questions

None at this time.
