package selfupdate

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/strongo/cli-helpers/cliinstall"
	"github.com/strongo/cli-helpers/selfupdate"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// TestNewConfigIdentity pins synchestra's own release identity: the release
// repository is the PUBLIC mirror (not this source repo, which publishes no
// GitHub Release of its own), gated by the "cli-" tag prefix that separates
// this CLI's releases from the other Synchestra products the same mirror
// carries. It also pins that this identity comes from the catalog id
// "synchestra" (cli-install#req:host-identity-from-catalog): an id absent
// from cli-helpers/cliinstall's compiled-in catalog would panic in
// newConfig rather than surface as a runtime state, so this test is what
// actually catches that.
func TestNewConfigIdentity(t *testing.T) {
	if _, ok := cliinstall.ByID(catalogID); !ok {
		t.Fatalf("cliinstall has no catalog entry for id %q", catalogID)
	}

	cfg := newConfig("0.15.1")

	if cfg.BinaryName != "synchestra" {
		t.Errorf("BinaryName = %q, want %q", cfg.BinaryName, "synchestra")
	}
	if cfg.Repository != "synchestra-io/synchestra-releases" {
		t.Errorf("Repository = %q, want %q", cfg.Repository, "synchestra-io/synchestra-releases")
	}
	if cfg.TagPrefix != "cli-" {
		t.Errorf("TagPrefix = %q, want %q", cfg.TagPrefix, "cli-")
	}
	if cfg.CurrentVersion != "0.15.1" {
		t.Errorf("CurrentVersion = %q, want %q", cfg.CurrentVersion, "0.15.1")
	}
}

// TestNewConfigUndeterminedVersions pins REQ: version-identity. synchestra's
// buildinfo.Info.Version (resolved by github.com/strongo/buildinfo.Get,
// wired in pkg/cli/main.go) has exactly one non-release value in practice,
// "dev", buildinfo.Get's own final fallback. That is also exactly
// selfupdate.Config's own documented default when UndeterminedVersions is
// left empty (see Config.UndeterminedVersions's doc comment), so the
// cliinstall.Entry this package reads (cli-helpers/cliinstall/
// catalog_synchestra.go) correctly declares no override at all — this test
// pins that the catalog entry stays that way rather than starting to name
// an explicit set that could drift from the shared default, including the
// Go toolchain's own "(devel)" source-tree placeholder (never produced by a
// real release build, whose -X stamps are always set) or a Go pseudo-version
// (a KNOWN version that must sort below its eventual release, not be swept
// into the undetermined set).
func TestNewConfigUndeterminedVersions(t *testing.T) {
	cfg := newConfig("dev")

	if len(cfg.UndeterminedVersions) != 0 {
		t.Errorf("UndeterminedVersions = %v, want empty (catalog entry defers to selfupdate.Config's own {\"dev\"} default)", cfg.UndeterminedVersions)
	}
}

// TestNewConfigNoManagers pins that .goreleaser.yml publishes through no
// package manager (no homebrew_casks/scoops/winget block) — every install
// must classify as Manual or Ambiguous, never Redirected.
func TestNewConfigNoManagers(t *testing.T) {
	cfg := newConfig("0.15.1")
	if cfg.Managers != nil {
		t.Errorf("Managers = %v, want nil (no package manager publishes this CLI)", cfg.Managers)
	}
}

// TestNewConfigSupportedPlatforms pins the exact darwin/linux/windows x
// amd64/arm64 matrix .goreleaser.yml publishes, EXCLUDING windows/arm64
// (explicitly `ignore`d there) — a host outside this set must be refused by
// the library's own unsupported-platform rule rather than attempting a swap
// synchestra has no asset for.
func TestNewConfigSupportedPlatforms(t *testing.T) {
	cfg := newConfig("0.15.1")
	want := map[selfupdate.Platform]bool{
		{GOOS: "darwin", GOARCH: "amd64"}:  true,
		{GOOS: "darwin", GOARCH: "arm64"}:  true,
		{GOOS: "linux", GOARCH: "amd64"}:   true,
		{GOOS: "linux", GOARCH: "arm64"}:   true,
		{GOOS: "windows", GOARCH: "amd64"}: true,
	}
	if len(cfg.SupportedPlatforms) != len(want) {
		t.Fatalf("SupportedPlatforms = %v, want exactly %d entries", cfg.SupportedPlatforms, len(want))
	}
	for _, p := range cfg.SupportedPlatforms {
		if !want[p] {
			t.Errorf("unexpected platform %+v in SupportedPlatforms", p)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Errorf("SupportedPlatforms is missing %v", want)
	}
	if slices.Contains(cfg.SupportedPlatforms, selfupdate.Platform{GOOS: "windows", GOARCH: "arm64"}) {
		t.Error("SupportedPlatforms must not contain windows/arm64 — .goreleaser.yml explicitly ignores that combination")
	}
}

// TestNewConfigVersionProbeArgs pins that the post-swap probe invokes the
// "version" subcommand, not the library's own "--version" default: this
// CLI's root --version flag is fang's own formatted output, not a bare
// version string, while `synchestra version` is guaranteed to contain it.
func TestNewConfigVersionProbeArgs(t *testing.T) {
	cfg := newConfig("0.15.1")
	want := []string{"version"}
	if len(cfg.VersionProbeArgs) != len(want) || cfg.VersionProbeArgs[0] != want[0] {
		t.Errorf("VersionProbeArgs = %v, want %v", cfg.VersionProbeArgs, want)
	}
}

// TestNewConfigDefaultAssetNaming pins that synchestra's asset naming must
// match .goreleaser.yml (synchestra_<version>_<os>_<arch>.tar.gz/.zip and
// synchestra_<version>_checksums.txt) by NOT overriding AssetName,
// ChecksumsName, or DownloadURL: the library's own GoReleaser-shaped
// defaults already produce those names, confirmed against a real published
// release (gh release view cli-v0.15.1 --repo synchestra-io/synchestra-releases).
// An override here would be a silent, needless divergence — and, unlike the
// synchestra-channel and synchestra-vm-host siblings, none is needed because
// the publish-releases job uploads dist/*.tar.gz, dist/*.zip, and
// dist/synchestra_*_checksums.txt unchanged, with no flattening/renaming
// step.
func TestNewConfigDefaultAssetNaming(t *testing.T) {
	cfg := newConfig("0.15.1")
	if cfg.AssetName != nil {
		t.Error("AssetName is overridden; synchestra's naming must match the library's GoReleaser-shaped default")
	}
	if cfg.ChecksumsName != nil {
		t.Error("ChecksumsName is overridden; synchestra's naming must match the library's GoReleaser-shaped default")
	}
	if cfg.DownloadURL != nil {
		t.Error("DownloadURL is overridden; synchestra's naming must match the library's GoReleaser-shaped default")
	}
}

// TestCommandRegistration pins REQ: command-and-alias: the command is named
// "self-update" and answers to the "update" alias, with --check,
// --format json (JSONFormat), --version, --allow-downgrade, and --dry-run
// all present (registered by cobracmd.New, not reimplemented here).
func TestCommandRegistration(t *testing.T) {
	cmd := Command("0.15.1")

	if cmd.Use != "self-update" {
		t.Errorf("Use = %q, want %q", cmd.Use, "self-update")
	}
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "update" {
		t.Errorf("Aliases = %v, want [update]", cmd.Aliases)
	}
	for _, flag := range []string{"check", "yes", "version", "allow-downgrade", "dry-run", "format"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag %q is not registered", flag)
		}
	}
}

// TestErrorMapperFailureInvalidArgs pins the two FailureKinds fixed by
// passing a different flag: KindDowngrade (--allow-downgrade) and
// KindNonInteractive (--yes) both map to exitcode.InvalidArgs (2).
func TestErrorMapperFailureInvalidArgs(t *testing.T) {
	cases := []struct {
		name string
		kind selfupdate.FailureKind
	}{
		{"downgrade", selfupdate.KindDowngrade},
		{"non-interactive", selfupdate.KindNonInteractive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &selfupdate.Failure{Kind: tc.kind, Err: errors.New("boom")}
			mapped := errorMapper{}.Failure(err)
			assertExitCode(t, mapped, exitcode.InvalidArgs)
		})
	}
}

// TestErrorMapperFailureNotFound pins the two FailureKinds meaning "the
// release asset this operation needs does not exist": KindUnknownTag (a bad
// --version pin) and KindUnsupportedPlatform (no asset published for this
// host at all) both map to exitcode.NotFound (3).
func TestErrorMapperFailureNotFound(t *testing.T) {
	cases := []struct {
		name string
		kind selfupdate.FailureKind
	}{
		{"unknown tag", selfupdate.KindUnknownTag},
		{"unsupported platform", selfupdate.KindUnsupportedPlatform},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &selfupdate.Failure{Kind: tc.kind, Err: errors.New("boom")}
			mapped := errorMapper{}.Failure(err)
			assertExitCode(t, mapped, exitcode.NotFound)
		})
	}
}

// TestErrorMapperFailureInvalidState pins KindAmbiguous — the current
// install's location doesn't match any recognized pattern, so the
// self-replace transition is refused given that state — mapping to
// exitcode.InvalidState (4).
func TestErrorMapperFailureInvalidState(t *testing.T) {
	err := &selfupdate.Failure{Kind: selfupdate.KindAmbiguous, Path: "/opt/synchestra", Err: errors.New("ambiguous")}
	mapped := errorMapper{}.Failure(err)
	assertExitCode(t, mapped, exitcode.InvalidState)
}

// TestErrorMapperFailureUnexpected pins every remaining FailureKind —
// release lookup, download, checksum, permission, and the library's own
// catch-all — plus a plain, non-*selfupdate.Failure error, onto
// exitcode.Unexpected (10): none of these are fixed by a different flag and
// none name a missing resource or a blocked state transition of their own.
func TestErrorMapperFailureUnexpected(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"release lookup", &selfupdate.Failure{Kind: selfupdate.KindReleaseLookup, Err: errors.New("network unreachable")}},
		{"download", &selfupdate.Failure{Kind: selfupdate.KindDownload, Err: errors.New("404")}},
		{"checksum", &selfupdate.Failure{Kind: selfupdate.KindChecksum, Err: errors.New("mismatch")}},
		{"permission", &selfupdate.Failure{Kind: selfupdate.KindPermission, Path: "/usr/local/bin/synchestra", Err: errors.New("denied")}},
		{"unexpected", &selfupdate.Failure{Kind: selfupdate.KindUnexpected, Err: errors.New("boom")}},
		{"plain error", errors.New("some other failure")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapped := errorMapper{}.Failure(tc.err)
			assertExitCode(t, mapped, exitcode.Unexpected)
		})
	}
}

// TestErrorMapperFailureNewKinds pins REQ: host-owned-exit-codes's explicit
// mapping requirement for the three FailureKinds the Install Command
// Library (cli-install, package cliinstall) appended after
// KindManagedCommand: KindUnknownTarget (a named install target is not a
// catalog id — fixed by passing a valid one, the same "missing or invalid
// command arguments" shape as KindDowngrade/KindNonInteractive) maps to
// InvalidArgs (2); KindNoInstallDir and KindDestinationExists (no usable
// destination directory, or one already occupied — the same blocked-
// transition-given-current-state shape as KindAmbiguous) both map to
// InvalidState (4). None of the three may fall into the Unexpected
// catch-all, even though this package's own `install` command does not
// exist yet — see errorMapper.Failure's doc comment.
func TestErrorMapperFailureNewKinds(t *testing.T) {
	cases := []struct {
		name string
		kind selfupdate.FailureKind
		want int
	}{
		{"unknown target", selfupdate.KindUnknownTarget, exitcode.InvalidArgs},
		{"no install dir", selfupdate.KindNoInstallDir, exitcode.InvalidState},
		{"destination exists", selfupdate.KindDestinationExists, exitcode.InvalidState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &selfupdate.Failure{Kind: tc.kind, Err: errors.New("boom")}
			mapped := errorMapper{}.Failure(err)
			assertExitCode(t, mapped, tc.want)
		})
	}
}

// TestErrorMapperUpdateAvailable pins REQ: exit-code-mapping's second half:
// both non-UpToDate --check verdicts (an available update, and a version too
// undetermined to compare) map to exitcode.Conflict (1) — the same code
// pkg/cli/spec/lint.go already uses for "this read-only inspection command
// found something to report" — and the message names both the current and
// latest versions.
func TestErrorMapperUpdateAvailable(t *testing.T) {
	cases := []selfupdate.CheckResult{
		{Current: "0.15.0", Latest: "0.15.1", Verdict: selfupdate.UpdateAvailable},
		{Current: "dev", Latest: "0.15.1", Verdict: selfupdate.Undetermined},
	}
	for _, result := range cases {
		t.Run(result.Verdict.String(), func(t *testing.T) {
			mapped := errorMapper{}.UpdateAvailable(result)
			assertExitCode(t, mapped, exitcode.Conflict)
			if got := mapped.Error(); !strings.Contains(got, result.Current) || !strings.Contains(got, result.Latest) {
				t.Errorf("message %q does not name both current (%q) and latest (%q)", got, result.Current, result.Latest)
			}
		})
	}
}

// assertExitCode fails the test unless err is (or wraps) an
// exitcode.Error carrying want.
func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	var coded *exitcode.Error
	if !errors.As(err, &coded) {
		t.Fatalf("error %v does not carry an exitcode.Error", err)
	}
	if coded.ExitCode() != want {
		t.Errorf("ExitCode() = %d, want %d", coded.ExitCode(), want)
	}
}
