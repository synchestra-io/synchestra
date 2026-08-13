// Package agent implements the "synchestra agent" CLI command group:
// fenced worktree-claim operations (claim/list/renew/release/handoff),
// audited messaging (message send/list/ack), and the minimal run-lifecycle
// surface agent-coordination#ac:optional-model-provenance-is-correctable
// needs (run start/correct), per spec/features/agent-coordination and
// task-3 of spec/plans/synchestra-coordination-foundation.md. It follows
// pkg/cli/task's conventions (command registration, per-command --project/
// --sync flags, exitcode error mapping, YAML-default output) rather than
// introducing new ones.
package agent

// Features implemented: agent-coordination

import "github.com/spf13/cobra"

// Command returns the "agent" command group.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Fenced worktree claims and audited messaging (agent coordination)",
	}
	cmd.AddCommand(
		claimCommand(),
		listCommand(),
		renewCommand(),
		releaseCommand(),
		handoffCommand(),
		messageCommand(),
		runCommand(),
	)
	return cmd
}
