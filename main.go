package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/synchesta-io/synchestra/cli"
)

var (
	exit = os.Exit
)

func main() {
	fatal := func(err error) {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		type exitCoder interface{ ExitCode() int }
		var ec exitCoder
		if errors.As(err, &ec) {
			exit(ec.ExitCode())
			return
		}
		exit(1)
	}
	logf := func(args ...any) {
		_, _ = fmt.Fprintln(os.Stderr, args...)
	}
	cli.Run(os.Args, os.UserHomeDir, os.Getwd, fatal, logf)
}
