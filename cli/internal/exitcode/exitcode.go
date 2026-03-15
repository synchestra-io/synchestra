// Package exitcode defines an error type that carries a process exit code.
package exitcode

// Features implemented: cli/project/new

import "fmt"

// Error is an error that carries a specific process exit code.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// New returns an *Error with the given exit code and formatted message.
func New(code int, format string, args ...any) *Error {
	return &Error{Code: code, Err: fmt.Errorf(format, args...)}
}
