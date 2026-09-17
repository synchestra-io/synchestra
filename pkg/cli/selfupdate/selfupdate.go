// Package selfupdate wires the "synchestra self-update" command (aliased
// "update") onto the shared github.com/strongo/cli-helpers/selfupdate
// module. It implements none of the update logic itself: install-method
// detection, release resolution, checksum verification, atomic replacement,
// and every failure rule belong to that library. This package supplies only
// what is genuinely synchestra's own — reading its release identity (the
// public synchestra-io/synchestra-releases mirror, keyed by the "cli-" tag
// prefix, and the platforms .goreleaser.yml actually publishes) from the
// Install Command Library's compiled-in catalog
// (github.com/strongo/cli-helpers/cliinstall), so this identity is the same
// single source every other fleet CLI's `install synchestra` resolves
// releases from (cli-install#req:catalog-identity-single-source), and the
// mapping from the library's outcomes onto synchestra's own exit-code
// contract (documented in spec/features/cli/README.md's "Exit code
// contract" table). See spec/features/cli/self-update/README.md for the
// Feature spec that draws this boundary.
package selfupdate

// Features implemented: cli/self-update

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/strongo/cli-helpers/cliinstall"
	"github.com/strongo/cli-helpers/selfupdate"
	"github.com/strongo/cli-helpers/selfupdate/cobracmd"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// catalogID is synchestra's own id in the Install Command Library's
// compiled-in catalog (cli-install#req:host-identity-from-catalog). An id
// absent from the catalog is a programming error caught by
// TestNewConfigIdentity, never a runtime state a user sees.
const catalogID = "synchestra"

// newConfig returns synchestra's own selfupdate.Config, built from its
// cliinstall.Entry (cli-helpers/cliinstall/catalog_synchestra.go): the
// public mirror repository, the "cli-" tag prefix, the darwin/linux/windows
// x amd64/arm64 platforms .goreleaser.yml publishes (minus windows/arm64,
// which it explicitly ignores), the "version" probe subcommand, and no
// package managers (.goreleaser.yml publishes no homebrew_casks/scoops/
// winget block; scripts/install.sh's curl-based installer is the only
// documented install path). Every field the catalog entry carries is
// reproduced exactly by Entry.Config — see cliinstall.Entry.Config's own
// doc comment — so this function adds nothing beyond currentVersion.
// AssetName, ChecksumsName, ReleasesAPIURL, DownloadURL, and HTTPClient are
// all left at the library's GoReleaser-shaped defaults, which the catalog
// entry also leaves unset because synchestra's own release naming already
// matches them exactly (confirmed against a real published release: `gh
// release view cli-v0.15.1 --repo synchestra-io/synchestra-releases` lists
// "synchestra_0.15.1_<os>_<arch>.tar.gz" assets and a
// "synchestra_0.15.1_checksums.txt").
//
// newConfig is a plain function, not inlined into Command, purely so
// selfupdate_test.go can assert its fields directly without constructing a
// command or touching any I/O.
func newConfig(currentVersion string) selfupdate.Config {
	entry, ok := cliinstall.ByID(catalogID)
	if !ok {
		// Caught by TestNewConfigIdentity at compile-review time; never a
		// runtime state a user can trigger (cli-install#req:host-identity-
		// from-catalog).
		panic("selfupdate: catalog has no entry for " + catalogID)
	}
	return entry.Config(currentVersion)
}

// NewConfig is newConfig, exported so pkg/cli/upgrade can build its
// HostConfig from the exact same catalog identity Command does, making
// `synchestra self-update` and `synchestra upgrade synchestra` reach the
// same library call by construction
// (cli-install#req:self-update-equals-upgrade-self).
func NewConfig(currentVersion string) selfupdate.Config { return newConfig(currentVersion) }

// Command returns the "self-update" command (aliased "update"). Every
// decision it makes — install-method detection, checksum-verified atomic
// replacement, the downgrade guard, the confirmation gate — comes from
// cobracmd.New and the selfupdate.Config newConfig returns. This function's
// only job is to describe synchestra to that library and to translate its
// outcomes onto synchestra's own exit-code contract via errorMapper.
//
// currentVersion is pkg/cli's own buildinfo.Info.Version (resolved by
// buildinfo.Get in pkg/cli/main.go), threaded through explicitly (rather
// than read as a package-level import) so this package stays testable
// without depending on pkg/cli's link-time state.
func Command(currentVersion string) *cobra.Command {
	return cobracmd.New(newConfig(currentVersion), cobracmd.CommandOptions{
		Short:      "Update the installed synchestra binary to the latest release",
		Aliases:    []string{"update"},
		JSONFormat: true,
		Errors:     errorMapper{},
	})
}

// errorMapper maps github.com/strongo/cli-helpers/selfupdate's outcomes onto
// synchestra's own exit-code contract, documented in
// spec/features/cli/README.md's "Exit code contract" table:
//
//	0  Success
//	1  Conflict     — concurrent-modification conflict
//	2  InvalidArgs  — missing or invalid command arguments/flags
//	3  NotFound     — requested resource does not exist
//	4  InvalidState — state transition is not allowed
//	10 Unexpected   — catch-all runtime error
//
// That same table says: "Standard exit codes (0–10) should be preferred
// whenever possible. Use a group-specific code only when the error
// semantics cannot be expressed by a standard code." Every one of
// selfupdate's FailureKinds, and both non-UpToDate --check verdicts, fit a
// standard code (see Failure and UpdateAvailable below), so self-update
// reserves none of the 20–109 command-group ranges the table lists for
// project/feature/task/spec/state/code/runner/session/auth.
type errorMapper struct{}

// Failure maps a non-nil error from Config.Update or Config.Check onto the
// standard code whose documented meaning it actually matches:
//
//   - KindDowngrade (a pinned --version older than the running build,
//     without --allow-downgrade) and KindNonInteractive (confirmation
//     needed, no --yes and no terminal) are both fixed by passing a
//     different flag — exactly "missing or invalid command arguments/
//     flags" (InvalidArgs's documented meaning) — so both map to
//     InvalidArgs.
//   - KindUnknownTag (a pinned --version tag matching no published
//     release, or no asset for this platform within that release) and
//     KindUnsupportedPlatform (no release asset is configured for this
//     host's platform at all) both mean the same thing at the exit-code
//     level: the release asset the operation needs does not exist. Both
//     map to NotFound.
//   - KindAmbiguous means the current install's location doesn't match any
//     recognized pattern, so the self-replace transition is refused given
//     that state — the same "a guard blocks the requested transition given
//     current state" shape pkg/cli/task/status.go already uses InvalidState
//     for ("status guard failed: expected %s, got %s"). Maps to
//     InvalidState.
//   - KindReleaseLookup, KindDownload, KindChecksum, KindPermission, and
//     KindUnexpected are genuine runtime/operational failures (network,
//     integrity, filesystem) that no different flag fixes and that name no
//     missing resource or blocked state transition of their own. All map
//     to the Unexpected catch-all.
//   - KindUnknownTarget, KindNoInstallDir, and KindDestinationExists belong
//     to the Install Command Library (cli-install, package cliinstall)
//     built on this same self-update library, not to self-update itself —
//     Config.Update and Config.Check never produce them. They are mapped
//     explicitly here anyway, ahead of this package's own `install` command
//     (a later change), so the mapping in
//     cli-install#req:host-owned-exit-codes ("Every host MUST map the three
//     new kinds explicitly … MUST NOT let them fall into a self-update
//     default branch") is pinned by TestErrorMapperFailure* from the day
//     the library started producing them, not only once `install` exists.
//     KindUnknownTarget is fixed by passing a valid target name — the same
//     "missing or invalid command arguments" shape as KindDowngrade and
//     KindNonInteractive above — so it maps to InvalidArgs. KindNoInstallDir
//     and KindDestinationExists both mean a destination cannot be used as
//     asked (no directory on PATH, or a file already occupies it) — the
//     same blocked-transition-given-current-state shape as KindAmbiguous —
//     so both map to InvalidState.
func (errorMapper) Failure(err error) error {
	msg := fmt.Sprintf("self-update: %v", err)
	switch selfupdate.KindOf(err) {
	case selfupdate.KindDowngrade, selfupdate.KindNonInteractive:
		return exitcode.InvalidArgsError(msg)
	case selfupdate.KindUnknownTag, selfupdate.KindUnsupportedPlatform:
		return exitcode.NotFoundError(msg)
	case selfupdate.KindAmbiguous:
		return exitcode.InvalidStateError(msg)
	case selfupdate.KindUnknownTarget:
		return exitcode.InvalidArgsError(msg)
	case selfupdate.KindNoInstallDir, selfupdate.KindDestinationExists:
		return exitcode.InvalidStateError(msg)
	default: // KindReleaseLookup, KindDownload, KindChecksum, KindPermission, KindUnexpected
		return exitcode.UnexpectedError(msg)
	}
}

// UpdateAvailable maps a --check verdict that is not up to date (an update
// available, or a running version too undetermined to compare) onto
// Conflict — the same code pkg/cli/spec/lint.go already uses for "this
// read-only inspection command found something to report"
// (exitcode.ConflictErrorf("%d violation(s) found", ...)). `self-update
// --check` is structurally the same kind of command: read-only, and its
// finding — an update exists, or the running version can't be classified
// against the latest release — is that same shape of non-clean result, not
// a runtime failure, so it does not belong on the Unexpected catch-all.
func (errorMapper) UpdateAvailable(res selfupdate.CheckResult) error {
	if res.Verdict == selfupdate.Undetermined {
		return exitcode.ConflictErrorf("self-update: current version is undetermined (%s); latest stable is %s", res.Current, res.Latest)
	}
	return exitcode.ConflictErrorf("self-update: update available (%s -> %s)", res.Current, res.Latest)
}
