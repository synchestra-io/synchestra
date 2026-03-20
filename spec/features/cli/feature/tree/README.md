# Command: `synchestra feature tree`

**Parent:** [feature](../README.md)
**Skill:** [synchestra-feature-tree](../../../../../ai-plugin/skills/synchestra-feature-tree/README.md)

## Synopsis

```
synchestra feature tree [<feature_id>] [--direction up|down] [--project <project_id>] [--fields <fields>]
```

## Description

Displays the feature hierarchy as an indented tree.

- Without `<feature_id>`: shows the full project tree.
- With `<feature_id>`: shows the feature in context — ancestors (path to root) plus its subtree by default.
- `--direction` narrows the view to ancestors only (`up`) or subtree only (`down`). Only valid when `<feature_id>` is provided.

This replaces the need for separate `ancestors` / `successors` commands.

**`tree` vs `list`:** `tree` shows the structural hierarchy with indentation — use it for navigation, understanding nesting, and focusing on a feature's context. For a flat, grep/pipe-friendly list with full feature IDs, use [`feature list`](../list/README.md) instead.

This is a read-only command. It pulls the latest state from the spec repository but does not mutate anything.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| `<feature_id>` | No | Feature ID to focus on (positional argument, e.g., `cli/task`). When omitted, the full project tree is shown |
| `--direction` | No | `up` (ancestors only), `down` (subtree only). Default when `<feature_id>` is given: both (ancestors + subtree). Invalid without `<feature_id>` |
| [`--project`](../../_args/project.md) | No | Project identifier (e.g., `synchestra`). Autodetected from current directory if omitted |
| [`--fields`](../_args/fields.md) | No | Inline selected metadata next to each feature. Auto-switches output to YAML |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments (e.g., `--direction` without `<feature_id>`) |
| `3` | Feature or project not found |

## Behaviour

### Full project tree (no `<feature_id>`)

1. Pull latest state from the spec repository
2. Walk the project's features directory recursively
3. Each directory containing a `README.md` is a feature
4. Output the feature hierarchy with tab indentation for each nesting level, sorted alphabetically at each level

### Feature-focused tree (with `<feature_id>`)

1. Pull latest state from the spec repository
2. Locate the feature at `{features_dir}/{feature_id}/README.md`
3. Determine the direction:
   - **both** (default): collect ancestors (parent chain to root) and the feature's subtree
   - **up**: collect only the ancestor chain from the feature to the root
   - **down**: collect only the feature and its descendants
4. Output the resulting tree with tab indentation; the target feature is marked with `*` in text output
5. Sorting: alphabetical at each level; the target feature's branch is not reordered

### Format behaviour

- Without `--fields`: compact text with tab indentation (human-friendly default).
- With `--fields`: auto-switches to YAML. Each feature becomes a YAML node with the requested fields. Nesting uses `children` keys.
- `--format` overrides in either direction.

## Output

### Full project tree

```bash
synchestra feature tree
```

```
agent-skills
claim-and-push
cli
	feature
		deps
		info
		list
		refs
		tree
	task
		claim
		create
		list
		update
conflict-resolution
cross-repo-sync
```

Each nesting level adds one tab character. Only the leaf name is shown for nested features — the full path is implied by the indentation context.

### Feature in context (default: both directions)

```bash
synchestra feature tree cli/task
```

```
cli
	* task
		claim
		create
		list
		update
```

The `*` marker indicates the target feature. Ancestors are shown above, descendants below.

### Ancestors only

```bash
synchestra feature tree cli/task --direction=up
```

```
cli
	* task
```

### Subtree only

```bash
synchestra feature tree cli/task --direction=down
```

```
* task
	claim
	create
	list
	update
```

### With fields (auto-switches to YAML)

```bash
synchestra feature tree cli/task --fields=status,oq
```

```yaml
- path: cli
  status: "In Progress"
  oq: 3
  children:
    - path: cli/task
      focus: true
      status: "Conceptual"
      oq: 2
      children:
        - path: cli/task/claim
          status: "Conceptual"
          oq: 0
        - path: cli/task/create
          status: "Conceptual"
          oq: 1
```

The `focus: true` field marks the target feature in YAML output (equivalent to `*` in text).

## Outstanding Questions

- Should there be a `--depth` flag to limit tree depth?
