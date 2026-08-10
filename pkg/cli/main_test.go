package cli

// Features implemented: cli/self-update

import "testing"

func TestRootCmdDoesNotAdvertiseWithdrawnStateStubs(t *testing.T) {
	root := newRootCmd(nil, nil)
	if found, _, err := root.Find([]string{"state"}); err == nil && found.Name() == "state" {
		t.Fatal("withdrawn state stubs must not be registered or advertised")
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
