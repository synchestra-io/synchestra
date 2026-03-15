# exitcode

Package `exitcode` defines an error type that carries a process exit code alongside an error message. It is used by CLI commands to signal specific exit codes (1 = conflict, 2 = invalid arguments, 3 = repo not found, 10+ = unexpected errors) while still returning a Go `error`.

## Outstanding Questions

None at this time.
