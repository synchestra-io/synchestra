// Package install wires the "synchestra install" command onto the shared
// github.com/strongo/cli-helpers/cliinstall module (via its Cobra adapter,
// cliinstall/cobracmd). It implements none of the listing, planning or
// download logic itself: the relevance matrix, status probing, destination
// policy, Homebrew-cask decision, checksum-verified download and the
// confirmation gate all belong to that library. This package supplies only
// what is genuinely synchestra's own — its catalog id
// (cli-install#req:host-identity-from-catalog) and the mapping from the
// library's outcomes onto synchestra's own exit-code contract (documented
// in spec/features/cli/README.md's "Exit code contract" table), mirroring
// pkg/cli/selfupdate's own errorMapper exactly. See
// spec/features/cli/install/README.md for the Feature spec that draws this
// boundary.
package install

// Features implemented: cli/install

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/strongo/cli-helpers/cliinstall/cobracmd"
	"github.com/strongo/cli-helpers/selfupdate"
	"github.com/synchestra-io/specscore/pkg/exitcode"
)

// catalogID is synchestra's own id in the Install Command Library's
// compiled-in catalog (cli-install#req:host-identity-from-catalog) — the
// same id pkg/cli/selfupdate uses. An id absent from the catalog is a
// programming error cobracmd.New itself panics on, caught by
// TestCommandRegistration, never a runtime state a user sees.
const catalogID = "synchestra"

// Command returns the "install" command. Every decision it makes —
// relevance listing, status probing, destination policy, download and
// verification — comes from cobracmd.New and the compiled-in catalog entry
// for catalogID. This function's only job is to name synchestra to that
// library and to translate its outcomes onto synchestra's own exit-code
// contract via errorMapper.
func Command() *cobra.Command {
	return cobracmd.New(cobracmd.CommandOptions{
		Short:  "List and install fleet CLIs relevant to synchestra",
		Errors: errorMapper{},
		HostID: catalogID,
	})
}

// errorMapper maps github.com/strongo/cli-helpers/cliinstall's outcomes
// onto synchestra's own exit-code contract, documented in
// spec/features/cli/README.md's "Exit code contract" table:
//
//	0  Success
//	1  Conflict     — concurrent-modification conflict
//	2  InvalidArgs  — missing or invalid command arguments/flags
//	3  NotFound     — requested resource does not exist
//	4  InvalidState — state transition is not allowed
//	10 Unexpected   — catch-all runtime error
//
// This mirrors pkg/cli/selfupdate's errorMapper.Failure kind-for-kind
// (cli-install#req:host-owned-exit-codes: hosts "keep their self-update
// mapping for the shared kinds"), so a failure kind synchestra's
// self-update and install commands both can produce always yields the same
// exit code, and adds the two new kinds cli-install requires every host to
// map explicitly:
//
//   - KindUnknownTarget: a named install target is not a catalog id — fixed
//     by passing a valid one, the same "missing or invalid command
//     arguments" shape self-update already uses for KindDowngrade and
//     KindNonInteractive. Maps to InvalidArgs.
//   - KindNoInstallDir, KindDestinationExists: no usable destination
//     directory, or one already occupied — the same blocked-transition-
//     given-current-state shape self-update already uses for KindAmbiguous.
//     Both map to InvalidState.
//
// A *cobracmd.UsageError (an invalid --format, or --all combined with
// names) is also a flag mistake, so it maps to InvalidArgs alongside
// KindUnknownTarget.
type errorMapper struct{}

// Failure maps a command error onto the standard code whose documented
// meaning it actually matches, with an "install: " message prefix and no
// "self-update: " prefix (cli-install#req:host-owned-exit-codes).
//
// cliinstall/cobracmd v0.21.0's own mapFailure short-circuits a nil error
// before ever calling opts.Errors.Failure (see that package's doc comment
// on mapFailure and its TestMapFailure_NeverCallsMapperWithNil), so the
// v0.19.0-era nil-guard this method used to carry is gone.
func (errorMapper) Failure(err error) error {
	msg := fmt.Sprintf("install: %v", err)

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
