---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: `synchestra self-update`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/self-update?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/self-update?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/self-update?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/self-update?op=request-change) |
**Status:** Draft

**Parent:** [CLI](../README.md)

**Source Ideas:** —

## Summary

Top-level `synchestra self-update` command (alias `update`) that updates the installed `synchestra` binary in place. This Feature is deliberately **thin**: it binds the shared [`github.com/strongo/cli-helpers/selfupdate`](https://github.com/strongo/cli-helpers) module and specifies only what is genuinely synchestra's own — command surface and exit-code mapping. Install-method detection, release resolution, checksum verification, atomic replacement, and every failure rule are that library's own behavior contract, specified at [`strongo/cli-helpers`'s `self-update` Feature](https://specscore.studio/app/github.com/strongo/cli-helpers/spec/features/self-update?op=explore). Release identity itself (repository, tag prefix, supported platforms, asset/checksums naming) is no longer hand-specified here either: it is read from `strongo/cli-helpers`'s compiled-in install catalog (`cliinstall.ByID("synchestra")`), the single source every other fleet CLI's `install synchestra` also resolves releases from — see the [Install Command Library Feature](https://specscore.studio/app/github.com/strongo/cli-helpers/spec/features/cli-install?op=explore), specifically `REQ: catalog-identity-single-source` and the catalog's `synchestra` row in `REQ: relevance-matrix`. This document does not restate either contract; it only says how synchestra configures and consumes them.

## Problem

Every Synchestra CLI release ships as a standalone binary (`scripts/install.sh` / `scripts/install.ps1`, or a direct download from the release mirror — see the top-level README's Installation section). Without a self-update command, staying current means re-running the installer or manually downloading a new archive. `github.com/strongo/cli-helpers/selfupdate` already solves this generically — detecting how a binary was installed, verifying checksums, and swapping it in atomically — for several sibling Synchestra CLIs (`synchestra-channel`, `synchestra-vm-host`) and unrelated fleet CLIs (`wb`, `chatwright`, `ovdb`) alike, and its sibling `cliinstall` package now carries synchestra's release identity so the rest of the fleet can install synchestra the same way synchestra updates itself. Wiring it into `synchestra` itself is a matter of naming synchestra's catalog id and describing its exit-code contract to that library, not reimplementing any part of its logic or re-declaring an identity the catalog already owns.

## Behavior

### Command surface

#### REQ: top-level-command

`synchestra self-update` MUST be registered as a top-level command on the `synchestra` root, aliased `update`. Per the CLI Feature's noun-verb convention (`spec/features/cli/README.md`), this is a deliberate second bare-verb exception alongside `init`: there is no "resource" self-update acts on other than the CLI binary itself.

Flags (`--check`, `--yes`/`-y`, `--version`, `--allow-downgrade`, `--dry-run`, `--format text|json`) are registered by `cobracmd.New` from the shared library, not reimplemented here — see that library's own `self-update` Feature for their behavior.

### Release identity

#### REQ: release-identity

`synchestra self-update` MUST build its `selfupdate.Config` from `cliinstall.ByID("synchestra").Config(currentVersion)`, never from a hand-specified `selfupdate.Config` literal — this is the one release identity every other fleet CLI's `install synchestra` also resolves from (`cli-install#req:catalog-identity-single-source`). The catalog entry itself, not this document, is authoritative for the values; this repository's own offline test (`pkg/cli/selfupdate/catalog_naming_test.go`) proves the entry still agrees with `.goreleaser.yml`'s archive/checksum naming and `.github/workflows/release.yml`'s tag prefix and mirror repository, so a change to either without first updating the catalog fails this repository's own CI.

At the time of writing that identity is: releases resolved from the public `synchestra-io/synchestra-releases` mirror, not this source repository (`gh release list --repo synchestra-io/synchestra` is empty; this repo's own `.goreleaser.yml` sets `release.disable: true`); the `cli-` tag prefix, since the mirror also carries `servers-*` (synchestra-servers) and `vm-*` (synchestra-vm-host) releases in the same tag stream; and the library's own GoReleaser-shaped default asset/checksums naming (`synchestra_<version>_<os>_<arch>.tar.gz`/`.zip`, `synchestra_<version>_checksums.txt`), which the catalog entry leaves unoverridden.

#### REQ: version-identity

The undetermined-version placeholder set MUST be exactly `{"dev"}` — `pkg/cli`'s own `version` var's zero-configuration default, overwritten only by `.goreleaser.yml`'s link-time ldflag at release build time. This module never calls `runtime/debug.ReadBuildInfo`, so the Go toolchain's `"(devel)"` source-tree placeholder MUST NOT be declared: a placeholder that cannot occur must not be treated as a real, comparable value should it ever unexpectedly leak in. The catalog entry leaves `UndeterminedVersions` unset, deferring to the shared library's own `{"dev"}` default, which already equals this requirement's value.

#### REQ: supported-platforms

`SupportedPlatforms`, read from the catalog entry, MUST match `.goreleaser.yml`'s `builds.goos` x `builds.goarch` matrix exactly, including its explicit `windows`/`arm64` exclusion. A host outside this set MUST be refused by the shared library's own unsupported-platform rule rather than attempting a swap synchestra publishes no asset for.

#### REQ: no-package-manager

`Managers`, read from the catalog entry, MUST be empty (nil): `.goreleaser.yml` configures no `homebrew_casks`/`scoops`/`winget` publisher block anywhere in this repository. Every install therefore classifies as Manual or Ambiguous, never Redirected.

### Exit-code mapping

#### REQ: exit-code-mapping

`synchestra self-update` MUST map every outcome from the shared library onto synchestra's own exit-code contract (`spec/features/cli/README.md`'s "Exit code contract" table) using only the standard codes documented there — no command-group range (`20–109`) is reserved for self-update, per that table's own preference for standard codes whenever the error semantics fit one:

| Library outcome | Exit code | Why |
|---|---|---|
| `KindDowngrade`, `KindNonInteractive` | `2` InvalidArgs | Both are fixed by passing a different flag (`--allow-downgrade`, `--yes`) — exactly "missing or invalid command arguments/flags." |
| `KindUnknownTag`, `KindUnsupportedPlatform` | `3` NotFound | Both mean the release asset the operation needs does not exist — a bad `--version` pin, or no asset published for this host's platform at all. |
| `KindAmbiguous` | `4` InvalidState | The current install's location doesn't match any recognized pattern, so the self-replace transition is refused given that state — the same shape `pkg/cli/task/status.go` already uses InvalidState for. |
| `KindReleaseLookup`, `KindDownload`, `KindChecksum`, `KindPermission`, `KindUnexpected` | `10` Unexpected | Genuine runtime/operational failures (network, integrity, filesystem) that no different flag fixes and that name no missing resource or blocked state transition of their own. |
| `KindUnknownTarget` | `2` InvalidArgs | Belongs to the Install Command Library (`cliinstall`), not to `Config.Update`/`Config.Check` — a named install target that is not a catalog id, fixed by passing a valid one, the same "missing or invalid command arguments" shape as `KindDowngrade`/`KindNonInteractive` above (`cli-install#req:host-owned-exit-codes`). |
| `KindNoInstallDir`, `KindDestinationExists` | `4` InvalidState | Also `cliinstall`'s, not self-update's: no usable destination directory, or one already occupied — the same blocked-transition-given-current-state shape as `KindAmbiguous`. |
| `--check` verdict `UpdateAvailable` or `Undetermined` | `1` Conflict | The same code `pkg/cli/spec/lint.go` already uses for "this read-only inspection command found something to report" (`exitcode.ConflictErrorf("%d violation(s) found", ...)`) — `self-update --check` is structurally the same shape of command. |

`KindUnknownTarget`, `KindNoInstallDir`, and `KindDestinationExists` are mapped explicitly even though `synchestra self-update`'s own `Config.Update`/`Config.Check` never produce them (`cliinstall`'s planner does, once `synchestra install` exists) — appended after `KindManagedCommand` in the shared library's `FailureKind` enum, so no existing mapping's meaning changes, and pinned by `pkg/cli/selfupdate/selfupdate_test.go`'s `TestErrorMapperFailureNewKinds` ahead of `synchestra install`'s own thin Feature.

## Architecture

The command lives at `pkg/cli/selfupdate/selfupdate.go` (a top-level command package, peer of `pkg/cli/task/`, `pkg/cli/synchinit/`, etc.), registered on the root `cobra.Command` in `pkg/cli/main.go`. It contains no update logic and no hand-specified identity of its own: `newConfig` looks up `cliinstall.ByID("synchestra")` and calls its `Config(currentVersion)` method, `Command` wraps the result via `cobracmd.New`, and `errorMapper` implements `cobracmd.ErrorMapper` to apply the exit-code table above. All decision logic — detection, resolution, verification, replacement — lives in `github.com/strongo/cli-helpers/selfupdate`; the release identity itself lives in that module's sibling `cliinstall` package (`github.com/strongo/cli-helpers/cliinstall/catalog_synchestra.go`).

## Acceptance Criteria

### AC: registered-with-alias

**Requirements:** cli/self-update#req:top-level-command

**Given** a built `synchestra` binary
**When** the user runs `synchestra --help`
**Then** the help output lists `self-update` as a top-level command; running `synchestra update --help` resolves to the same command via its alias.

### AC: check-names-correct-product

**Requirements:** cli/self-update#req:release-identity, cli/self-update#req:supported-platforms

**Given** a host on a supported platform (e.g. `darwin/arm64`)
**When** the user runs `synchestra self-update --check` or `synchestra self-update --dry-run`
**Then** the resolved/planned release tag has the `cli-v` prefix and the planned asset name has the `synchestra_` prefix — never `servers-v*`/`synchestra-channel_*` or `vm-*`/`synchestra-vm-host_*`, the tag/asset shapes the same mirror publishes for the other Synchestra products.

### AC: undetermined-version-not-misreported

**Requirements:** cli/self-update#req:version-identity

**Given** a `synchestra` binary built via a plain `go build` (no release ldflags), so `pkg/cli.version == "dev"`
**When** the user runs `synchestra self-update --check`
**Then** the verdict is `Undetermined`, not `UpdateAvailable` — `"dev"` is never compared as if it were a real, orderable version.

### AC: exit-codes-match-table

**Requirements:** cli/self-update#req:exit-code-mapping

**Given** each outcome in the exit-code mapping table above
**When** `synchestra self-update` (or `--check`) produces that outcome
**Then** the process exits with the mapped code, and none of the `20–109` command-group-reserved ranges are ever used.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
