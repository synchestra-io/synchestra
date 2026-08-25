# Arguments: `synchestra runner invoke`

Command-specific arguments for [`runner invoke`](../README.md).

| Argument | Type | Required | Default | Description |
|---|---|---|---|---|
| [`@<payload-file>`](payload-file.md) | File path | Yes | — | Opaque JSON payload file, denoted by a leading `@`. |
| [`--runner`](runner.md) | String | Yes | — | Exact registered target runner ID. |
| [`--handler`](handler.md) | Enum | Yes | — | Closed registered WB handler name. |
| [`--invocation-id`](invocation-id.md) | String | Yes | — | Caller-owned invocation identity. |
| [`--deadline`](deadline.md) | RFC3339 timestamp | No | — | Immutable optional invocation deadline. |

The global [`--format`](../../../_args/format.md) argument also applies.

## Open Questions

None at this time.
