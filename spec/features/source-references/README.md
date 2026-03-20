# Feature: Source References

**Status:** Conceptual

## Summary

Source references are inline annotations in any source file that link code to Synchestra resources (features, plans, documents, tasks). A single prefix — `synchestra:` — lets any tool, linter, or pre-commit hook discover references by binary search, resolve them against the project's Synchestra instance, and transform them into clickable URLs pointing to `synchestra.io`.

## Problem

Code that implements a feature often has no machine-readable link back to the specification that defines it. Developers add ad-hoc comments like `// see spec X` or `// Features implemented: cli/task/claim`, but these are convention-dependent, hard to validate, and invisible to the Synchestra platform.

Two concrete gaps exist:

1. **Discoverability** — `synchestra feature refs` currently scans only `## Dependencies` sections in spec files. It cannot tell you which *source files* implement or depend on a feature. Developers lose the spec ↔ code traceability that makes specifications useful.
2. **Platform stickiness** — embedding links to `synchestra.io` in user code creates a natural touchpoint. Every developer who clicks a reference lands on the Synchestra feature page, reinforcing the platform as the source of truth.

## Design Philosophy

- **Language-agnostic** — the notation must work in any language's comment syntax. Detection is a byte-level prefix search (`synchestra:` or `https://synchestra.io/`), not an AST operation.
- **Strict validation** — following Go's philosophy, references that point to non-existent resources are errors, not warnings. Invalid references are caught by linter, pre-commit hook, or PR check.
- **Single prefix** — `synchestra:` covers all resource types (features, plans, docs, tasks). One prefix to search, one parser to maintain, one convention to learn.
- **Graceful cross-repo** — same-repo references omit org/repo for brevity. Cross-repo references append `@{org}/{repo}`. Org/repo for the current context is inferred from git remote and can be overridden in project config.

## Behavior

### Notation format

```
synchestra:{type}/{path}
synchestra:{type}/{path}@{org}/{repo}
```

- **`{type}`** — resource type (see [Resource types](#resource-types))
- **`{path}`** — resource identifier, using `/` as separator for hierarchical paths
- **`@{org}/{repo}`** — optional; omitted when referencing resources in the same project

### Resource types

The initial set of resource types is fixed. User-configurable types may be added later via project configuration.

| Type | Resolves to | Example |
|---|---|---|
| `feature` | `spec/features/{path}/README.md` | `synchestra:feature/cli/task/claim` |
| `plan` | `spec/plans/{path}/README.md` | `synchestra:plan/v2-migration` |
| `doc` | `docs/{path}` | `synchestra:doc/api/rest` |
| `task` | Task in the state repository | `synchestra:task/plan-slug/task-slug` |

### Examples

What developers type (authoring):

```go
// synchestra:feature/cli/task/claim
// synchestra:feature/agent-skills@acme/orchestrator
```

What gets committed after lint/pre-commit expansion:

```go
// https://synchestra.io/synchestra-io/synchestra/feature/cli/task/claim
// https://synchestra.io/acme/orchestrator/feature/agent-skills
```

```python
# https://synchestra.io/synchestra-io/synchestra/feature/model-selection
# https://synchestra.io/synchestra-io/synchestra/task/chat-feature/implement-fast-path
```

```yaml
# https://synchestra.io/synchestra-io/synchestra/feature/project-definition
```

### URL mapping

Every short reference expands to a canonical URL on `synchestra.io`:

```
synchestra:{type}/{path}
  → https://synchestra.io/{org}/{repo}/{type}/{path}

synchestra:{type}/{path}@{org}/{repo}
  → https://synchestra.io/{org}/{repo}/{type}/{path}
```

For same-repo references, `{org}/{repo}` is resolved at expansion time from git remote or project configuration.

**Examples:**

| Short reference | Expanded URL |
|---|---|
| `synchestra:feature/cli/task/claim` | `https://synchestra.io/synchestra-io/synchestra/feature/cli/task/claim` |
| `synchestra:feature/agent-skills@acme/orchestrator` | `https://synchestra.io/acme/orchestrator/feature/agent-skills` |
| `synchestra:plan/v2-migration` | `https://synchestra.io/synchestra-io/synchestra/plan/v2-migration` |

### Canonical form and auto-expansion

The **expanded URL** is the canonical form stored in source files. The short `synchestra:` notation is an **authoring convenience** — developers type the short form, and the linter (or pre-commit hook) auto-expands it to the full URL before commit.

**Rationale:** every `https://synchestra.io/...` URL in a codebase is a clickable entry point. Developers can open the feature specification in the Synchestra web app with one click — in any IDE, GitHub diff view, or `grep` output. No tooling is required to resolve the reference. This also serves as a platform discovery mechanism: new contributors encountering these URLs are directed to synchestra.io, improving adoption and engagement.

**Authoring workflow:**

1. Developer writes `synchestra:feature/cli/task/claim` in a comment
2. Pre-commit hook (or `synchestra lint refs --fix`) expands it to `https://synchestra.io/synchestra-io/synchestra/feature/cli/task/claim`
3. The expanded URL is what gets committed and stored in the repository

The short form is never persisted in committed source — it exists only between authoring and the next lint/commit cycle.

### Detection strategy

Tools detect references using two byte-level prefix searches — no language-specific parsing required:

1. **Short notation** — scan for `synchestra:` prefix, then parse `{type}/{path}[@{org}/{repo}]`
2. **Expanded URLs** — scan for `https://synchestra.io/` prefix, then extract `{org}/{repo}/{type}/{path}` from the URL path

Both forms are recognized by all tools. The linter auto-expands short notation to URLs, so committed code should only contain expanded URLs. The short form is accepted as input for authoring convenience and backward compatibility.

### Org/repo resolution

When a reference omits `@{org}/{repo}`, the current project's org and repo must be inferred:

1. **Git remote** — parse `origin` remote URL to extract `{org}/{repo}`. This is the default.
2. **Project config override** — `synchestra-spec-repo.yaml` may declare an explicit `org` and `repo` that overrides git remote inference. This handles forks, mirrors, and non-standard remote names.

```yaml
# synchestra-spec-repo.yaml
project:
  org: synchestra-io
  repo: synchestra
```

### Validation

References are validated strictly — a reference to a non-existent resource is an error.

**Validation rules:**

| Check | Error condition |
|---|---|
| Resource type is known | Type is not in the fixed set (`feature`, `plan`, `doc`, `task`) |
| Resource exists | The resolved path does not exist in the target repository |
| Org/repo is resolvable | Same-repo reference but org/repo cannot be inferred (no git remote, no config override) |
| Cross-repo is reachable | `@{org}/{repo}` points to a repository that is not accessible (optional — may be deferred to CI) |

**Enforcement points:**

- **Linter** (`synchestra lint refs`) — scans source files, validates all references, reports errors with file:line locations
- **Pre-commit hook** — runs the linter on staged files before commit
- **PR check** — GitHub Actions workflow that runs the linter on changed files

### Relationship to existing Go feature annotations

The current convention in this repository uses structured comments:

```go
// Features implemented: cli/task/claim, cli/task/update
// Features depended on:  state-sync/pull, project-definition/state-repo
```

Source references (`synchestra:feature/...`) supersede this convention. The new format is:

- **Language-agnostic** — works in Go, Python, TypeScript, YAML, or any file with comments
- **Linkable** — transforms to clickable URLs
- **Validatable** — a single linter validates all resource types
- **Richer** — references plans, docs, and tasks in addition to features

Migration path: existing `// Features implemented:` annotations are replaced with `synchestra:feature/` references. A codemod or linter auto-fix can perform this transformation.

### Integration with `synchestra feature refs`

`synchestra feature refs` currently scans only `## Dependencies` sections in spec files. With source references, it gains a second data source:

1. **Spec references** — features whose `## Dependencies` section lists the target (existing behavior)
2. **Source references** — source files containing `synchestra:feature/{target}` annotations (new behavior)

The command output distinguishes the two:

```bash
synchestra feature refs cli/task/claim
```

```
# Spec references (Dependencies sections)
conflict-resolution
cross-repo-sync

# Source references
pkg/cli/task/claim.go:12
pkg/cli/task/update.go:8
```

With `--fields=type`:

```yaml
- path: conflict-resolution
  type: spec
- path: cross-repo-sync
  type: spec
- path: pkg/cli/task/claim.go:12
  type: source
- path: pkg/cli/task/update.go:8
  type: source
```

## Dependencies

- [feature](../feature/README.md)
- [cli](../cli/README.md)
- [project-definition](../project-definition/README.md)

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Feature](../feature/README.md) | Source references point to features; `feature refs` consumes them |
| [CLI](../cli/README.md) | `synchestra lint refs` validates references; `feature refs` scans for them |
| [Project Definition](../project-definition/README.md) | `synchestra-spec-repo.yaml` provides org/repo override for resolution |
| [LSP](../lsp/README.md) | LSP server can provide go-to-definition for `synchestra:` references in IDEs |
| [Development Plan](../development-plan/README.md) | Plans are a referenceable resource type |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Should `synchestra:` references in non-comment contexts (e.g., string literals, documentation) be detected, or only in comments? Limiting to comments requires language-specific parsing, which conflicts with the language-agnostic detection goal.
- How should `synchestra:task/...` references be validated, given that tasks live in a separate state repository that may not be locally available?
- Should there be a `synchestra refs` top-level command (scanning all resource types) in addition to `synchestra feature refs` (feature-only)?
