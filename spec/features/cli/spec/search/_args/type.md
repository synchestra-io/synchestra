# --type

Filters search results by specification resource type.

| Detail | Value |
|---|---|
| Type | String (enum) |
| Required | No |
| Default | (none — all types included) |

## Syntax

```bash
synchestra spec search <query> --type <type>
```

## Valid Values

| Value | Matches files under | Description |
|---|---|---|
| `feature` | `spec/features/` | Feature specifications |
| `plan` | `spec/plans/` | Development plans |
| `proposal` | `spec/features/*/proposals/` | Feature proposals |

## Behavior

Files are classified by their location in the directory tree. Only files matching the specified type are searched.

When combined with `--feature`, the type filter is applied after scoping. For example, `--feature cli/task --type plan` searches plans that reference features under `cli/task` (since plans live under `spec/plans/`, not under the feature directory).

## Examples

```bash
# Search only plans
synchestra spec search "phase 2" --type plan

# Search only feature specs
synchestra spec search "exit code" --type feature

# Search proposals for a specific feature
synchestra spec search "alternative" --feature cli/task --type proposal
```

## Outstanding Questions

None at this time.
