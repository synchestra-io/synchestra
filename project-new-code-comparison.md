# `project new` implementation comparison

**Compared branches:** `worktree-config-opus`, `worktree-config-sonnet`, `worktree-config-haiku`

**Reference spec:** `spec/features/cli/project/new/README.md` plus related argument specs and global config behavior.

**Validation used:** branch diff review, targeted source inspection, and `go test ./... -cover` in each worktree.

## Executive summary

Winner: Opus

**Recommended branch to merge:** `worktree-config-opus`

**Model that did the best job:** `Opus`

**Best practical strategy:** do *not* try to merge all three branches together. They overlap heavily in the same command surface and will conflict. Merge `worktree-config-opus` as the base implementation, then port a few ideas from the others in a follow-up hardening pass.

## Ranking

| Rank | Branch / model | Score | Why it landed there |
| --- | --- | --- | --- |
| **1** | `worktree-config-opus` / Opus | **8/10** | Best overall balance of implementation completeness, documentation, package design, and test depth. It fixed the relative-`repos_dir` bug that still exists in the other two branches. |
| **2** | `worktree-config-sonnet` / Sonnet | **7/10** | Clean architecture and strong test coverage, but still misses an important global-config requirement and has a few correctness/completeness gaps around repo handling. |
| **3** | `worktree-config-haiku` / Haiku | **4/10** | Implements the happy path, but it has the weakest exit-code behavior, lighter tests, and multiple spec/completeness issues that make it the riskiest merge. |

## Scorecard

| Branch | Code quality | Test coverage | Completeness vs spec | Other review considerations |
| --- | --- | --- | --- | --- |
| `config-opus` | Strong | Best observed | Mostly complete | Medium risk |
| `config-sonnet` | Strong | Strong | Good but incomplete | Medium risk |
| `config-haiku` | Fair | Weakest | Material gaps | Highest risk |

## Branch-by-branch review

### 1. `config-opus` — best overall

- **What it does well:** best overall completeness, best supporting docs/readmes, strongest spec-facing polish, and the only branch that clearly fixes relative `repos_dir` resolution to be home-relative as required by `spec/features/global-config/README.md:46`.
- **Evidence:** `cli/globalconfig/globalconfig.go` resolves relative paths with `filepath.Join(homeDir, reposDir)`, and its tests cover that case in `cli/globalconfig/globalconfig_test.go`.
- **Main concerns:** it accepts overlapping repo roles because it never checks that `state-repo` differs from `spec-repo` and the targets; it also trusts an existing checkout at the resolved path without verifying that its `origin` matches the requested repo reference.
- **Spec gap:** required-flag handling still comes out wrong for some missing-flag cases because `MarkFlagRequired` fires before the command’s own exit-code logic. Observed behavior: built binary returned exit code `1` for missing `--target-repo`, while the spec says invalid arguments should be `2`.
- **Bottom line:** this is the branch I would merge, but I would pair it with a short follow-up fix for repo-role validation, origin verification, and required-flag exit codes.

### 2. `config-sonnet` — good architecture, not the best final result

- **What it does well:** very solid package structure, good dependency injection via `gitops.Runner`, strong coverage, and better runtime exit-code preservation than Haiku.
- **Main correctness gap:** it still resolves `repos_dir` incorrectly for relative paths. In `cli/internal/globalconfig/globalconfig.go:35-39`, it only expands `~`; a plain relative path is left unchanged, which conflicts with `spec/features/global-config/README.md:46`.
- **Other concern:** it mixes URL sources by using `specRef.OriginURL()` for the spec repo but `git.OriginURL(...)` for state/targets in `cli/cmd/project/new.go:122-157`. That inconsistency is avoidable and makes the implementation harder to reason about.
- **Completeness gap:** like Opus, it does not reject an invalid repository layout where the state repo overlaps with the spec or target repos.
- **Bottom line:** good engineering style and tests, but it is still behind Opus on spec compliance and overall readiness.

### 3. `config-haiku` — useful ideas, weakest merge candidate

- **What it does well:** some thoughtful hardening in repo-path handling, including path-traversal checks and a symlink check before cloning.
- **Main correctness gap:** it has the same relative-`repos_dir` bug as Sonnet. `internal/config.go:48-54` expands only `~` and leaves plain relative paths untouched, contrary to `spec/features/global-config/README.md:46`.
- **Main runtime issue:** custom exit codes are effectively lost. The command wraps `*internal.ExitError` into a plain error in `cli/project_new.go:27-31`, and `main.go` exits `1` for all command failures. In direct binary checks, both invalid repo input and missing `--target-repo` exited `1` instead of the spec’s `2`.
- **Completeness issue:** the spec repo conflict check is stricter than the state/target checks and rejects any existing spec config outright in `cli/project_new.go:145-151`, making reruns inconsistent.
- **Bottom line:** I would not merge this branch as the base implementation.

## Can we merge “the best of all three”?

**Not by merging all three branches directly.** They overlap too much in the same files and package layout. A direct multi-branch merge would be noisy and conflict-heavy.

**Recommended approach:** merge `worktree-config-opus` as the base, then selectively port these ideas:

- **From Sonnet:** keep its cleaner runner/testability patterns and its stronger exit-code plumbing approach for command-level errors.
- **From Haiku:** port its repo path hardening ideas, especially path traversal and symlink checks around clone targets.
- **Before or immediately after merge:** add explicit validation that the state repo is distinct from the spec and target repos, and verify that any pre-existing checkout at a resolved path actually points at the requested remote.

## Final recommendation

**Merge branch:** `worktree-config-opus`

**Best model:** `Opus`

**Confidence:** medium-high

**Why:** it is the strongest combination of code quality, test depth, completeness, and repository-convention fit. It is not perfect, but it is clearly the best base to build on.

**Suggested merge condition:** if possible, land one short hardening commit alongside or immediately after merge to address repo-role overlap, origin verification for pre-existing checkouts, and missing-flag exit code behavior.
