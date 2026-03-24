package state

// Features implemented: cli/state

// exitError carries an exit code for CLI error handling.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }
func (e *exitError) ExitCode() int { return e.code }
