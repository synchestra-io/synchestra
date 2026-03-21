# CLI Arguments: `spec search`

**Parent:** [spec _args](../../_args/README.md)

Arguments specific to `synchestra spec search`.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`--feature`](feature.md) | String | No | Scope search to a feature subtree (e.g., `cli/task`) |
| [`--section`](section.md) | String | No | Restrict search to named markdown sections (e.g., `Outstanding Questions`) |
| [`--status`](status.md) | String | No | Include only files belonging to features with the given status |
| [`--type`](type.md) | String (enum) | No | Filter by resource type: `feature`, `plan`, `proposal` |
| [`--context`](context.md) | Integer | No | Lines of context around each match (default: `2`) |
| [`--refs`](refs.md) | Boolean (flag) | No | Enrich results with dependency and reverse-dependency context |

Global arguments also accepted: [`--format`](../../../_args/format.md), [`--project`](../../../_args/project.md).

## Outstanding Questions

None at this time.
