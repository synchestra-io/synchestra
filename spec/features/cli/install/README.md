---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: `synchestra install`

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/install?op=explore) | [Edit](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/install?op=edit) | [Ask question](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/install?op=ask) | [Request change](https://specscore.studio/app/github.com/synchestra-io/synchestra/spec/features/cli/install?op=request-change) |
**Status:** Draft

**Parent:** [CLI](../README.md)

**Source Ideas:** —

## Summary

Top-level `synchestra install` command that lists and installs the other
fleet CLIs relevant to synchestra. This Feature is deliberately **thin**: it
binds the shared
[`github.com/strongo/cli-helpers/cliinstall`](https://github.com/strongo/cli-helpers)
module (via its Cobra adapter, `cliinstall/cobracmd`) and specifies only what
is genuinely synchestra's own — command surface and exit-code mapping. The
relevance matrix, status probing, destination policy, the Homebrew-cask
decision, checksum-verified download, the confirmation gate, `--dry-run` and
`--format json` are that library's own behavior contract, specified at
[`strongo/cli-helpers`'s CLI Install Command Library
Feature](https://specscore.studio/app/github.com/strongo/cli-helpers/spec/features/cli-install?op=explore).
This document does not restate that contract; it only says how synchestra
configures and consumes it. Release identity itself (repository, tag prefix,
supported platforms, asset/checksums naming) is not hand-specified here
either: it is read from the same compiled-in catalog entry
[self-update](../self-update/README.md) builds its `selfupdate.Config` from
(`cliinstall.ByID("synchestra")`,
cli-install#req:catalog-identity-single-source).

## Problem

synchestra's catalog entry declares it relevant to `ingitdb`, `specscore`,
`datatug`, `ovdb` and `chatwright` (the [CLI Install Command
Library](https://github.com/strongo/cli-helpers/blob/main/spec/features/cli-install/README.md)'s
relevance matrix), and several of those CLIs' own `install` command lists
`synchestra` in turn. Without `synchestra install` itself, a user of one of
those CLIs could discover and install `synchestra`, but a user who started
from `synchestra` had no equivalent way to discover inGitDB, SpecScore,
DataTug, OpenVaultDB or Chatwright, or to install any of them consistently
with how they installed `synchestra` — the same asymmetry
[self-update](../self-update/README.md)'s Problem section describes for the
download-verify-swap machinery, one layer up.

## Behavior

### Command surface

#### REQ: command-name

The CLI MUST expose the command as `synchestra install`, built from
`github.com/strongo/cli-helpers/cliinstall/cobracmd`. The command inherits
the library's full flag surface — `--all`, `--yes`/`-y`, `--dry-run`,
`--dir`, and `--format text|json` — none of which is re-specified here.
`install` MUST NOT gain an `update` alias: `synchestra self-update` alone
keeps that alias, per
[cli-install#req:update-alias-policy](https://specscore.studio/app/github.com/strongo/cli-helpers/spec/features/cli-install?op=explore).

### Host identity

#### REQ: host-id

`synchestra install` MUST identify the host to the library as catalog id
`"synchestra"` only (`cobracmd.CommandOptions.HostID`), never a hand-written
duplicate of the catalog entry (cli-install#req:host-identity-from-catalog).
`cliinstall.ByID("synchestra")` is the same entry `self-update` resolves
(`cliinstall/catalog_synchestra.go` in `strongo/cli-helpers`), so `install`'s
relevant-targets listing, and every other fleet CLI's `install synchestra`,
resolve synchestra's own release identically. `cobracmd.New` panics if
`"synchestra"` is absent from the compiled catalog — a programming error
`TestCatalogHasSynchestra` and `TestCommandRegistration` catch, never a
runtime state a user sees.

### Exit codes

#### REQ: exit-codes

`synchestra install` MUST map every outcome from the shared library onto
synchestra's own exit-code contract (`spec/features/cli/README.md`'s "Exit
code contract" table), kind-for-kind with
[self-update's own mapping](../self-update/README.md#req-exit-code-mapping)
for every kind the two commands can both produce
(cli-install#req:host-owned-exit-codes: hosts "keep their self-update
mapping for the shared kinds"), and adds the two new kinds `cli-install`
requires every host to map explicitly:

| Library outcome | Exit code | Why |
|---|---|---|
| `*cobracmd.UsageError` (invalid `--format`, or `--all` with names), `KindDowngrade`, `KindNonInteractive` | `2` InvalidArgs | All are fixed by passing a different flag — exactly "missing or invalid command arguments/flags." |
| `KindUnknownTag`, `KindUnsupportedPlatform` | `3` NotFound | Both mean the release asset the operation needs does not exist. |
| `KindAmbiguous` | `4` InvalidState | The current install's location doesn't match any recognized pattern, so the transition is refused given that state. |
| `KindReleaseLookup`, `KindDownload`, `KindChecksum`, `KindPermission`, `KindManagedCommand`, `KindUnexpected` | `10` Unexpected | Genuine runtime/operational failures that no different flag fixes and that name no missing resource or blocked state transition of their own. |
| `KindUnknownTarget` | `2` InvalidArgs | A named install target that is not a catalog id, fixed by passing a valid one — the same shape as `KindDowngrade`/`KindNonInteractive` (cli-install#req:host-owned-exit-codes). |
| `KindNoInstallDir`, `KindDestinationExists` | `4` InvalidState | No usable destination directory, or one already occupied — the same shape as `KindAmbiguous`. |

`install nosuchcli` MUST fail before any confirmation, network request or
write, name the unknown target, and exit `2`
(cli-install#req:unknown-target-refused).

## Architecture

The command lives at `pkg/cli/install/install.go` (a top-level command
package, peer of `pkg/cli/selfupdate/`, `pkg/cli/task/`, etc.), registered on
the root `cobra.Command` in `pkg/cli/main.go`. It contains no listing,
planning or download logic of its own: `Command` looks up the catalog
through `HostID: "synchestra"` and wraps `cobracmd.New`, and `errorMapper`
implements `cobracmd.ErrorMapper` to apply the exit-code table above. All
decision logic — the relevance matrix, status probing, destination policy,
checksum-verified download, the confirmation gate — lives in
`github.com/strongo/cli-helpers/cliinstall`; the release identity itself
lives in that module's sibling `cliinstall/catalog_synchestra.go`, the same
file [self-update](../self-update/README.md) reads.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [self-update](../self-update/README.md) | Both commands build from the same `cliinstall.ByID("synchestra")` catalog entry, so `install`'s view of synchestra (shown by other CLIs) and `self-update`'s own release identity never disagree; their exit-code mappings share every kind both can produce. |

## Acceptance Criteria

### AC: registration-and-host-id

**Requirements:** cli/install#req:command-name, cli/install#req:host-id

**Given** the compiled `cliinstall` catalog
**When** `Command()` builds the command
**Then** it registers `--all`, `--yes`/`-y`, `--dry-run`, `--dir` and
`--format`, resolves against catalog id `"synchestra"`, and carries no
`update` alias.

### AC: unknown-target-exit-code

**Requirements:** cli/install#req:exit-codes

**Given** the real command built exactly as `pkg/cli/main.go` wires it
**When** the user runs `synchestra install nosuchcli`
**Then** the command fails before any confirmation, network request or
write, names `nosuchcli` in its error, and exits `2` (InvalidArgs).

The remaining behavior — the relevance matrix, listing and status probing,
destination policy, Homebrew-cask installs, checksum-verified direct
installs, the confirmation gate, `--dry-run`, and `--format json` — is
specified and tested once in the
[CLI Install Command Library](https://github.com/strongo/cli-helpers/blob/main/spec/features/cli-install/README.md)'s
own Acceptance Criteria, which this command inherits by construction rather
than re-proving.

## Open Questions

None at this time.

---
*This document follows the https://specscore.md/feature-specification*
