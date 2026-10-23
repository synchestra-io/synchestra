package cli

// Features implemented: cli/self-update, cli/install

import (
	"errors"
	"io"
	"testing"
)

// TestRootCmdAdvertisesStateWaitButNotTheWithdrawnSyncStubs pins the current
// "state" command group's real, intentional shape: `synchestra state wait`
// (state-store/topology's mirror barrier) is real and reachable, while
// `pull`/`push`/`sync` remain the withdrawn stubs from
// "fix(state): harden physical journal boundaries" -- unregistered because
// they print "not implemented yet" and always fail, not because "state" as a
// whole is off-limits. See pkg/cli/state/state.go's Command() doc comment.
func TestRootCmdAdvertisesStateWaitButNotTheWithdrawnSyncStubs(t *testing.T) {
	root := newRootCmd(nil, nil)

	found, _, err := root.Find([]string{"state", "wait"})
	if err != nil {
		t.Fatalf("synchestra state wait: %v", err)
	}
	if found.Name() != "wait" {
		t.Errorf("synchestra state wait resolved to %q", found.Name())
	}

	for _, stub := range []string{"pull", "push", "sync"} {
		found, _, err := root.Find([]string{"state", stub})
		if err == nil && found.Name() == stub {
			t.Errorf("withdrawn state stub %q must not be registered or advertised", stub)
		}
	}
}

// TestRootCmdRegistersSelfUpdate pins that "self-update" is actually wired
// into the root command tree, and that both its canonical name and its
// "update" alias resolve there — the alias resolution `synchestra update`
// relies on happens through cobra.Command.Find, not a second registration.
func TestRootCmdRegistersSelfUpdate(t *testing.T) {
	root := newRootCmd(nil, nil)

	found, _, err := root.Find([]string{"self-update"})
	if err != nil {
		t.Fatalf("synchestra self-update: %v", err)
	}
	if found.Name() != "self-update" {
		t.Errorf("synchestra self-update resolved to %q", found.Name())
	}

	foundAlias, _, err := root.Find([]string{"update"})
	if err != nil {
		t.Fatalf("synchestra update: %v", err)
	}
	if foundAlias.Name() != "self-update" {
		t.Errorf("synchestra update resolved to %q, want the self-update command", foundAlias.Name())
	}
}

// TestRootCmdRegistersInstall pins that "install" is wired into the root
// command tree, distinct from "self-update", and carries no "update" alias
// of its own (cli-install#req:update-alias-policy).
func TestRootCmdRegistersInstall(t *testing.T) {
	root := newRootCmd(nil, nil)

	found, _, err := root.Find([]string{"install"})
	if err != nil {
		t.Fatalf("synchestra install: %v", err)
	}
	if found.Name() != "install" {
		t.Errorf("synchestra install resolved to %q", found.Name())
	}
	if len(found.Aliases) != 0 {
		t.Errorf("install Aliases = %v, want none", found.Aliases)
	}
}

// TestSynchestraInstallNosuchcliExitsInvalidArgs runs the real command tree
// end-to-end, offline (cli-install#req:unknown-target-refused,
// cli-install#req:no-network-in-tests): an unrecognized target name MUST
// fail before any confirmation, network request or write, with exit code 2
// (InvalidArgs) — pkg/cli/install's own errorMapper mapping of
// selfupdate.KindUnknownTarget, exercised through the full Cobra tree
// rather than only the unit-level errorMapper.Failure tests in
// pkg/cli/install/install_test.go.
func TestSynchestraInstallNosuchcliExitsInvalidArgs(t *testing.T) {
	root := newRootCmd(nil, nil)
	root.SetArgs([]string{"install", "nosuchcli"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil {
		t.Fatal("synchestra install nosuchcli: expected an error, got nil")
	}

	type exitCoder interface{ ExitCode() int }
	var ec exitCoder
	if !errors.As(err, &ec) {
		t.Fatalf("synchestra install nosuchcli: error %v does not carry an ExitCode()", err)
	}
	if got, want := ec.ExitCode(), 2; got != want {
		t.Errorf("synchestra install nosuchcli: ExitCode() = %d, want %d", got, want)
	}
}
