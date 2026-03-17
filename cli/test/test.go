package test

// Features implemented: cli
// Features depended on:  testing-framework/test-scenario, testing-framework/test-runner

import (
	"github.com/spf13/cobra"
	rehearse "github.com/synchestra-io/rehearse/cli"
)

// Command returns the "test" cobra command backed by the Rehearse testing framework.
func Command() *cobra.Command {
	return rehearse.Command()
}
