# --context

Controls the number of surrounding lines displayed around each match.

| Detail | Value |
|---|---|
| Type | Integer |
| Required | No |
| Default | `2` |

## Syntax

```bash
synchestra spec search <query> --context <n>
```

## Behavior

For each matching line, `--context <n>` includes `n` lines before and `n` lines after the match in the output. Context lines are prefixed with their line number. The matching line itself is marked with `>` in text output.

Setting `--context 0` shows only the matching line with no surrounding context.

Context is bounded by file start/end — if a match is on line 2, only 1 line of leading context is shown regardless of the `--context` value.

## Examples

```bash
# Default: ±2 lines of context
synchestra spec search "claiming"

# No context — just matching lines
synchestra spec search "claiming" --context 0

# Wider context for understanding surrounding logic
synchestra spec search "conflict resolution" --context 5
```

## Outstanding Questions

None at this time.
