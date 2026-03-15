package internal

// Features depended on: cli

import "fmt"

// ExitError wraps an error with an exit code for CLI handling.
type ExitError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewExitError creates a new ExitError with a message.
func NewExitError(code int, message string) *ExitError {
	return &ExitError{Code: code, Message: message}
}

// NewExitErrorf creates a new ExitError with formatted message.
func NewExitErrorf(code int, format string, args ...interface{}) *ExitError {
	return &ExitError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WrapExitError wraps an existing error with an exit code.
func WrapExitError(code int, message string, err error) *ExitError {
	return &ExitError{Code: code, Message: message, Err: err}
}
