package agent

// Features implemented: agent-coordination, state-store/journal-batching
// Features depended on:  state-store, state-store/backends/git

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synchestra-io/specscore/pkg/exitcode"
	"github.com/synchestra-io/synchestra/pkg/state"
)

func messageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message",
		Short: "Audited message-thread operations (send, list, ack)",
	}
	cmd.AddCommand(messageSendCommand(), messageListCommand(), messageAckCommand())
	return cmd
}

func messageSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a typed, audited message in a coordination thread",
		Long: "Send records an immutable message envelope. --kind categorizes the typed negotiation " +
			"sequence (coordination.request, coordination.proposal, coordination.counterexample, " +
			"coordination.decision.accepted — agent-coordination/cross-harness-conformance); a " +
			"decision.accepted message requires at least one --evidence reference.",
		Args: cobra.NoArgs,
		RunE: runMessageSend,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("thread", "", "thread ID this message belongs to (required)")
	cmd.Flags().String("run", "", "sender run ID (required)")
	cmd.Flags().StringSlice("to", nil, "recipient run IDs")
	cmd.Flags().String("effort", "", "effort ID this message relates to (optional)")
	cmd.Flags().String("correlation", "", "correlation ID (defaults to --thread)")
	cmd.Flags().String("repository", "", "repository this message relates to (optional)")
	cmd.Flags().String("claim", "", "claim ID this message relates to (optional)")
	cmd.Flags().String("kind", "", "coordination.request, coordination.proposal, coordination.counterexample, coordination.decision.accepted, or coordination.note (default)")
	cmd.Flags().StringSlice("evidence", nil, "evidence references as kind:description:reference (repeatable; required for --kind coordination.decision.accepted)")
	cmd.Flags().String("body", "", "message body")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runMessageSend(cmd *cobra.Command, _ []string) error {
	thread, _ := cmd.Flags().GetString("thread")
	sender, _ := cmd.Flags().GetString("run")
	recipients, _ := cmd.Flags().GetStringSlice("to")
	effort, _ := cmd.Flags().GetString("effort")
	correlation, _ := cmd.Flags().GetString("correlation")
	repository, _ := cmd.Flags().GetString("repository")
	claimID, _ := cmd.Flags().GetString("claim")
	kind, _ := cmd.Flags().GetString("kind")
	evidenceFlags, _ := cmd.Flags().GetStringSlice("evidence")
	body, _ := cmd.Flags().GetString("body")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(thread) == "" {
		return exitcode.InvalidArgsError("--thread is required")
	}
	if strings.TrimSpace(sender) == "" {
		return exitcode.InvalidArgsError("--run is required")
	}
	evidence, err := parseEvidenceFlags(evidenceFlags)
	if err != nil {
		return err
	}

	store, _, err := resolveStore(cmd, project, syncFlag, sender)
	if err != nil {
		return err
	}
	// Send issues exactly one Append, so it is safe to accelerate through
	// state.CloseAfter — see that function's doc comment for the "at most
	// one Append" contract this relies on.
	msg, err := state.CloseAfter(cmd.Context(), store, 0, func(ctx context.Context) (state.Message, error) {
		return store.Agent().Message().Send(ctx, state.MessageSendParams{
			EffortID: effort, ThreadID: thread, SenderRunID: sender, RecipientRunIDs: recipients,
			CorrelationID: correlation, RepositoryID: repository, ClaimID: claimID,
			Body: body, Kind: state.MessageKind(kind), Evidence: evidence,
		})
	})
	if err != nil {
		return mapStoreError(err)
	}
	return writeMessage(cmd.OutOrStdout(), format, msg)
}

func messageListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every message in a thread, in delivery order",
		Args:  cobra.NoArgs,
		RunE:  runMessageList,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("thread", "", "thread ID (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	cmd.Flags().String("format", "yaml", "output format: yaml, json")
	return cmd
}

func runMessageList(cmd *cobra.Command, _ []string) error {
	thread, _ := cmd.Flags().GetString("thread")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")
	format, _ := cmd.Flags().GetString("format")

	if strings.TrimSpace(thread) == "" {
		return exitcode.InvalidArgsError("--thread is required")
	}

	store, _, err := resolveStore(cmd, project, syncFlag, "")
	if err != nil {
		return err
	}
	messages, err := store.Agent().Message().Thread(cmd.Context(), thread)
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}
	return writeMessageList(cmd.OutOrStdout(), format, messages)
}

func messageAckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ack",
		Short: "Acknowledge a message as a recipient",
		Args:  cobra.NoArgs,
		RunE:  runMessageAck,
	}
	cmd.Flags().String("project", "", "project identifier, e.g. github.com/org/repo (autodetected from the origin Git remote if omitted)")
	cmd.Flags().String("message", "", "message ID to acknowledge (required)")
	cmd.Flags().String("run", "", "acknowledging recipient run ID (required)")
	cmd.Flags().String("sync", "", "override sync policy for this invocation (remote, local)")
	return cmd
}

func runMessageAck(cmd *cobra.Command, _ []string) error {
	messageID, _ := cmd.Flags().GetString("message")
	run, _ := cmd.Flags().GetString("run")
	project, _ := cmd.Flags().GetString("project")
	syncFlag, _ := cmd.Flags().GetString("sync")

	if strings.TrimSpace(messageID) == "" {
		return exitcode.InvalidArgsError("--message is required")
	}
	if strings.TrimSpace(run) == "" {
		return exitcode.InvalidArgsError("--run is required")
	}

	store, _, err := resolveStore(cmd, project, syncFlag, run)
	if err != nil {
		return err
	}
	// Acknowledge issues exactly one Append, but first reads the FULL
	// message history (Message().Acknowledge -> loadMessages -> the
	// journal's After(), which scans every event, not the O(1) Head() Send
	// reads) to find the message being acknowledged. state.CloseAfter's
	// fixed grace period only accounts for the enqueue step, not an
	// unbounded pre-read that grows with journal history, so this call is
	// NOT wrapped in CloseAfter — see its doc comment's "at most one
	// Append" contract and pkg/cli/agent/README.md.
	_, err = store.Agent().Message().Acknowledge(cmd.Context(), messageID, run)
	if closeErr := closeStore(cmd.Context(), store); closeErr != nil && err == nil {
		return closeErr
	}
	if err != nil {
		return mapStoreError(err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "message %s acknowledged by %s\n", messageID, run)
	return nil
}
