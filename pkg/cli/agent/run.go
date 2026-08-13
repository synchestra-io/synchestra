package agent

// Features implemented: agent-coordination, state-store/journal-batching
// Features depended on:  state-store, state-store/backends/git

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

// runCommand returns the "run" subgroup: the minimal run-lifecycle surface
// task-3's agent-coordination#ac:optional-model-provenance-is-correctable AC
// needs a CLI path for (start records a run's nullable model identity plus
// its runtime_observed/caller_declared provenance; correct appends an
// audited correction without rewriting the original event). Effort/run
// creation is otherwise intentionally NOT part of this command group's
// surface — see pkg/cli/agent/README.md — start/correct are the exception
// because the provenance-correction AC specifically requires them.
func runCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run lifecycle: start and correct nullable model provenance",
	}
	cmd.AddCommand(runStartCommand(), runCorrectCommand())
	return cmd
}

func modelProvenanceFlags(cmd *cobra.Command) {
	cmd.Flags().String("model", "", "model identity, if known (e.g. claude-opus-4-6); omit to record null/unknown")
	cmd.Flags().String("model-provenance", "", "runtime_observed or caller_declared (required when --model is set)")
}

func readModelProvenanceFlags(cmd *cobra.Command) (*string, state.ModelProvenance, error) {
	model, _ := cmd.Flags().GetString("model")
	provenanceFlag, _ := cmd.Flags().GetString("model-provenance")
	if strings.TrimSpace(model) == "" {
		if strings.TrimSpace(provenanceFlag) != "" {
			return nil, "", exitcode.InvalidArgsError("--model-provenance requires --model")
		}
		return nil, "", nil
	}
	provenance := state.ModelProvenance(provenanceFlag)
	if provenance != state.ModelProvenanceRuntimeObserved && provenance != state.ModelProvenanceCallerDeclared {
		return nil, "", exitcode.InvalidArgsError("--model-provenance must be runtime_observed or caller_declared when --model is set")
	}
	return &model, provenance, nil
}

func runStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new run within an effort",
		Args:  cobra.NoArgs,
		RunE:  runRunStart,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("effort", "", "effort ID this run executes within (required)")
	cmd.Flags().String("family", "", "agent family: codex, claude, or human (required)")
	cmd.Flags().String("role", "primary", "primary or subagent")
	cmd.Flags().String("parent", "", "parent run ID, if this is a subagent run")
	modelProvenanceFlags(cmd)
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runRunStart(cmd *cobra.Command, _ []string) error {
	effort, _ := cmd.Flags().GetString("effort")
	family, _ := cmd.Flags().GetString("family")
	role, _ := cmd.Flags().GetString("role")
	parent, _ := cmd.Flags().GetString("parent")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(effort) == "" {
		return exitcode.InvalidArgsError("--effort is required")
	}
	if strings.TrimSpace(family) == "" {
		return exitcode.InvalidArgsError("--family is required")
	}
	model, provenance, err := readModelProvenanceFlags(cmd)
	if err != nil {
		return err
	}

	store, _, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	// Start issues exactly one Append — safe to accelerate.
	run, err := state.CloseAfter(cmd.Context(), store, 0, func(ctx context.Context) (state.Run, error) {
		return store.Agent().Run().Start(ctx, state.RunStartParams{
			EffortID: effort, AgentFamily: state.AgentFamily(family), Model: model, ModelProvenance: provenance,
			Role: state.RunRole(role), ParentRunID: parent,
		})
	})
	if err != nil {
		return mapStoreError(err)
	}
	return writeRun(cmd.OutOrStdout(), format, run)
}

func runCorrectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "correct",
		Short: "Correct a run's model identity without rewriting its original event",
		Long: "Correct appends an audited correction event naming the superseded value and the " +
			"replacement — it never rewrites the original run.started event " +
			"(agent-coordination#ac:optional-model-provenance-is-correctable). Omit --model to " +
			"correct a wrongly-declared value back to null/unknown.",
		Args: cobra.NoArgs,
		RunE: runRunCorrect,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("run", "", "run ID to correct (required)")
	modelProvenanceFlags(cmd)
	cmd.Flags().String("reason", "", "why this correction is being made")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runRunCorrect(cmd *cobra.Command, _ []string) error {
	runID, _ := cmd.Flags().GetString("run")
	reason, _ := cmd.Flags().GetString("reason")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(runID) == "" {
		return exitcode.InvalidArgsError("--run is required")
	}
	model, provenance, err := readModelProvenanceFlags(cmd)
	if err != nil {
		return err
	}

	store, _, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	// Correct issues exactly one Append, but first reads the run (Run().
	// Correct -> Get -> loadRuns -> the journal's After(), a full scan) to
	// find it — the same unbounded-pre-read reason messageAck's comment
	// gives for not using state.CloseAfter here.
	run, err := store.Agent().Run().Correct(cmd.Context(), runID, state.RunCorrection{Model: model, ModelProvenance: provenance, Reason: reason})
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}
	return writeRun(cmd.OutOrStdout(), format, run)
}
