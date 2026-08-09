// Package selfupdate wires the "synchestra self-update" command (aliased
// "update") onto the shared github.com/strongo/selfupdate module. It
// implements none of the update logic itself: install-method detection,
// release resolution, checksum verification, atomic replacement, and every
// failure rule belong to that library. This package supplies only what is
// genuinely synchestra's own — its release identity (the public
// synchestra-io/synchestra-releases mirror, keyed by the "cli-" tag prefix),
// the platforms .goreleaser.yml actually publishes, and the mapping from the
// library's outcomes onto synchestra's own exit-code contract (documented in
// spec/features/cli/README.md's "Exit code contract" table). See
// spec/features/cli/self-update/README.md for the Feature spec that draws
// this boundary.
package selfupdate

// Features implemented: cli/self-update

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/strongo/selfupdate"
	"github.com/strongo/selfupdate/cobracmd"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// undeterminedVersions lists every value pkg/cli's `version` var can hold
// that does not identify a real release. synchestra has exactly one: "dev",
// that var's own zero-configuration default (pkg/cli/main.go), overwritten
// only by .goreleaser.yml's link-time
// "-X github.com/synchestra-io/synchestra/pkg/cli.version={{.Version}}"
// ldflag at release build time. This module never calls
// runtime/debug.ReadBuildInfo anywhere (confirmed by grep across the whole
// repository), so the Go toolchain's own "(devel)" source-tree placeholder
// is not a value `version` can ever take — it is deliberately NOT declared
// here. An undeclared placeholder that can actually occur would compare as a
// real version and report an update available FROM a version that does not
// exist; declaring one that cannot occur would just be noise.
var undeterminedVersions = []string{"dev"}

// newConfig returns synchestra's own selfupdate.Config: its release
// identity and the platforms .goreleaser.yml's build matrix publishes.
// Every other field is left at the library's GoReleaser-shaped defaults
// because this CLI's own release naming already matches them exactly — see
// the field comments below for how that was verified. newConfig is a plain
// function, not inlined into Command, purely so selfupdate_test.go can
// assert its fields directly without constructing a command or touching any
// I/O.
func newConfig(currentVersion string) selfupdate.Config {
	return selfupdate.Config{
		BinaryName: "synchestra",
		// Repository is the PUBLIC mirror, not this source repository:
		// `gh release list --repo synchestra-io/synchestra` is empty — this
		// repo's own .goreleaser.yml sets release.disable: true and
		// publishes no GitHub Release here at all. Every synchestra CLI
		// release actually lands in synchestra-io/synchestra-releases,
		// confirmed by .github/workflows/release.yml's publish-releases
		// job, which uploads dist/*.tar.gz, dist/*.zip, and
		// dist/synchestra_*_checksums.txt to that repository unchanged
		// (no renaming step).
		Repository: "synchestra-io/synchestra-releases",
		// TagPrefix distinguishes this CLI's own releases ("cli-v0.15.1")
		// from the other Synchestra products the same mirror repository
		// carries (e.g. "servers-v...", "vm-..."). release.yml computes
		// RELEASE_TAG="cli-${TAG}" for exactly this reason. Without a
		// prefix, "latest release" resolution has no way to tell the
		// products apart. Confirmed against a real published release:
		// `gh release view cli-v0.15.1 --repo synchestra-io/synchestra-releases`
		// lists "synchestra_0.15.1_<os>_<arch>.tar.gz" assets and a
		// "synchestra_0.15.1_checksums.txt" — exactly the library's own
		// GoReleaser-shaped defaults for AssetName/ChecksumsName (see
		// selfupdate.Config's own doc comments), so neither is overridden
		// below.
		TagPrefix:            "cli-",
		CurrentVersion:       currentVersion,
		UndeterminedVersions: undeterminedVersions,
		// .goreleaser.yml configures no homebrew_casks/scoops/winget
		// publisher block anywhere in this repository —
		// scripts/install.sh's curl-based installer is the only documented
		// install path (README.md's Installation section). A nil Managers
		// is correct: every install classifies as Manual or Ambiguous,
		// never Redirected.
		Managers: nil,
		// Matches .goreleaser.yml's builds.goos x builds.goarch exactly,
		// including its one explicit exclusion: windows/arm64 is `ignore`d
		// there (scripts/install.sh independently refuses that combination
		// too — "windows/arm64 is not released; please build from
		// source"). A host outside this set is refused by the library's
		// own unsupported-platform rule rather than attempting a swap
		// synchestra publishes no asset for.
		SupportedPlatforms: []selfupdate.Platform{
			{GOOS: "darwin", GOARCH: "amd64"},
			{GOOS: "darwin", GOARCH: "arm64"},
			{GOOS: "linux", GOARCH: "amd64"},
			{GOOS: "linux", GOARCH: "arm64"},
			{GOOS: "windows", GOARCH: "amd64"},
		},
		// `synchestra version` (github.com/ingitdb/ingitdb-cli's
		// commands.Version, wired in pkg/cli/main.go) prints
		// "ingitdb <version> (<commit>) @ <date>\n" — the "ingitdb" prefix
		// is baked into that shared command, not "synchestra", but the
		// bare version string this probe needs (see the library's own
		// post-swap verifyBinaryVersion, which only checks the output
		// CONTAINS the target version) is present in that line regardless.
		// This overrides the library's own default ("--version"): this
		// CLI's root command has no bare --version flag of its own that
		// reliably prints just the version (fang.WithVersion formats it
		// alongside other framework chrome), so the explicit "version"
		// subcommand is the one output guaranteed to contain the raw
		// version string.
		VersionProbeArgs: []string{"version"},
		// AssetName, ChecksumsName, ReleasesAPIURL, DownloadURL, and
		// HTTPClient are all left at the library's GoReleaser-shaped
		// defaults — see the TagPrefix comment above for how those were
		// verified against a real published release.
	}
}

// Command returns the "self-update" command (aliased "update"). Every
// decision it makes — install-method detection, checksum-verified atomic
// replacement, the downgrade guard, the confirmation gate — comes from
// cobracmd.New and the selfupdate.Config newConfig returns. This function's
// only job is to describe synchestra to that library and to translate its
// outcomes onto synchestra's own exit-code contract via errorMapper.
//
// currentVersion is pkg/cli's own `version` var, threaded through explicitly
// (rather than read as a package-level import) so this package stays
// testable without depending on pkg/cli's link-time state.
func Command(currentVersion string) *cobra.Command {
	return cobracmd.New(newConfig(currentVersion), cobracmd.CommandOptions{
		Short:      "Update the installed synchestra binary to the latest release",
		Aliases:    []string{"update"},
		JSONFormat: true,
		Errors:     errorMapper{},
	})
}

// errorMapper maps github.com/strongo/selfupdate's outcomes onto
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
func (errorMapper) Failure(err error) error {
	msg := fmt.Sprintf("self-update: %v", err)
	switch selfupdate.KindOf(err) {
	case selfupdate.KindDowngrade, selfupdate.KindNonInteractive:
		return exitcode.InvalidArgsError(msg)
	case selfupdate.KindUnknownTag, selfupdate.KindUnsupportedPlatform:
		return exitcode.NotFoundError(msg)
	case selfupdate.KindAmbiguous:
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
