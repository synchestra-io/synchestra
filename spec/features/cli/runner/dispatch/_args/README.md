# CLI Arguments: `runner dispatch`

**Parent:** [runner dispatch](../README.md)

Arguments specific to `synchestra runner dispatch`.

## Arguments

| Argument | Type | Required | Description |
|---|---|---|---|
| [`<target>`](target.md) | String (positional) | Conditional | SpecScore Plan or Task target |
| [`--prompt`](prompt.md) | String | Conditional | Ad-hoc work description |
| [`--runner`](runner.md) | String | No | Preferred registered worker |
| [`--profile`](profile.md) | Enum | No | Provider-neutral execution profile |
| [`--agent`](agent.md) | String | No | Agent adapter override |
| [`--model`](model.md) | String | No | Exact or adapter-specific model selector |

Also accepts the global [`--format`](../../../_args/format.md) argument.

Exactly one of `<target>` or `--prompt` is required.

## Outstanding Questions

None at this time.
