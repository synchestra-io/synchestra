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

- **Language-agnostic** — the notation must work in any language's comment syntax. Detection requires a recognized comment prefix on the same line — no AST parsing, just a single-line regex match.
- **Strict validation** — following Go's philosophy, references that point to non-existent resources are errors, not warnings. Invalid references are caught by linter, pre-commit hook, or PR check.
- **Single prefix** — `synchestra:` covers all resource types (features, plans, docs, tasks). One prefix to search, one parser to maintain, one convention to learn.
- **Graceful cross-repo** — same-repo references omit host/org/repo for brevity. Cross-repo references append `@{host}/{org}/{repo}`. Host, org, and repo for the current context are inferred from git remote and can be overridden in project config.

## Behavior

### Notation format

```
synchestra:{reference}
synchestra:{reference}@{host}/{org}/{repo}
```

- **`{reference}`** — either a type-prefixed shortcut or a repo-root-relative path (see [Resolution](#short-notation-resolution))
- **`@{host}/{org}/{repo}`** — optional; omitted when referencing resources in the same project. `{host}` is the repository host (e.g., `github.com`, `bitbucket.org`, `gitlab.mycompany.com`)

### Resource type shortcuts

Known type prefixes provide shorthand for common paths. User-configurable types may be added later via project configuration.

| Type prefix | Expands to repo path | Example shortcut | Resolved path |
|---|---|---|---|
| `feature/` | `spec/features/{path}` | `feature/cli/task/claim` | `spec/features/cli/task/claim` |
| `plan/` | `spec/plans/{path}` | `plan/v2-migration` | `spec/plans/v2-migration` |
| `doc/` | `docs/{path}` | `doc/api/rest` | `docs/api/rest` |

### Short notation resolution

When resolving a `synchestra:` reference, the following order is used:

1. **Type prefix** — if the first segment matches a known type prefix (`feature`, `plan`, `doc`), expand it to the corresponding repo path
2. **Fallback to path** — if the first segment is not a known prefix, or if type-based resolution fails (path does not exist), treat the entire value as a repo-root-relative path

**Examples:**

| Short notation | Resolution | Resolved repo path |
|---|---|---|
| `synchestra:feature/cli/task/claim` | Type prefix `feature/` | `spec/features/cli/task/claim` |
| `synchestra:plan/v2-migration` | Type prefix `plan/` | `spec/plans/v2-migration` |
| `synchestra:doc/api/rest` | Type prefix `doc/` | `docs/api/rest` |
| `synchestra:spec/features/cli/task/claim` | Not a known prefix → path | `spec/features/cli/task/claim` |
| `synchestra:docs/api/rest` | Not a known prefix → path | `docs/api/rest` |
| `synchestra:README.md` | Not a known prefix → path | `README.md` |

### Examples

What developers type (authoring — using type shortcuts or full paths):

```go
// synchestra:feature/cli/task/claim
// synchestra:feature/agent-skills@github.com/acme/orchestrator
// synchestra:spec/features/cli/task/claim    (full path — equivalent to above)
```

What gets committed after lint/pre-commit expansion:

```go
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/task/claim
// https://synchestra.io/github.com/acme/orchestrator/spec/features/agent-skills
```

```python
# https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/model-selection
```

```yaml
# https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/project-definition
```

```sql
-- https://synchestra.io/bitbucket.org/acme/data-pipeline/spec/features/etl-config
```

### URL mapping

Every short reference expands to a canonical URL on `synchestra.io`. The URL uses the **resolved repo-root-relative path** — the `{type}` prefix is not present in the URL.

```
synchestra:{reference}
  → https://synchestra.io/{host}/{org}/{repo}/{resolved_path}

synchestra:{reference}@{host}/{org}/{repo}
  → https://synchestra.io/{host}/{org}/{repo}/{resolved_path}
```

For same-repo references, `{host}/{org}/{repo}` is resolved at expansion time from git remote or project configuration.

**Examples:**

| Short reference | Expanded URL |
|---|---|
| `synchestra:feature/cli/task/claim` | `https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/task/claim` |
| `synchestra:spec/features/cli/task/claim` | `https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/task/claim` |
| `synchestra:feature/agent-skills@github.com/acme/orchestrator` | `https://synchestra.io/github.com/acme/orchestrator/spec/features/agent-skills` |
| `synchestra:plan/v2-migration` | `https://synchestra.io/github.com/synchestra-io/synchestra/spec/plans/v2-migration` |
| `synchestra:doc/api/rest@bitbucket.org/acme/docs` | `https://synchestra.io/bitbucket.org/acme/docs/docs/api/rest` |
| `synchestra:README.md` | `https://synchestra.io/github.com/synchestra-io/synchestra/README.md` |

### Canonical form and auto-expansion

The **expanded URL** is the canonical form stored in source files. The short `synchestra:` notation is an **authoring convenience** — developers type the short form, and the linter (or pre-commit hook) auto-expands it to the full URL before commit.

**Rationale:** every `https://synchestra.io/...` URL in a codebase is a clickable entry point. Developers can open the feature specification in Synchestra Hub with one click — in any IDE, GitHub diff view, or `grep` output. No tooling is required to resolve the reference. This also serves as a platform discovery mechanism: new contributors encountering these URLs are directed to synchestra.io, improving adoption and engagement.

**Authoring workflow:**

1. Developer writes `synchestra:feature/cli/task/claim` in a comment
2. Pre-commit hook (or `synchestra lint refs --fix`) resolves the type prefix and expands it to `https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/task/claim`
3. The expanded URL is what gets committed and stored in the repository

The short form is never persisted in committed source — it exists only between authoring and the next lint/commit cycle.

### Detection strategy

A valid source reference must be preceded on the same line by a recognized comment prefix followed by optional whitespace. This eliminates false positives from string literals and non-comment code without requiring AST parsing.

**Detection regex (single line):**

```regex
^\s*(//|#|--|[/*]|%|;)\s*(synchestra:|https://synchestra\.io/)
```

**Recognized comment prefixes:**

| Prefix | Languages |
|---|---|
| `//` | Go, JS, TS, Java, C, C++, Rust, Swift, Kotlin |
| `#` | Python, Ruby, YAML, Shell, Perl, Elixir |
| `--` | SQL, Lua, Haskell |
| `*` or `/*` | Block comments in C-family languages |
| `%` | LaTeX, Erlang |
| `;` | Lisp, Clojure, INI files |

**Valid examples:**

```
// synchestra:feature/cli/task/claim          ✓ (Go, JS)
//synchestra:feature/cli/task/claim           ✓ (no space)
#  synchestra:feature/model-selection         ✓ (Python, YAML)
-- https://synchestra.io/github.com/org/repo/spec/features/x   ✓ (SQL)
; synchestra:plan/v2-migration                ✓ (Lisp)
```

**Invalid examples (not detected):**

```
synchestra:feature/cli/task/claim             ✗ (no comment prefix)
fmt.Println("synchestra:feature/x")          ✗ (inside string literal)
var x = "https://synchestra.io/github.com/org/repo/..." ✗ (inside string literal)
```

Users with uncommon comment syntax can open an issue to expand the prefix set, or override it in project configuration (future).

**Two reference forms are recognized:**

1. **Short notation** — `synchestra:` prefix, then `{reference}[@{host}/{org}/{repo}]`
2. **Expanded URLs** — `https://synchestra.io/` prefix, then `{host}/{org}/{repo}/{resolved_path}`

The linter auto-expands short notation to URLs, so committed code should only contain expanded URLs. The short form is accepted as input for authoring convenience.

### Host/org/repo resolution

When a reference omits `@{host}/{org}/{repo}`, the current project's host, org, and repo must be inferred:

1. **Git remote** — parse `origin` remote URL to extract `{host}`, `{org}`, and `{repo}`. This is the default. For example, `git@github.com:synchestra-io/synchestra.git` yields `github.com/synchestra-io/synchestra`.
2. **Project config override** — `synchestra-spec-repo.yaml` may declare explicit values that override git remote inference. This handles forks, mirrors, and non-standard remote names.

```yaml
# synchestra-spec-repo.yaml
project:
  host: github.com
  org: synchestra-io
  repo: synchestra
```

### Validation

References are validated strictly — a reference to a non-existent resource is an error.

**Validation rules:**

| Check | Error condition |
|---|---|
| Reference resolves | The resolved repo path does not exist in the target repository (after trying type prefix expansion and path fallback) |
| Host/org/repo is resolvable | Same-repo reference but host/org/repo cannot be inferred (no git remote, no config override) |
| Cross-repo is reachable | `@{host}/{org}/{repo}` points to a repository that is not accessible (optional — may be deferred to CI) |

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
| [CLI](../cli/README.md) | `synchestra lint refs` validates references; `feature refs` and [`code deps`](../cli/code/deps/README.md) scan for them |
| [CLI / code deps](../cli/code/deps/README.md) | Primary consumer — scans source files and lists referenced resources (code → spec direction) |
| [CLI / feature deps](../cli/feature/deps/README.md) | Complementary — shows spec → spec dependencies; `code deps` shows code → spec dependencies |
| [Project Definition](../project-definition/README.md) | `synchestra-spec-repo.yaml` provides org/repo override for resolution |
| [LSP](../lsp/README.md) | LSP server can provide go-to-definition for `synchestra:` references in IDEs |
| [Development Plan](../development-plan/README.md) | Plans are a referenceable resource type |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

None at this time.
