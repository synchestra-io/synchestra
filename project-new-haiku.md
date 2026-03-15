# Review report: `config-haiku`

**Reviewed worktree:** `.claude/worktrees/config-haiku`

**Reviewed branch:** `worktree-config-haiku` against `main`

**Scope reviewed:** branch delta introduced by commits `8eaa2b1` and `3512602`

## Summary

The branch adds `synchestra project new`, supporting config helpers, and Go-file feature-reference comments. I found two meaningful issues in the new implementation. I also ran `go test ./...` in the worktree; the current tests pass, but they do not cover the problems below.

## Findings

### High — Relative `repos_dir` paths do not follow the spec

**Why it matters:** the global-config spec says relative `repos_dir` values are resolved relative to the user's home directory. The new loader returns relative paths unchanged, so a config like `repos_dir: repos` resolves relative to whatever directory the CLI happens to run from instead of the home directory. That can clone or modify repositories in the wrong location.

**Evidence:**

- `internal/config.go:48-54` only expands `~`; non-absolute relative paths are returned unchanged.
- `internal/config_test.go:57-67` explicitly encodes the wrong behavior by expecting `"relative/path"` instead of a home-relative path.
- `spec/features/global-config/README.md:46` says: “Relative paths are resolved relative to the user's home directory.”

### High — `project new` is not atomic across the repos it mutates

**Why it matters:** the command writes and pushes the spec repo first, then the state repo, then targets one by one. If a later commit or push fails, earlier repositories are already permanently updated, leaving the project half-created and inconsistent. That breaks the CLI's documented mutation guarantees and creates a hard-to-recover operational state.

**Evidence:**

- `cli/project_new.go:219-233` performs separate commit/push operations sequentially for each repo.
- `spec/features/README.md:91` and `spec/features/cli/README.md:56` describe CLI mutations as atomic commit-and-push operations.

## Validation

- Inspected the branch diff against `main`.
- Ran `go test ./...` in `.claude/worktrees/config-haiku` successfully.
