# CLI Arguments: `runner dispatch`

Arguments accepted by dispatch creation and observation operations.

| Argument | Supported by | Description |
|---|---|---|
| [`<target>`](target.md) | create | Auto-resolved Plan or Task target |
| [`--prompt`](prompt.md) | create | Ad-hoc instruction |
| [`--plan`](plan.md) | create | Plan-only target |
| [`--task`](task.md) | create | Task-only target |
| [`--runner`](runner.md) | create | Exact runner constraint |
| [`--profile`](profile.md) | create | Provider-neutral execution profile |
| [`--agent`](agent.md) | create | Agent adapter selector |
| [`--model`](model.md) | create | Exact or adapter-specific model selector |
| [`--effort`](effort.md) | create | Adapter-specific effort selector |
| [`<dispatch-id>`](dispatch-id.md) | status, logs, retry, cancel | Durable dispatch identifier |
| [`--cursor`](cursor.md) | logs | Log cursor |
| [`--reason`](reason.md) | retry, cancel | Audit reason |

All operations also accept [`--format`](../../../_args/format.md).

## Outstanding Questions

None at this time.
