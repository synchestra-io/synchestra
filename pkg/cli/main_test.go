package cli

// Features implemented: cli/self-update

import "testing"

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
