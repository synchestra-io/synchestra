# gitops

Package `gitops` provides git operations used by Synchestra CLI commands.

The central type is `Runner`, a struct whose fields are functions (one per git operation). Callers receive a real implementation via `NewRunner()` and can substitute fakes in tests without any interface indirection.

## Operations

| Function field | Description |
|---|---|
| `IsRepo(dir)` | Returns `true` if `dir` is inside a git repository. Returns `false, nil` for non-existent paths or non-repo directories. |
| `Clone(url, dir)` | Clones a remote URL into `dir`, creating parent directories as needed. |
| `OriginURL(dir)` | Returns the URL of the `origin` remote for the repo at `dir`. |
| `CommitAndPush(dir, files, msg)` | Stages the given files, commits with `msg`, and pushes to `origin HEAD`. |
| `Push(dir)` | Pushes the current branch to `origin HEAD` (used for retry paths). |
| `Pull(dir)` | Pulls the latest changes from the upstream remote. |

## Design notes

- `isRepo` treats any `*exec.ExitError` as "not a repo" — this covers both non-existent paths and directories that are not git repos, without returning an error to the caller.
- `CommitAndPush` and `Push` both use `--set-upstream origin HEAD` for reliability across git versions.
- `Push` is kept separate from `CommitAndPush` to support retry logic in `project new`, where a commit may already exist but the push failed.

## Outstanding Questions

None at this time.
