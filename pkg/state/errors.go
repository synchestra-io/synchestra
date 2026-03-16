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
)
