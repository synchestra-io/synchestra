# Feature: `synchestra init`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/init?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/init?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/init?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/init?op=request-change) |

**Status:** Approved

**Parent:** [CLI](../README.md)

**Source Ideas:** unified-project-definition

## Summary

Top-level `synchestra init` command — the one-stop bootstrap entry-point for a Synchestra-managed project. Creates `synchestra.yaml` at the repo root, sets up the chosen state mode (embedded by default), and is idempotent on rerun. Top-level placement deliberately breaks Synchestra's strict noun-verb CLI convention for the bootstrap entry-point — the same exception every CLI ecosystem makes (`git init`, `npm init`, `cargo init`).

This Feature defines the contract. Per the unified Idea, the legacy `project init` and `project new` subcommands have been removed; `synchestra init` is the sole bootstrap surface. v1 ships embedded mode only — separate-repo and hub-managed modes are recognized by the flag parser but exit `2` with "not yet implemented." The surface is designed to accept them in a future release.

## Problem

Earlier Synchestra CLI iterations carried two parallel bootstrap commands — `synchestra project init` (embedded mode, single repo) and `synchestra project new` (multi-repo project). New users had to choose between them based on internal architecture knowledge, and the choice was irreversible without manual config-file surgery (starting with `project init` and later wanting Hub registration meant editing YAML files by hand).

A top-level `synchestra init` collapses the decision into a single entrypoint that takes a `--state-mode` choice. The user no longer needs to know which subcommand to invoke; the CLI surface matches the muscle-memory established by `git init`, `npm init`, and `cargo init`. Per the source Idea's "no users yet, no migration" stance, the legacy commands were deleted outright rather than aliased.

## Behavior

### Top-level placement

#### REQ: top-level-command

`synchestra init` MUST be registered as a top-level command on the synchestra root. It is the only command in v1 that breaks the noun-verb convention. The CLI [README](../README.md) MUST document this exception explicitly.

### Mode selection

#### REQ: state-mode-flag

`synchestra init` accepts a `--state-mode <mode>` flag. The accepted values are `embedded`, `separate-repo`, and `hub-managed`. The default is `embedded`. Unknown values produce exit code `2` (invalid arguments). In v1 only `embedded` is implemented; passing `separate-repo` or `hub-managed` produces exit code `2` with a message stating the mode is recognized but not yet implemented.

### Pre-flight checks

#### REQ: requires-git-repo

`synchestra init` MUST verify that the current working directory (or `--project` flag's value) is inside a git repository. If not, it MUST exit with code `3` (resource not found) and a message instructing the user to run `git init` first. The command MUST NOT initialize a git repo on the user's behalf.

#### REQ: project-flag

The command accepts a `--project <path>` flag overriding the project root. The path MUST exist and MUST be a directory; otherwise exit code `2`. When omitted, the command operates on the current working directory.

### `synchestra.yaml` creation

#### REQ: writes-synchestra-yaml

On a successful greenfield run, `synchestra init` MUST create `synchestra.yaml` at the project root with:

1. Line 1 = the canonical schema-header comment per [`repo-config#req:schema-header-comment`](../../repo-config/README.md).
2. A `state:` block reflecting the chosen `--state-mode`. For embedded mode, this is `state: { mode: embedded }` with `branch:` populated from `--branch` (default `synchestra-state`).
3. A blank line between the schema header and the body (matching SpecScore's WriteSpecConfig style).

The command MUST NOT write any other top-level field that is not explicitly declared above. Identity fields (`title`, `host`, `org`, `repo`, `repositories`) are forbidden in `synchestra.yaml` per [`repo-config#req:specscore-yaml-as-identity-source`](../../repo-config/README.md).

### State setup

#### REQ: embedded-state-setup

When `--state-mode embedded`, `synchestra init` MUST set up the orphan branch and worktree exactly as the existing [`embedded-state`](../../embedded-state/README.md) Feature describes. Specifically:

1. Create the orphan branch (default `synchestra-state`) if it does not already exist.
2. Write `synchestra-state.yaml` (per the [state-repo-config](../../state-repo-config/README.md) Feature) on the orphan branch root with `mode: embedded` and a `spec_repos:` entry for the current repo's origin URL.
3. Create a worktree at `.synchestra/` pointing at the orphan branch.

The `.gitignore` update is governed separately by `gitignore-synchestra-entry`. Implementation may factor common helpers into shared internal packages; the surface contract is what this REQ governs.

### Idempotence

#### REQ: idempotent-rerun

`synchestra init` MUST be safe to rerun. On the second invocation in the same project:

1. If `synchestra.yaml` already exists and is valid (parses successfully, line-1 header correct), the command MUST NOT rewrite it. The exit code is `0`.
2. If the orphan branch already exists, the command MUST NOT recreate it. The worktree is recreated only when missing.
3. If the existing `synchestra.yaml`'s state mode disagrees with the requested `--state-mode`, the command MUST exit with code `4` (invalid state transition) and surface both modes; rerun with `--force` (a future flag) is the documented escape hatch — for v1 there is no `--force`, the user must edit the file manually.

### `.gitignore` management

#### REQ: gitignore-synchestra-entry

When `--state-mode embedded` and the current branch's working tree contains a `.gitignore`, `synchestra init` MUST ensure that file contains a line `.synchestra` (or a more specific pattern that already matches). If `.gitignore` does not exist, it MUST be created with the single line `.synchestra`. The command MUST preserve any pre-existing content of `.gitignore`.

### Output

#### REQ: success-output

On a successful first-run init, the command MUST print to stdout:

1. A one-line summary naming the created `synchestra.yaml` path and the resolved state mode.
2. The orphan branch name and worktree path (when applicable).
3. A "next step" hint pointing the user at the appropriate follow-on command (`synchestra task new`, `synchestra info`, etc.).

On a no-op rerun the command MUST print a single line indicating the project is already initialized and exit `0`.

## Architecture

The command lives at `pkg/cli/init/init.go` (a new top-level command package, peer of `pkg/cli/project/`, `pkg/cli/task/`, etc.). Its `RunE` handler:

1. Parses flags.
2. Resolves the project root (from `--project` or `os.Getwd()`).
3. Verifies git-repo membership.
4. Reads any existing `synchestra.yaml` to detect rerun-vs-first-time.
5. Dispatches to mode-specific helpers (only `runEmbedded` exists in v1).
6. The embedded-mode helper reuses orphan-branch / worktree logic from `pkg/cli/project/` (refactored into a shared internal package as part of the implementation; this is the only structural refactor introduced by this Feature).

The command is registered on the root `cobra.Command` in `pkg/cli/main.go` alongside the existing top-level groups.

## Acceptance Criteria

### AC: top-level-registered

**Requirements:** cli/init#req:top-level-command

**Given** a built `synchestra` binary
**When** the user runs `synchestra --help`
**Then** the help output lists `init` as a top-level command (not nested under `project`); when the user runs `synchestra init --help`, the help output describes the bootstrap purpose and lists the `--state-mode`, `--project`, and `--branch` flags.

### AC: greenfield-embedded-init

**Requirements:** cli/init#req:state-mode-flag, cli/init#req:requires-git-repo, cli/init#req:writes-synchestra-yaml, cli/init#req:embedded-state-setup, cli/init#req:gitignore-synchestra-entry, cli/init#req:success-output

**Given** an empty git repository at a temp directory with no pre-existing `synchestra.yaml`, `.synchestra/`, or `synchestra-state` branch
**When** the user runs `synchestra init` (embedded default) inside that directory
**Then** the command exits `0`; a file `synchestra.yaml` exists at the project root whose line 1 reads exactly `# Synchestra Repo Config Schema: https://synchestra.md/repo-config` (per `repo-config#req:schema-header-comment`) and whose body contains a `state: { mode: embedded, branch: synchestra-state }` block; an orphan branch named `synchestra-state` exists carrying a `synchestra-state.yaml` with `mode: embedded` and a `spec_repos:` entry pointing at this repo; a `.synchestra/` worktree directory exists; `.gitignore` on the source branch contains `.synchestra`; stdout summarises the created file paths, state mode, branch name, and a next-step hint.

### AC: rejects-non-git-dir

**Requirements:** cli/init#req:requires-git-repo

**Given** a directory that is not inside any git repository
**When** the user runs `synchestra init` in that directory
**Then** the command exits `3`; stderr contains a message instructing the user to run `git init` first; no `synchestra.yaml` is created; no orphan branch is created.

### AC: rejects-unknown-mode

**Requirements:** cli/init#req:state-mode-flag

**Given** any git repository
**When** the user runs `synchestra init --state-mode magical`
**Then** the command exits `2`; stderr names the invalid value and lists the accepted values (`embedded`, `separate-repo`, `hub-managed`); no files are written.

### AC: rejects-unimplemented-mode

**Requirements:** cli/init#req:state-mode-flag

**Given** any git repository
**When** the user runs `synchestra init --state-mode separate-repo` (or `hub-managed`)
**Then** the command exits `2`; stderr names the requested mode and states it is recognized but not yet implemented in v1; no files are written.

### AC: idempotent-rerun

**Requirements:** cli/init#req:idempotent-rerun

**Given** a project that has already been initialized via `synchestra init` (embedded mode)
**When** the user runs `synchestra init` again with no arguments
**Then** the command exits `0`; the existing `synchestra.yaml` is not rewritten (mtime unchanged, contents byte-for-byte identical to the pre-rerun state); the existing orphan branch is unchanged; stdout indicates the project is already initialized; if `.synchestra/` worktree was missing it is recreated.

### AC: rejects-mode-mismatch

**Requirements:** cli/init#req:idempotent-rerun

**Given** a project initialized in embedded mode
**When** the user runs `synchestra init --state-mode separate-repo`
**Then** the command exits `4`; stderr names both the existing mode and the requested mode and instructs the user to edit `synchestra.yaml` manually (since `--force` is not available in v1); no files are modified.

### AC: project-flag-resolves

**Requirements:** cli/init#req:project-flag

**Given** a git repository at `/tmp/repo` and the user's current working directory `/tmp/elsewhere`
**When** the user runs `synchestra init --project /tmp/repo` from `/tmp/elsewhere`
**Then** `synchestra.yaml` is created at `/tmp/repo/synchestra.yaml`, not at `/tmp/elsewhere/`; running with `--project /nonexistent` exits `2`.

## Open Questions

- The `--force` flag for re-init (allowing mode change) is intentionally deferred to a follow-up Feature. The MVP's escape hatch is manual file editing.
- ~~Deprecation warnings on `synchestra project init` and `synchestra project new`~~ — resolved: per the source Idea's "no users yet, no migration" stance, both legacy commands were deleted outright rather than aliased.
- Whether `synchestra init` should also write a starter `synchestra.yaml` containing a commented-out example of the `hub:` block (to aid discovery) is a UX choice deferred to implementation.
- The shared internal package for orphan-branch / worktree logic (factored out of `pkg/cli/project/`) needs its own naming decision; package name TBD at implementation time.

---
*This document follows the https://specscore.md/feature-specification*
