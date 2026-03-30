# SpecScore Decoupling Design

## Summary

Extract spec-format-aware Go code from the synchestra CLI into the specscore
repository as reusable library packages with a standalone CLI binary. Synchestra
then imports specscore packages and builds orchestration on top.

## Goals

1. **Standalone specscore tooling** — anyone can `specscore lint`, `specscore feature list` without synchestra
2. **Clean separation** — spec-format logic (features, linting, source refs) lives in specscore; orchestration (tasks, state, agents) stays in synchestra
3. **Reusable Go packages** — `pkg/` is the public API, importable by synchestra and any other consumer
4. **CI/CD for specscore** — build, test, lint, and release workflows from day one

## Decisions

- **Go module path:** `github.com/synchestra-io/specscore`
- **Approach:** Both CLI binary and library packages (option C)
- **Migration strategy:** Incremental with `go.mod replace` directive during development (option B)
- **Package extraction order:** exitcode → sourceref → feature → lint → projectdef → CLI wiring

## What Moves to SpecScore

| Code area | Current location (synchestra) | New location (specscore) |
|---|---|---|
| Exit codes | `pkg/cli/exitcode/` | `pkg/exitcode/` |
| Source references | `pkg/sourceref/` | `pkg/sourceref/` |
| Feature discovery, traversal, metadata | `pkg/cli/feature/` (12 files) | `pkg/feature/` |
| Spec linting engine + all rules | `pkg/cli/spec/` (10 files) | `pkg/lint/` |
| Project definition YAML schema | `pkg/cli/project/` (partial) | `pkg/projectdef/` |
| Code-to-spec tracing | `pkg/cli/code/` | CLI command moves to `internal/cli/code.go`; logic lives in `pkg/sourceref/` |

## What Stays in Synchestra

- `pkg/cli/task/` — task lifecycle (claim, complete, etc.)
- `pkg/cli/state/`, `pkg/state/`, `pkg/state/gitstore/` — state management
- `pkg/cli/project/` — project setup commands (new, init, info) that compose specscore types with state/ingitdb
- `pkg/cli/gitops/` — git commit/push operations
- `pkg/cli/globalconfig/` — `~/.synchestra.yaml`
- `pkg/cli/reporef/`, `pkg/cli/resolve/` — repository resolution via ingitdb
- `pkg/cli/test/` — testing framework wrapper

After migration, synchestra's `pkg/cli/feature/`, `pkg/cli/spec/`, and `pkg/cli/code/` become thin cobra wrappers that import specscore packages.

## SpecScore Repository Structure

```
specscore/
├── spec/                          # (existing) SpecScore format specifications
├── docs/                          # (existing)
├── pkg/
│   ├── exitcode/                  # Shared exit code constants & Error type
│   ├── feature/                   # Feature discovery, traversal, metadata
│   │   ├── discover.go
│   │   ├── info.go
│   │   ├── deps.go
│   │   ├── refs.go
│   │   ├── transitive.go
│   │   ├── fields.go
│   │   ├── slug.go
│   │   └── template.go
│   ├── lint/                      # Spec linting engine + all rules
│   │   ├── linter.go
│   │   ├── checkers_extended.go
│   │   ├── readme_exists.go
│   │   ├── oq_section.go
│   │   ├── index_entries.go
│   │   ├── heading_levels.go
│   │   ├── feature_ref_syntax.go
│   │   ├── internal_links.go
│   │   ├── forward_refs.go
│   │   ├── code_annotations.go
│   │   ├── plan_hierarchy.go
│   │   └── plan_roi.go
│   ├── sourceref/                 # specscore: annotation parsing & scanning
│   │   ├── sourceref.go
│   │   └── scan.go
│   └── projectdef/                # specscore-project.yaml schema & parsing
│       └── projectdef.go
├── cmd/
│   └── specscore/
│       └── main.go                # CLI entry point
├── internal/
│   └── cli/                       # CLI command wiring (thin wrappers around pkg/)
│       ├── root.go
│       ├── feature.go
│       ├── spec.go
│       └── code.go
├── go.mod
├── go.sum
├── .goreleaser.yml
├── .github/
│   └── workflows/
│       ├── go-ci.yml
│       └── release.yml
├── AGENTS.md
├── CLAUDE.md
├── README.md
└── LICENSE
```

## Package API Design

### `pkg/exitcode`

Moves as-is from synchestra. Same constants (0, 1, 2, 3, 4, 10), same `Error` type.
Only change: module path becomes `github.com/synchestra-io/specscore/pkg/exitcode`.

### `pkg/feature`

Pure library — no cobra dependency. Current synchestra code has business logic mixed
with cobra command wiring; the extraction separates them.

```go
package feature

type Feature struct { ID, Status string; Deps, Children []string; Sections []Section }
type TreeNode struct { ID string; Children []*TreeNode }
type NewOptions struct { Title string }

func Discover(root string) ([]Feature, error)
func Info(root, featureID string) (*Feature, error)
func List(root string) ([]string, error)
func Tree(root string) (*TreeNode, error)
func Deps(root, featureID string) ([]string, error)
func Refs(root, featureID string) ([]string, error)
func TransitiveDeps(root, featureID string) ([]string, error)
func TransitiveRefs(root, featureID string) ([]string, error)
func New(root, featureID string, opts NewOptions) error
func ValidateSlug(slug string) error
func Fields(root, featureID string, fields []string) (map[string]string, error)
```

No `fmt.Print` — callers decide how to render output.

### `pkg/lint`

```go
package lint

type Violation struct {
    File     string
    Line     int
    Severity string
    Rule     string
    Message  string
}

type Options struct {
    SpecRoot string
    Rules    []string
    Ignore   []string
    Severity string
}

func Lint(opts Options) ([]Violation, error)
func FilterBySeverity(violations []Violation, min string) []Violation
```

Individual rules are registered internally. Both the specscore CLI and synchestra
call `lint.Lint()`.

### `pkg/sourceref`

Moves nearly as-is. Exports: `Reference`, `DetectReference`, `ExtractReference`,
`ParseReference`, `ScanLine`.

### `pkg/projectdef`

Extracted from synchestra's `pkg/cli/project/`:

```go
package projectdef

type SpecConfig struct { Title, StateRepo string; Repos []string }
type StateConfig struct { SpecRepos []string }
type CodeConfig struct { SpecRepos []string }

func ReadSpecConfig(dir string) (SpecConfig, error)
func WriteSpecConfig(dir string, cfg SpecConfig) error
func ReadStateConfig(dir string) (StateConfig, error)
func WriteStateConfig(dir string, cfg StateConfig) error
func ReadCodeConfig(dir string) (CodeConfig, error)
func WriteCodeConfig(dir string, cfg CodeConfig) error
```

## CI/CD for SpecScore

### go-ci.yml

Uses shared `strongo/go-ci-action` workflow, same as synchestra. Triggers on `.go`,
`go.mod`, `go.sum`, and workflow file changes.

### release.yml

Tag-triggered goreleaser release. Snapshot on push to `main`. Builds `specscore`
binary for linux/darwin/windows (amd64, arm64, excluding windows/arm64).

### .goreleaser.yml

Same pattern as synchestra minus `ai-plugin.zip`. Version injection via ldflags into
`internal/cli.version`, `internal/cli.commit`, `internal/cli.date`.

## Migration Strategy

### Phase 1: Bootstrap specscore Go module

- Initialize `go.mod`
- Create `cmd/specscore/main.go` and `internal/cli/root.go` — minimal skeleton
- Add `pkg/exitcode/` — copy from synchestra, update module path
- Add `.goreleaser.yml`, `.github/workflows/go-ci.yml`, `.github/workflows/release.yml`
- Add `AGENTS.md`, `CLAUDE.md`

### Phase 2: Extract packages incrementally

Order follows internal dependency chain:

1. `pkg/exitcode` — no deps
2. `pkg/sourceref` — depends only on stdlib
3. `pkg/feature` — depends on exitcode
4. `pkg/lint` — depends on exitcode, feature
5. `pkg/projectdef` — depends on exitcode
6. `internal/cli/` — wire cobra commands

For each package: copy to specscore, refactor out cobra coupling, verify builds/tests,
then update synchestra to import from specscore via `replace` directive.

### Phase 3: Finalize

- Remove `replace` directive from synchestra's `go.mod`
- Tag specscore `v0.1.0`
- Update synchestra to depend on tagged version
- Verify both repos build independently from clean checkout

### go.mod replace during development

In synchestra's `go.mod`:
```
require github.com/synchestra-io/specscore v0.0.0

replace github.com/synchestra-io/specscore => ../specscore
```

## Multi-Agent Execution Plan

Plan will be stored at `spec/plans/specscore-decoupling/` in the synchestra repo.

### Task breakdown with agent assignment and model recommendations

| Task | Repo | Model | Depends on | Description |
|---|---|---|---|---|
| T1 | specscore | Sonnet | — | Bootstrap Go module (go.mod, main.go skeleton, AGENTS.md, CLAUDE.md) |
| T2 | specscore | Sonnet | T1 | Add CI/CD workflows (go-ci.yml, release.yml, .goreleaser.yml) |
| R1 | specscore | Sonnet | T2 | Review bootstrap + CI/CD |
| T3 | specscore | Haiku | R1 | Extract `pkg/exitcode` |
| T4 | specscore | Sonnet | R1 | Extract `pkg/sourceref` |
| T5 | specscore | Opus | T3 | Extract `pkg/feature` (decouple from cobra, design clean API) |
| T6 | specscore | Opus | T3, T5 | Extract `pkg/lint` (linter + all rules) |
| T7 | specscore | Sonnet | T3 | Extract `pkg/projectdef` |
| R2 | specscore | Opus | T4–T7 | Review all extracted packages — APIs clean, no cobra leakage, tests pass |
| T8 | specscore | Sonnet | R2 | Wire `internal/cli/` commands + `cmd/specscore/main.go` |
| R3 | specscore | Sonnet | T8 | Review CLI wiring, verify specscore builds and commands work |
| T9 | synchestra | Opus | R3 | Update synchestra to import specscore (replace directive, rewrite thin wrappers, delete extracted logic) |
| T10 | both | Sonnet | T9 | Finalize — remove replace, tag v0.1.0, verify clean builds |
| R4 | both | Opus | T10 | Final review — both repos build independently, wrappers are thin, behavior preserved |

### Parallelization opportunities

- T3 and T4 can run in parallel (both depend only on R1)
- T5 and T7 can run in parallel (both depend on T3)
- T6 depends on T5 completing first

### Review scope

- **R1:** Skeleton compiles, workflows match synchestra's pattern, AGENTS.md/CLAUDE.md correct
- **R2:** APIs are clean, no cobra in `pkg/`, tests pass, no synchestra-specific logic in packages
- **R3:** All specscore CLI commands work (`specscore lint`, `specscore feature list`, etc.)
- **R4:** Synchestra builds and passes tests, wrappers are thin, replace directive removed, both repos independently buildable

### Verification commands (required for every task)

```bash
gofmt -w .
golangci-lint run ./...
go test ./...
go build ./...
go vet ./...
```

## Outstanding Questions

- None at this time.
