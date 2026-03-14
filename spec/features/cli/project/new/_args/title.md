# --title

Human-readable project title.

| Detail | Value |
|---|---|
| Type | String |
| Required | No |
| Default | Derived from spec repo `README.md` heading, or repo identifier |

## Supported by

| Command |
|---|
| [`project new`](../README.md) |

## Description

Sets the `title` field in `synchestra-spec.yaml`. If omitted, the CLI derives the title using this fallback chain:

1. First `# heading` in the spec repo's `README.md` (if it exists)
2. The repo identifier (e.g., `acme-spec` from `github.com/acme/acme-spec`)

## Examples

```bash
# Explicit title
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --target-repo github.com/acme/acme-api \
  --title "Acme Platform"

# Derived title (from README.md or repo name)
synchestra project new --spec-repo github.com/acme/acme-spec \
  --state-repo github.com/acme/acme-synchestra \
  --target-repo github.com/acme/acme-api
```

## Outstanding Questions

None at this time.
