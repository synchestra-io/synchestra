# Review report: `config-sonnet`

**Reviewed worktree:** `.claude/worktrees/config-sonnet`

**Reviewed branch:** `worktree-config-sonnet` against `main`

**Scope reviewed:** branch delta introduced by commits `d7ae8e3` through `52f050a`

## Summary

The branch adds `synchestra project new`, repo/config helpers, tests, and feature-reference comments. I found three meaningful correctness issues that still look unsafe to merge. I also ran `go test ./...` in the worktree; the current tests pass, but they do not cover the problems below.

## Findings

### High — `project new` creates a spec repo shape that the rest of the CLI will not recognize

**Why it matters:** the new command writes `synchestra-spec.yaml` to the spec repository, but the rest of Synchestra still documents and discovers projects via `synchestra-project.yaml`. A freshly created project can therefore fail autodetection for commands that walk upward looking for the canonical project file, which means `serve`, `mcp`, and implicit `--project` resolution will not work from the new spec repo layout.

**Evidence:**

- `cli/cmd/project/new.go:134-168` conflict-checks and writes `synchestra-spec.yaml` in the spec repo.
- `spec/features/project-definition/README.md:17-18` and `spec/architecture/repository-types.md:123-136` define `synchestra-project.yaml` as the project entry point and discovery anchor.
- `spec/features/cli/_args/project.md:17-21`, `spec/features/cli/serve/README.md:46-50`, and `spec/features/cli/mcp/README.md:36-38` say CLI autodetection looks for `synchestra-project.yaml`.

### High — Relative `repos_dir` values still resolve incorrectly

**Why it matters:** the global-config spec says relative `repos_dir` paths are resolved relative to the user's home directory. The new loader only expands `~` and otherwise returns the path unchanged, so a config like `repos_dir: repos` ends up resolving relative to whatever directory the CLI was launched from. That can clone or mutate repositories in the wrong location.

**Evidence:**

- `cli/internal/globalconfig/globalconfig.go:35-49` only performs tilde expansion; non-absolute relative paths are returned unchanged.
- `cli/internal/globalconfig/globalconfig_test.go:23-52` covers absolute and tilde paths, but there is no test for the required home-relative behavior.
- `spec/features/global-config/README.md:44-47` says: “Relative paths are resolved relative to the user's home directory.”

### High — `project new` still allows an invalid shared state/spec/target repository layout

**Why it matters:** Synchestra requires the state repository to remain dedicated and separate, even when spec and code are combined. The new command never checks whether `--state-repo` resolves to the same repository as `--spec-repo` or one of the `--target-repo` values, so it can create an unsupported topology and write multiple role-specific config files into the same checkout.

**Evidence:**

- `cli/cmd/project/new.go:80-120` parses and resolves all repo refs, but never validates that the resolved repositories are distinct.
- `cli/cmd/project/new.go:160-205` writes separate spec/state/target config files and commits each role independently, so overlapping refs will treat the same repo as multiple repository types.
- `spec/architecture/repository-types.md:62-67` says the state repo “must” be dedicated, and `spec/architecture/repository-types.md:150-168` says the state repo remains separate even when spec and code are combined.

## Validation

- Inspected the branch diff against `main`.
- Ran `go test ./...` in `.claude/worktrees/config-sonnet` successfully.
