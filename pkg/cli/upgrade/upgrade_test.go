package upgrade

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/strongo/cli-helpers/cliinstall"
	"github.com/strongo/cli-helpers/cliinstall/cobracmd"
	"github.com/strongo/cli-helpers/selfupdate"
	selfupdatecobracmd "github.com/strongo/cli-helpers/selfupdate/cobracmd"
	"github.com/synchestra-io/specscore/pkg/exitcode"

	cliselfupdate "github.com/synchestra-io/synchestra/pkg/cli/selfupdate"
)

// TestCatalogHasSynchestra pins REQ: host-id, mirroring
// pkg/cli/install's own TestCatalogHasSynchestra: an id absent from the
// compiled catalog is a programming error caught here before Command() ever
// runs (cli-install#req:host-identity-from-catalog).
func TestCatalogHasSynchestra(t *testing.T) {
	if _, ok := cliinstall.ByID(catalogID); !ok {
		t.Fatalf("cliinstall has no catalog entry for id %q", catalogID)
	}
}

// TestCommandRegistration pins the command name, flag surface, and the
// absence of any "update" alias (cli-install#req:update-alias-policy: that
// alias stays reserved for self-update alone).
func TestCommandRegistration(t *testing.T) {
	cmd := Command("0.15.1")

	if !strings.HasPrefix(cmd.Use, "upgrade") {
		t.Errorf("Use = %q, want it to start with upgrade", cmd.Use)
	}
	if cmd.HasAlias("update") {
		t.Error(`upgrade must not alias "update"; that stays reserved for self-update (cli-install#req:update-alias-policy)`)
	}
	for _, flag := range []string{"all", "check", "yes", "dry-run", "format"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag %q is not registered", flag)
		}
	}
	if cmd.Flags().Lookup("dir") != nil {
		t.Error("unexpected --dir flag; upgrade has no --dir")
	}
}

func TestCommandRegistrationPanicsWhenCatalogEntryMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unknown catalog id")
		}
	}()
	cobracmd.NewUpgrade(cobracmd.UpgradeCommandOptions{HostID: "nosuchcli"})
}

// TestErrorMapperFailureUsageError pins that a *cobracmd.UsageError maps to
// InvalidArgs (2), matching pkg/cli/install's own mapping.
func TestErrorMapperFailureUsageError(t *testing.T) {
	err := &cobracmd.UsageError{Err: errors.New("invalid --format")}
	mapped := (errorMapper{}).Failure(err)
	assertExitCode(t, mapped, exitcode.InvalidArgs)
}

// TestErrorMapperFailureInvalidArgs pins the two self-update-shared
// FailureKinds fixed by passing a different flag, matching pkg/cli/
// selfupdate's and pkg/cli/install's own errorMapper kind-for-kind.
func TestErrorMapperFailureInvalidArgs(t *testing.T) {
	cases := []selfupdate.FailureKind{selfupdate.KindDowngrade, selfupdate.KindNonInteractive}
	for _, kind := range cases {
		err := &selfupdate.Failure{Kind: kind, Err: errors.New("boom")}
		mapped := (errorMapper{}).Failure(err)
		assertExitCode(t, mapped, exitcode.InvalidArgs)
	}
}

func TestErrorMapperFailureNotFound(t *testing.T) {
	cases := []selfupdate.FailureKind{selfupdate.KindUnknownTag, selfupdate.KindUnsupportedPlatform}
	for _, kind := range cases {
		err := &selfupdate.Failure{Kind: kind, Err: errors.New("boom")}
		mapped := (errorMapper{}).Failure(err)
		assertExitCode(t, mapped, exitcode.NotFound)
	}
}

func TestErrorMapperFailureInvalidState(t *testing.T) {
	err := &selfupdate.Failure{Kind: selfupdate.KindAmbiguous, Err: errors.New("ambiguous")}
	mapped := (errorMapper{}).Failure(err)
	assertExitCode(t, mapped, exitcode.InvalidState)
}

// TestErrorMapperFailureNewKinds pins REQ: host-owned-exit-codes's explicit
// mapping requirement for the three cli-install-only kinds, matching
// pkg/cli/install's own mapping kind-for-kind.
func TestErrorMapperFailureNewKinds(t *testing.T) {
	cases := []struct {
		kind selfupdate.FailureKind
		want int
	}{
		{selfupdate.KindUnknownTarget, exitcode.InvalidArgs},
		{selfupdate.KindNoInstallDir, exitcode.InvalidState},
		{selfupdate.KindDestinationExists, exitcode.InvalidState},
	}
	for _, tc := range cases {
		err := &selfupdate.Failure{Kind: tc.kind, Err: errors.New("boom")}
		mapped := (errorMapper{}).Failure(err)
		assertExitCode(t, mapped, tc.want)
	}
}

func TestErrorMapperFailureUnexpected(t *testing.T) {
	cases := []error{
		&selfupdate.Failure{Kind: selfupdate.KindReleaseLookup, Err: errors.New("network unreachable")},
		&selfupdate.Failure{Kind: selfupdate.KindDownload, Err: errors.New("404")},
		&selfupdate.Failure{Kind: selfupdate.KindChecksum, Err: errors.New("mismatch")},
		&selfupdate.Failure{Kind: selfupdate.KindPermission, Err: errors.New("denied")},
		&selfupdate.Failure{Kind: selfupdate.KindManagedCommand, Err: errors.New("brew failed")},
		&selfupdate.Failure{Kind: selfupdate.KindUnexpected, Err: errors.New("boom")},
		errors.New("plain error"),
	}
	for _, err := range cases {
		mapped := (errorMapper{}).Failure(err)
		assertExitCode(t, mapped, exitcode.Unexpected)
	}
}

// TestErrorMapperUpgradesAvailable_SingleTargetMatchesSelfUpdate proves
// UpgradesAvailable's single-target case is exit-code and message
// equivalent to pkg/cli/selfupdate's own UpdateAvailable — the case that
// matters for `upgrade synchestra --check` vs. `self-update --check`
// (cli-install#req:self-update-equals-upgrade-self).
func TestErrorMapperUpgradesAvailable_SingleTargetMatchesSelfUpdate(t *testing.T) {
	cases := []struct {
		name   string
		result cliinstall.UpgradeResult
	}{
		{"update available", cliinstall.UpgradeResult{Target: "synchestra", Current: "0.15.0", Latest: "0.15.1", Verdict: selfupdate.UpdateAvailable}},
		{"undetermined", cliinstall.UpgradeResult{Target: "synchestra", Current: "dev", Latest: "0.15.1", Verdict: selfupdate.Undetermined}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mapped := (errorMapper{}).UpgradesAvailable([]cliinstall.UpgradeResult{c.result})
			assertExitCode(t, mapped, exitcode.Conflict)
			if got := mapped.Error(); !strings.Contains(got, c.result.Current) || !strings.Contains(got, c.result.Latest) {
				t.Errorf("message %q does not name both current (%q) and latest (%q)", got, c.result.Current, c.result.Latest)
			}
		})
	}
}

func TestErrorMapperUpgradesAvailable_MultiTarget(t *testing.T) {
	results := []cliinstall.UpgradeResult{
		{Target: "datatug", Current: "1.0.0", Latest: "1.1.0", Verdict: selfupdate.UpdateAvailable},
		{Target: "ingitdb", Current: "dev", Latest: "1.1.0", Verdict: selfupdate.Undetermined},
	}
	mapped := (errorMapper{}).UpgradesAvailable(results)
	assertExitCode(t, mapped, exitcode.Conflict)
	if got := mapped.Error(); !strings.Contains(got, "datatug") || !strings.Contains(got, "ingitdb") {
		t.Errorf("message %q does not name both targets", got)
	}
}

// assertExitCode fails the test unless err is (or wraps) an exitcode.Error
// carrying want, mirroring pkg/cli/selfupdate's and pkg/cli/install's own
// helper.
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

// --- end-to-end exit-code contract, fully offline ---

// `upgrade nosuchcli` never reaches a status probe or a release lookup:
// cliinstall.CheckUpgrades/PlanUpgrade validate every name against the
// catalog before probing anything, mirroring install's own unknown-target
// path, so this is inherently offline (REQ: no-network-in-tests).
func TestUpgradeCmd_NoSuchTarget_ExitCodeContract(t *testing.T) {
	cmd := Command("0.15.1")
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs([]string{"nosuchcli"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected a non-nil error for an unknown upgrade target")
	}
	if !strings.Contains(err.Error(), "nosuchcli") {
		t.Errorf("error %q does not name the unknown target", err.Error())
	}
	assertExitCode(t, err, exitcode.InvalidArgs)
}

func TestUpgradeCmd_InvalidFormat_IsUsageError(t *testing.T) {
	cmd := Command("0.15.1")
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs([]string{"--format", "yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected a non-nil error for --format yaml")
	}
	assertExitCode(t, err, exitcode.InvalidArgs)
}

// --- self-update ≡ upgrade synchestra: same Config by construction ---

// fakeUpgradeEnv is a fully hermetic cliinstall.InstallEnv: no real PATH
// scan, no real filesystem, no real process execution
// (REQ: no-network-in-tests). The host row's classification/version come
// from HostConfig, never from this Env, but resolveUpgradeCandidates still
// calls Probe over every named entry for diagnostics, so every field must
// be set to avoid a nil-func panic.
func fakeUpgradeEnv() cliinstall.InstallEnv {
	return cliinstall.InstallEnv{
		Env: cliinstall.Env{
			PathDirs:     func() []string { return nil },
			HostDir:      func() (string, error) { return "", errors.New("no host dir in test") },
			IsExecutable: func(string) bool { return false },
			EvalSymlinks: func(p string) (string, error) { return p, nil },
			Run: func(context.Context, string, []string) ([]byte, error) {
				return nil, errors.New("process execution disabled in test")
			},
		},
		UserHomeDir: func() (string, error) { return "", errors.New("disabled in test") },
		Getenv:      func(string) string { return "" },
		MkdirAll:    func(string, fs.FileMode) error { return errors.New("disabled in test") },
	}
}

func upgradeReleasesServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSelfUpdateEqualsUpgradeSelf_CheckContract proves `synchestra
// self-update --check` and `synchestra upgrade synchestra --check` reach
// the same exit-code contract for the same fake releases server, built from
// the exact same Config (cliselfupdate.NewConfig)
// (cli-install#req:self-update-equals-upgrade-self,
// cli-install#req:upgrade-check).
func TestSelfUpdateEqualsUpgradeSelf_CheckContract(t *testing.T) {
	cases := []struct {
		name          string
		ver           string
		body          string
		wantAvailable bool
	}{
		{name: "up to date", ver: "0.15.1", body: `[{"tag_name":"cli-v0.15.1","prerelease":false,"draft":false}]`, wantAvailable: false},
		{name: "update available", ver: "0.15.0", body: `[{"tag_name":"cli-v0.15.1","prerelease":false,"draft":false}]`, wantAvailable: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := upgradeReleasesServer(t, c.body, http.StatusOK)
			cfg := cliselfupdate.NewConfig(c.ver)
			cfg.ReleasesAPIURL = srv.URL
			cfg.HTTPClient = srv.Client()

			selfCmd := selfupdatecobracmd.New(cfg, selfupdatecobracmd.CommandOptions{JSONFormat: true, Errors: cliSelfUpdateErrors{}})
			selfCmd.SetOut(&strings.Builder{})
			selfCmd.SetErr(&strings.Builder{})
			selfCmd.SetArgs([]string{"--check"})
			selfErr := selfCmd.Execute()

			upCmd := cobracmd.NewUpgrade(cobracmd.UpgradeCommandOptions{
				HostID:     catalogID,
				Errors:     errorMapper{},
				HostConfig: cfg,
				Env:        fakeUpgradeEnv(),
			})
			upCmd.SetOut(&strings.Builder{})
			upCmd.SetErr(&strings.Builder{})
			upCmd.SetArgs([]string{"synchestra", "--check"})
			upErr := upCmd.Execute()

			if c.wantAvailable {
				assertExitCode(t, selfErr, exitcode.Conflict)
				assertExitCode(t, upErr, exitcode.Conflict)
				return
			}
			if selfErr != nil || upErr != nil {
				t.Fatalf("want both nil (up to date): self-update=%v upgrade=%v", selfErr, upErr)
			}
		})
	}

	t.Run("release lookup failure fails both the same way", func(t *testing.T) {
		srv := upgradeReleasesServer(t, `not json`, http.StatusInternalServerError)
		cfg := cliselfupdate.NewConfig("0.15.1")
		cfg.ReleasesAPIURL = srv.URL
		cfg.HTTPClient = srv.Client()

		selfCmd := selfupdatecobracmd.New(cfg, selfupdatecobracmd.CommandOptions{JSONFormat: true, Errors: cliSelfUpdateErrors{}})
		selfCmd.SetOut(&strings.Builder{})
		selfCmd.SetErr(&strings.Builder{})
		selfCmd.SetArgs([]string{"--check"})
		selfErr := selfCmd.Execute()
		if selfErr == nil {
			t.Fatal("expected self-update --check to fail on a release-lookup error")
		}
		assertExitCode(t, selfErr, exitcode.Unexpected)

		upCmd := cobracmd.NewUpgrade(cobracmd.UpgradeCommandOptions{
			HostID:     catalogID,
			Errors:     errorMapper{},
			HostConfig: cfg,
			Env:        fakeUpgradeEnv(),
		})
		upCmd.SetOut(&strings.Builder{})
		upCmd.SetErr(&strings.Builder{})
		upCmd.SetArgs([]string{"synchestra", "--check"})
		upErr := upCmd.Execute()
		if upErr == nil {
			t.Fatal("expected upgrade synchestra --check to fail the same way")
		}
		assertExitCode(t, upErr, exitcode.Unexpected)
	})
}

// cliSelfUpdateErrors reimplements pkg/cli/selfupdate's own unexported
// errorMapper.Failure/UpdateAvailable mapping (kind-for-kind identical,
// per that package's own doc comment) so this test can build a real
// selfupdate/cobracmd.New command without exporting selfupdate's own
// internal type. Both selfupdate's Failure and this method reach the exact
// same exitcode mapping.
type cliSelfUpdateErrors struct{}

func (cliSelfUpdateErrors) Failure(err error) error {
	switch selfupdate.KindOf(err) {
	case selfupdate.KindDowngrade, selfupdate.KindNonInteractive:
		return exitcode.InvalidArgsError(err.Error())
	case selfupdate.KindUnknownTag, selfupdate.KindUnsupportedPlatform:
		return exitcode.NotFoundError(err.Error())
	case selfupdate.KindAmbiguous:
		return exitcode.InvalidStateError(err.Error())
	default:
		return exitcode.UnexpectedError(err.Error())
	}
}

func (cliSelfUpdateErrors) UpdateAvailable(res selfupdate.CheckResult) error {
	return exitcode.ConflictErrorf("self-update: update available (%s -> %s)", res.Current, res.Latest)
}
