package install

import (
	"errors"
	"testing"

	"github.com/strongo/cli-helpers/cliinstall"
	"github.com/strongo/cli-helpers/cliinstall/cobracmd"
	"github.com/strongo/cli-helpers/selfupdate"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// TestCatalogHasSynchestra pins REQ: host-id: an id absent from the
// compiled catalog is a programming error cobracmd.New itself panics on
// (cli-install#req:host-identity-from-catalog) — this test is what
// actually catches that before Command() ever runs.
func TestCatalogHasSynchestra(t *testing.T) {
	if _, ok := cliinstall.ByID(catalogID); !ok {
		t.Fatalf("cliinstall has no catalog entry for id %q", catalogID)
	}
}

// TestCommandRegistration pins REQ: command-name: the command is named
// "install", inherits the library's full flag surface (--all, --yes/-y,
// --dry-run, --dir, --format), and carries no "update" alias — only
// self-update keeps that alias (cli-install#req:update-alias-policy).
func TestCommandRegistration(t *testing.T) {
	cmd := Command()

	if cmd.Use != "install [name...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "install [name...]")
	}
	if len(cmd.Aliases) != 0 {
		t.Errorf("Aliases = %v, want none", cmd.Aliases)
	}
	for _, flag := range []string{"all", "yes", "dry-run", "dir", "format"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag %q is not registered", flag)
		}
	}
}

// TestCommandRegistrationPanicsWhenCatalogEntryMissing pins that
// cobracmd.New's own panic is what protects
// cli-install#req:host-identity-from-catalog, exercised through this
// package's own HostID plumbing rather than trusting the library alone.
func TestCommandRegistrationPanicsWhenCatalogEntryMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unknown catalog id")
		}
	}()
	cobracmd.New(cobracmd.CommandOptions{HostID: "nosuchcli"})
}

// TestErrorMapperFailureNilIsSuccess pins the documented, reachable nil
// path: cliinstall/cobracmd v0.19.0 calls opts.Errors.Failure(nil)
// unconditionally on every successful `install` and `install --dry-run`
// run (see errorMapper.Failure's doc comment) — a known library bug fixed
// in the next release. Failure(nil) MUST return nil, not a wrapped
// "install: <nil>" error, so a fully successful batch never fails the
// command.
func TestErrorMapperFailureNilIsSuccess(t *testing.T) {
	if err := (errorMapper{}).Failure(nil); err != nil {
		t.Errorf("Failure(nil) = %v, want nil", err)
	}
}

// TestErrorMapperFailureUsageError pins that a *cobracmd.UsageError (an
// invalid --format, or --all combined with names) maps to InvalidArgs (2),
// the same code KindUnknownTarget maps to below — both are flag/argument
// mistakes fixed by passing something different.
func TestErrorMapperFailureUsageError(t *testing.T) {
	err := &cobracmd.UsageError{Err: errors.New("invalid --format")}
	mapped := (errorMapper{}).Failure(err)
	assertExitCode(t, mapped, exitcode.InvalidArgs)
}

// TestErrorMapperFailureInvalidArgs pins the two self-update-shared
// FailureKinds fixed by passing a different flag: KindDowngrade and
// KindNonInteractive both map to exitcode.InvalidArgs (2), matching
// pkg/cli/selfupdate's own errorMapper kind-for-kind.
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
			mapped := (errorMapper{}).Failure(err)
			assertExitCode(t, mapped, exitcode.InvalidArgs)
		})
	}
}

// TestErrorMapperFailureNotFound pins the two self-update-shared
// FailureKinds meaning "the release asset this operation needs does not
// exist": KindUnknownTag and KindUnsupportedPlatform both map to
// exitcode.NotFound (3).
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
			mapped := (errorMapper{}).Failure(err)
			assertExitCode(t, mapped, exitcode.NotFound)
		})
	}
}

// TestErrorMapperFailureAmbiguousIsInvalidState pins KindAmbiguous mapping
// to exitcode.InvalidState (4), matching pkg/cli/selfupdate's own
// errorMapper.
func TestErrorMapperFailureAmbiguousIsInvalidState(t *testing.T) {
	err := &selfupdate.Failure{Kind: selfupdate.KindAmbiguous, Path: "/opt/synchestra", Err: errors.New("ambiguous")}
	mapped := (errorMapper{}).Failure(err)
	assertExitCode(t, mapped, exitcode.InvalidState)
}

// TestErrorMapperFailureNewKinds pins REQ: exit-codes's explicit mapping
// requirement for the three FailureKinds the Install Command Library
// (cli-install, package cliinstall) appended after KindManagedCommand:
// KindUnknownTarget maps to InvalidArgs (2); KindNoInstallDir and
// KindDestinationExists both map to InvalidState (4). None of the three
// falls into the Unexpected catch-all
// (cli-install#req:host-owned-exit-codes).
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
			mapped := (errorMapper{}).Failure(err)
			assertExitCode(t, mapped, tc.want)
		})
	}
}

// TestErrorMapperFailureUnexpected pins every remaining self-update-shared
// FailureKind — release lookup, download, checksum, permission, managed
// command, and the library's own catch-all — plus a plain, non-
// *selfupdate.Failure error, onto exitcode.Unexpected (10).
func TestErrorMapperFailureUnexpected(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"release lookup", &selfupdate.Failure{Kind: selfupdate.KindReleaseLookup, Err: errors.New("network unreachable")}},
		{"download", &selfupdate.Failure{Kind: selfupdate.KindDownload, Err: errors.New("404")}},
		{"checksum", &selfupdate.Failure{Kind: selfupdate.KindChecksum, Err: errors.New("mismatch")}},
		{"permission", &selfupdate.Failure{Kind: selfupdate.KindPermission, Path: "/usr/local/bin/ingitdb", Err: errors.New("denied")}},
		{"managed command", &selfupdate.Failure{Kind: selfupdate.KindManagedCommand, Err: errors.New("brew exited 1")}},
		{"unexpected", &selfupdate.Failure{Kind: selfupdate.KindUnexpected, Err: errors.New("boom")}},
		{"plain error", errors.New("some other failure")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapped := (errorMapper{}).Failure(tc.err)
			assertExitCode(t, mapped, exitcode.Unexpected)
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
