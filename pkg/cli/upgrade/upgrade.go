// Package upgrade wires the "synchestra upgrade" command onto the shared
// github.com/strongo/cli-helpers/cliinstall module (via its Cobra adapter,
// cliinstall/cobracmd.NewUpgrade). It implements none of the upgrade logic
// itself: target selection, release lookups, per-target policy, the
// confirmation gate and the underlying self-replace/manager-execute
// machinery all belong to that library and to
// github.com/strongo/cli-helpers/selfupdate. This package supplies only
// what is genuinely synchestra's own — its catalog id
// (cli-install#req:host-identity-from-catalog), the SAME HostConfig
// pkg/cli/selfupdate builds (so `synchestra self-update` and `synchestra
// upgrade synchestra` reach the exact same library call by construction,
// cli-install#req:self-update-equals-upgrade-self), and the mapping from
// the library's outcomes onto synchestra's own exit-code contract
// (documented in spec/features/cli/README.md's "Exit code contract"
// table), mirroring pkg/cli/selfupdate's and pkg/cli/install's own
// errorMapper kind-for-kind. See spec/features/cli/install/README.md for
// the Feature spec that draws this boundary.
package upgrade

// Features implemented: cli/install

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strongo/cli-helpers/cliinstall"
	"github.com/strongo/cli-helpers/cliinstall/cobracmd"
	"github.com/strongo/cli-helpers/selfupdate"
	"github.com/synchestra-io/specscore/pkg/exitcode"

	cliselfupdate "github.com/synchestra-io/synchestra/pkg/cli/selfupdate"
)

// catalogID is synchestra's own id in the Install Command Library's
// compiled-in catalog (cli-install#req:host-identity-from-catalog) — the
// same id pkg/cli/selfupdate and pkg/cli/install use. An id absent from the
// catalog is a programming error cobracmd.NewUpgrade itself panics on,
// caught by TestCommandRegistration, never a runtime state a user sees.
const catalogID = "synchestra"

// Command returns the "upgrade" command. Every decision it makes — target
// selection, release lookups, per-target policy, and the underlying
// self-replace/manager-execute machinery — comes from cobracmd.NewUpgrade
// and cliselfupdate.NewConfig(currentVersion), the exact same
// selfupdate.Config pkg/cli/selfupdate's own Command builds. synchestra has
// no after-update hook, so HostAfterUpdate is left unset, matching
// pkg/cli/selfupdate's own CommandOptions. No "update" alias: that alias
// stays reserved for self-update alone
// (cli-install#req:update-alias-policy).
func Command(currentVersion string) *cobra.Command {
	return cobracmd.NewUpgrade(cobracmd.UpgradeCommandOptions{
		Short:      "Upgrade installed fleet CLIs, including synchestra itself",
		Errors:     errorMapper{},
		HostID:     catalogID,
		HostConfig: cliselfupdate.NewConfig(currentVersion),
	})
}

// errorMapper maps github.com/strongo/cli-helpers/cliinstall's upgrade
// outcomes onto synchestra's own exit-code contract, documented in
// spec/features/cli/README.md's "Exit code contract" table:
//
//	0  Success
//	1  Conflict     — concurrent-modification conflict
//	2  InvalidArgs  — missing or invalid command arguments/flags
//	3  NotFound     — requested resource does not exist
//	4  InvalidState — state transition is not allowed
//	10 Unexpected   — catch-all runtime error
//
// Failure mirrors pkg/cli/selfupdate's and pkg/cli/install's own
// errorMapper.Failure kind-for-kind (cli-install#req:host-owned-exit-codes:
// hosts "keep their self-update mapping for the shared kinds"), so a
// failure kind self-update, install and upgrade can all produce always
// yields the same exit code, with an "upgrade: " message prefix instead of
// "self-update: "/"install: ".
type errorMapper struct{}

// Failure maps a command error onto the standard code whose documented
// meaning it actually matches, with an "upgrade: " message prefix
// (cli-install#req:host-owned-exit-codes).
func (errorMapper) Failure(err error) error {
	msg := fmt.Sprintf("upgrade: %v", err)

	var usage *cobracmd.UsageError
	if errors.As(err, &usage) {
		return exitcode.InvalidArgsError(msg)
	}

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
	default: // KindReleaseLookup, KindDownload, KindChecksum, KindPermission, KindManagedCommand, KindUnexpected
		return exitcode.UnexpectedError(msg)
	}
}

// UpgradesAvailable maps an available or undetermined upgrade onto
// exitcode.Conflict (1), exactly mirroring pkg/cli/selfupdate's own
// errorMapper.UpdateAvailable (cli-install#req:upgrade-check: "a host maps
// it as its self-update maps UpdateAvailable"). For the single-target case
// — in particular `upgrade synchestra --check`, matching `self-update
// --check` exactly — the message names current and latest the same way
// UpdateAvailable's own message does
// (cli-install#req:self-update-equals-upgrade-self).
func (errorMapper) UpgradesAvailable(results []cliinstall.UpgradeResult) error {
	if len(results) == 1 {
		r := results[0]
		if r.Verdict == selfupdate.Undetermined {
			return exitcode.ConflictErrorf("upgrade: current version is undetermined (%s); latest stable is %s", r.Current, r.Latest)
		}
		return exitcode.ConflictErrorf("upgrade: update available (%s -> %s)", r.Current, r.Latest)
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Target
	}
	return exitcode.ConflictErrorf("upgrade: upgrades available for %s", strings.Join(names, ", "))
}
