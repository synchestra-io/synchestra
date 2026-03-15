# Review report: `config-opus`

**Reviewed worktree:** `.claude/worktrees/config-opus`

**Reviewed branch:** `worktree-config-opus` against `main`

**Scope reviewed:** branch delta introduced by commits `541a75e` through `5c26bc8`

## Summary

The branch adds the new `synchestra project new` command, repo/config helpers, integration tests, and feature-reference comments. The previously reported relative-`repos_dir` bug is fixed in this version, but I still found two meaningful correctness issues in the new project-creation flow. I also ran `go test ./...` in the worktree; the current tests pass, but they do not cover the problems below.

## Findings

### High — `project new` allows the state repo to overlap with spec or target repos

**Why it matters:** Synchestra's repository model requires the state repository to stay dedicated and separate because it is the high-churn coordination database for agents. The new command accepts any combination of repo refs and never rejects a state repo that resolves to the same repository as the spec repo or one of the targets, so it can create an invalid project layout and mix machine-written state into repositories that are supposed to have different lifecycles.

**Evidence:**

- `cli/project/new.go:50-66` parses all repo references but never checks that `--state-repo` differs from `--spec-repo` and every `--target-repo`.
- `cli/project/new.go:131-166` blindly writes config files and commits each resolved path, so colliding refs will cause the same repository to be treated as multiple repository types.
- `spec/architecture/repository-types.md:62-68` and `spec/architecture/repository-types.md:150-168` say the state repo must be dedicated and remains separate even when spec and code are combined.

### High — Existing local checkouts are trusted without verifying their remote origin

**Why it matters:** The command resolves each repo ref to a deterministic path under `repos_dir`. If that path already contains a git repository cloned from the wrong remote, the implementation still treats it as the requested repo and writes project config into it. That can silently attach an unrelated local repository to the new project and push commits to the wrong origin.

**Evidence:**

- `cli/project/new.go:79-107` resolves disk paths and only checks `gitops.IsGitRepo`; it never verifies that the checkout's `origin` matches the requested repo reference.
- `cli/gitops/gitops.go:18-25` already includes a `GetOriginURL` helper, but `project new` does not use it.
- `spec/features/cli/project/README.md:13-20` and `spec/features/cli/project/new/README.md:46-49` describe these arguments as repository references that are resolved and validated, not just arbitrary existing git directories.

## Validation

- Inspected the branch diff against `main`.
- Ran `go test ./...` in `.claude/worktrees/config-opus` successfully.
