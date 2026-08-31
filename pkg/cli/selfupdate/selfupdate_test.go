package selfupdate

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/strongo/selfupdate"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// TestNewConfigIdentity pins synchestra's own release identity: the release
// repository is the PUBLIC mirror (not this source repo, which publishes no
// GitHub Release of its own), gated by the "cli-" tag prefix that separates
// this CLI's releases from the other Synchestra products the same mirror
// carries.
func TestNewConfigIdentity(t *testing.T) {
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
// "dev", buildinfo.Get's own final fallback. A real release build always has
// its -X stamps set, so the intermediate runtime/debug.ReadBuildInfo() vcs.*
// fallback (which could in principle surface the Go toolchain's own
// "(devel)" source-tree placeholder) is never exercised there, and
// "(devel)" must NOT be declared here — an undeclared placeholder that can
// occur compares as a real version, but a declared one that cannot occur in
// a real release build is simply wrong.
func TestNewConfigUndeterminedVersions(t *testing.T) {
	cfg := newConfig("dev")

	if !slices.Contains(cfg.UndeterminedVersions, "dev") {
		t.Errorf("UndeterminedVersions = %v, missing %q", cfg.UndeterminedVersions, "dev")
	}
	if slices.Contains(cfg.UndeterminedVersions, "(devel)") {
		t.Error(`UndeterminedVersions must not contain "(devel)": a real synchestra release build always has buildinfo's -X stamps set`)
	}
	// A Go pseudo-version is a KNOWN version that sorts below its eventual
	// release, so it must not be swept into the undetermined set.
	if slices.Contains(cfg.UndeterminedVersions, "v0.15.2-0.20260809071100-889b6d621f76") {
		t.Error("a Go pseudo-version must not be treated as undetermined")
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
