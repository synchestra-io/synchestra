# Arguments: `synchestra feature new`

Command-specific arguments for [`feature new`](../README.md).

| Argument | Type | Required | Default | Description |
|---|---|---|---|---|
| [`--title`](title.md) | String | Yes | — | Human-readable feature title |
| [`--slug`](slug.md) | String | No | Auto-generated from title | Feature slug (directory name) |
| [`--parent`](parent.md) | String | No | — | Parent feature ID for nesting |
| [`--status`](status.md) | String | No | `Conceptual` | Initial feature status |
| [`--description`](description.md) | String | No | — | Short description for the Summary section |
| [`--depends-on`](depends-on.md) | String | No | — | Comma-separated feature IDs |
| [`--commit`](commit.md) | Boolean | No | `false` | Create a git commit |
| [`--push`](push.md) | Boolean | No | `false` | Commit and push atomically |

Global arguments ([`--project`](../../../_args/project.md)) also apply. See the [global args](../../../_args/README.md) for details.

## Outstanding Questions

None at this time.
