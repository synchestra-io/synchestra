# CLI Arguments: `spec` commands

**Parent:** [_args](../../_args/README.md)

Arguments specific to `synchestra spec` command group.

## Arguments

| Argument | Type | Required | Scope | Description |
|---|---|---|---|---|
| [`--rules`](rules.md) | String (CSV) | No | `spec lint` | Enable only specified rules (comma-separated: `readme-exists,oq-section`) |
| [`--ignore`](ignore.md) | String (CSV) | No | `spec lint` | Disable specified rules (comma-separated: `forward-refs,code-annotations`) |
| [`--severity`](severity.md) | String | No | `spec lint` | Report violations at this level or higher (`error`, `warning`, `info`; default: `error`) |
| `--format` | String | No | `spec lint`, `spec search` | Output format: `text` (default), `json`, `yaml` |
| `PATH` (positional) | String | No | `spec lint`, `spec search` | Spec root directory to scan/search (default: `./spec`) |

## Outstanding Questions

None at this time.
