package state

// Features implemented: state-store

import "errors"

var (
	// ErrNotFound is returned when a task, chat, or artifact does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInvalidTransition is returned when a status transition is not allowed
	// from the current state (e.g., calling Complete on a queued task).
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrConflict is returned when a concurrent modification prevents the
	// operation (e.g., two agents claiming the same task).
	ErrConflict = errors.New("conflict")

	// ErrLeaseFenced is returned when a caller's fence token is no longer
	// current — e.g. after promotion incremented the authority epoch, or
	// after an explicit release — preserving the topology fencing guarantee
	// (state-store/topology#ac:promotion-fences-former-active) at the domain
	// layer.
	ErrLeaseFenced = errors.New("lease fenced")

	// ErrInvalidArgument is returned when a caller-supplied value is
	// malformed or outside the domain's accepted vocabulary/range, and the
	// problem is not itself a lifecycle transition (state.ErrInvalidTransition
	// covers those) — e.g. an unrecognized state.MessageKind, or a negative
	// lease TTL. CLI wiring (pkg/cli/agent/resolve.go's mapStoreError) maps
	// this to exitcode.InvalidArgs, matching how a bad flag caught before
	// ever reaching the store is reported.
	ErrInvalidArgument = errors.New("invalid argument")
)
