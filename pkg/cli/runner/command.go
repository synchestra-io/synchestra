package runner

// Features implemented: cli/runner, cli/runner/dispatch, cli/runner/invoke, wb-session-transport
// Features depended on:  cli/auth, dispatch, repo-config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

var dispatchIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Dependencies contains the side-effect boundaries used by runner commands.
type Dependencies struct {
	Getwd       func() (string, error)
	UserHomeDir func() (string, error)
	LookupEnv   func(string) (string, bool)
	HTTPClient  httpDoer
	NewID       func(string) (string, error)
}

type createOptions struct {
	prompt  string
	plan    string
	task    string
	runner  string
	profile string
	agent   string
	model   string
	effort  string
	format  string
}

// Command returns the "runner" command group. Supplying dependencies is
// intended for the root command and tests; callers may omit them.
func Command(optionalDependencies ...Dependencies) *cobra.Command {
	deps := defaultDependencies()
	if len(optionalDependencies) > 0 {
		deps = normalizeDependencies(optionalDependencies[0])
	}
	cmd := &cobra.Command{
		Use:          "runner",
		Short:        "Dispatch work to remote runners",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(dispatchCommand(deps), invokeCommand(deps))
	return cmd
}

func dispatchCommand(deps Dependencies) *cobra.Command {
	options := createOptions{}
	cmd := &cobra.Command{
		Use:   "dispatch [target]",
		Short: "Create and observe durable remote dispatches",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, deps, options, args)
		},
	}
	setFlagError(cmd, "create", &options.format)
	cmd.Flags().StringVar(&options.prompt, "prompt", "", "ad-hoc repository-scoped instruction")
	cmd.Flags().StringVar(&options.plan, "plan", "", "SpecScore Plan path, ID, or name")
	cmd.Flags().StringVar(&options.task, "task", "", "SpecScore Task path, ID, or name")
	cmd.Flags().StringVar(&options.runner, "runner", "", "constrain scheduling to this runner")
	cmd.Flags().StringVar(&options.profile, "profile", string(dispatchcontract.ProfileBalanced), "execution profile: fast, balanced, or large")
	cmd.Flags().StringVar(&options.agent, "agent", "", "exact agent adapter selector")
	cmd.Flags().StringVar(&options.model, "model", "", "exact or adapter-specific model selector")
	cmd.Flags().StringVar(&options.effort, "effort", "", "adapter-specific effort selector")
	cmd.Flags().StringVar(&options.format, "format", "text", "output format: text or json")
	cmd.AddCommand(
		statusCommand(deps),
		logsCommand(deps),
		retryCommand(deps),
		cancelCommand(deps),
	)
	return cmd
}

func runCreate(cmd *cobra.Command, deps Dependencies, options createOptions, args []string) error {
	resolved := resolvedOutput{Operation: "create", Runner: strings.TrimSpace(options.runner)}
	fail := func(err error) error {
		return outputFailure(cmd, options.format, resolved, err)
	}
	if err := validateFormat(options.format); err != nil {
		return fail(err)
	}
	if len(args) > 1 {
		return fail(invalidArgs("dispatch accepts at most one positional target"))
	}

	positional := ""
	if len(args) == 1 {
		positional = strings.TrimSpace(args[0])
	}
	sourceCount := 0
	for _, value := range []string{options.prompt, options.plan, options.task, positional} {
		if strings.TrimSpace(value) != "" {
			sourceCount++
		}
	}
	if sourceCount != 1 {
		return fail(invalidArgs("exactly one of --prompt, --plan, --task, or positional target is required"))
	}
	profile := dispatchcontract.ExecutionProfile(strings.TrimSpace(options.profile))
	requested := dispatchcontract.RequestedExecution{
		Profile:       profile,
		Agent:         strings.TrimSpace(options.agent),
		ModelSelector: strings.TrimSpace(options.model),
		Effort:        strings.TrimSpace(options.effort),
		Fallback:      dispatchcontract.FallbackPolicy{Mode: dispatchcontract.FallbackReject},
	}
	if err := requested.Validate(); err != nil {
		return fail(invalidArgs(err.Error()))
	}
	resolved.RequestedExecution = &requested

	cwd, err := deps.Getwd()
	if err != nil {
		return fail(unexpected("resolve current directory", err))
	}
	repository, err := resolveRepository(cmd.Context(), cwd)
	if err != nil {
		return fail(err)
	}
	resolved.Repository = &repository.Snapshot

	var source dispatchcontract.DispatchSource
	if strings.TrimSpace(options.prompt) != "" {
		source = dispatchcontract.DispatchSource{
			Kind:  dispatchcontract.SourceKindAdHoc,
			AdHoc: &dispatchcontract.AdHocSource{Prompt: strings.TrimSpace(options.prompt)},
		}
	} else {
		selector := targetSelector{Value: positional}
		if strings.TrimSpace(options.plan) != "" {
			selector = targetSelector{Kind: dispatchcontract.SpecScoreTargetPlan, Value: options.plan}
		}
		if strings.TrimSpace(options.task) != "" {
			selector = targetSelector{Kind: dispatchcontract.SpecScoreTargetTask, Value: options.task}
		}
		target, targetErr := resolveTarget(cmd.Context(), repository, selector)
		if targetErr != nil {
			return fail(targetErr)
		}
		source = dispatchcontract.DispatchSource{Kind: dispatchcontract.SourceKindSpecScore, SpecScore: &target}
	}
	resolved.Source = &source

	config, err := loadClientConfig(deps, repository.ProjectRoot)
	if err != nil {
		return fail(err)
	}
	client, err := newDispatchClient(config.BaseURL, config.Token, deps.HTTPClient)
	if err != nil {
		return fail(err)
	}
	idempotencyKey, err := deps.NewID("idem_")
	if err != nil {
		return fail(unexpected("generate dispatch idempotency key", err))
	}
	intent := dispatchcontract.DispatchIntent{
		Source:      source,
		Repository:  repository.Snapshot,
		Requested:   requested,
		Constraints: dispatchcontract.WorkerConstraints{RunnerID: strings.TrimSpace(options.runner)},
	}
	if err := intent.Validate(); err != nil {
		return fail(invalidArgs(fmt.Sprintf("invalid dispatch intent: %v", err)))
	}
	response, err := client.Create(cmd.Context(), dispatchcontract.CreateDispatchRequest{
		ProtocolVersion: dispatchcontract.ProtocolVersionV1,
		IdempotencyKey:  idempotencyKey,
		CreatedBy:       config.Actor,
		Intent:          intent,
	})
	if err != nil {
		return fail(err)
	}
	if options.format == "json" {
		return writeJSON(cmd.OutOrStdout(), createOutput{Resolved: resolved, Dispatch: &response.Dispatch})
	}
	return writeCreateText(cmd.OutOrStdout(), resolved, response.Dispatch)
}

func statusCommand(deps Dependencies) *cobra.Command {
	format := "text"
	cmd := &cobra.Command{
		Use:   "status <dispatch-id>",
		Short: "Inspect a dispatch and its attempts",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, dispatchID, err := validateObservationArgs("status", format, args)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			client, err := clientForObservation(deps)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			response, err := client.Status(cmd.Context(), dispatchID)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			invocation, invocationResponse, err := responseInvocation(response.Dispatch)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			if invocationResponse {
				resolved = applyInvocationResolution(resolved, response.Dispatch, invocation)
				publicDispatch := newInvocationDispatchOutput(response.Dispatch)
				publicAttempts := newInvocationAttemptOutputs(response.Attempts)
				if format == "json" {
					return writeJSON(cmd.OutOrStdout(), invocationStatusOutput{Resolved: resolved, Dispatch: &publicDispatch, Attempts: publicAttempts})
				}
				return writeInvocationStatusText(cmd.OutOrStdout(), resolved, publicDispatch, publicAttempts)
			}
			if format == "json" {
				return writeJSON(cmd.OutOrStdout(), statusOutput{Resolved: resolved, Dispatch: &response.Dispatch, Attempts: response.Attempts})
			}
			return writeStatusText(cmd.OutOrStdout(), response)
		},
	}
	setFlagError(cmd, "status", &format)
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func logsCommand(deps Dependencies) *cobra.Command {
	format := "text"
	var cursor int64
	cmd := &cobra.Command{
		Use:   "logs <dispatch-id>",
		Short: "Read the public dispatch log stream",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, dispatchID, err := validateObservationArgs("logs", format, args)
			resolved.Cursor = cursor
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			if cursor < 0 {
				return outputFailure(cmd, format, resolved, invalidArgs("--cursor cannot be negative"))
			}
			client, err := clientForObservation(deps)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			status, err := client.Status(cmd.Context(), dispatchID)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			invocation, invocationResponse, err := responseInvocation(status.Dispatch)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			response, err := client.Logs(cmd.Context(), dispatchID, cursor)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			if invocationResponse {
				resolved = applyInvocationResolution(resolved, status.Dispatch, invocation)
				if format == "json" {
					return writeJSON(cmd.OutOrStdout(), invocationLogsOutput{Resolved: resolved, Logs: newInvocationLogsPayload(response)})
				}
				return writeInvocationLogsText(cmd.OutOrStdout(), resolved, response)
			}
			if format == "json" {
				payload := &logsPayload{Reference: response.Reference, Events: response.Events, NextCursor: response.NextCursor}
				return writeJSON(cmd.OutOrStdout(), logsOutput{Resolved: resolved, Logs: payload})
			}
			return writeLogsText(cmd.OutOrStdout(), response)
		},
	}
	setFlagError(cmd, "logs", &format)
	cmd.Flags().Int64Var(&cursor, "cursor", 0, "return log events after this cursor")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func retryCommand(deps Dependencies) *cobra.Command {
	format := "text"
	reason := ""
	cmd := &cobra.Command{
		Use:   "retry <dispatch-id>",
		Short: "Append a retry attempt",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, dispatchID, err := validateObservationArgs("retry", format, args)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			client, config, err := clientAndConfigForObservation(deps)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			operationID, err := deps.NewID("op_")
			if err != nil {
				return outputFailure(cmd, format, resolved, unexpected("generate retry operation ID", err))
			}
			response, err := client.Retry(cmd.Context(), dispatchID, dispatchcontract.RetryDispatchRequest{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				DispatchID:      dispatchID,
				OperationID:     operationID,
				RequestedBy:     config.Actor,
				Reason:          strings.TrimSpace(reason),
			})
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			invocation, invocationResponse, err := responseInvocation(response.Dispatch)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			if invocationResponse {
				resolved = applyInvocationResolution(resolved, response.Dispatch, invocation)
				publicDispatch := newInvocationDispatchOutput(response.Dispatch)
				publicAttempt := newInvocationAttemptOutput(response.Attempt)
				if format == "json" {
					return writeJSON(cmd.OutOrStdout(), invocationRetryOutput{Resolved: resolved, Dispatch: &publicDispatch, Attempt: &publicAttempt})
				}
				return writeInvocationRetryText(cmd.OutOrStdout(), resolved, publicDispatch, publicAttempt)
			}
			if format == "json" {
				return writeJSON(cmd.OutOrStdout(), retryOutput{Resolved: resolved, Dispatch: &response.Dispatch, Attempt: &response.Attempt})
			}
			return writeRetryText(cmd.OutOrStdout(), response)
		},
	}
	setFlagError(cmd, "retry", &format)
	cmd.Flags().StringVar(&reason, "reason", "", "audit reason for retry")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func cancelCommand(deps Dependencies) *cobra.Command {
	format := "text"
	reason := ""
	cmd := &cobra.Command{
		Use:   "cancel <dispatch-id>",
		Short: "Request durable cancellation",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, dispatchID, err := validateObservationArgs("cancel", format, args)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			client, config, err := clientAndConfigForObservation(deps)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			operationID, err := deps.NewID("op_")
			if err != nil {
				return outputFailure(cmd, format, resolved, unexpected("generate cancel operation ID", err))
			}
			response, err := client.Cancel(cmd.Context(), dispatchID, dispatchcontract.CancelDispatchRequest{
				ProtocolVersion: dispatchcontract.ProtocolVersionV1,
				DispatchID:      dispatchID,
				OperationID:     operationID,
				RequestedBy:     config.Actor,
				Reason:          strings.TrimSpace(reason),
			})
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			invocation, invocationResponse, err := responseInvocation(response.Dispatch)
			if err != nil {
				return outputFailure(cmd, format, resolved, err)
			}
			if invocationResponse {
				resolved = applyInvocationResolution(resolved, response.Dispatch, invocation)
				publicDispatch := newInvocationDispatchOutput(response.Dispatch)
				if format == "json" {
					return writeJSON(cmd.OutOrStdout(), invocationMutationOutput{Resolved: resolved, Dispatch: &publicDispatch})
				}
				return writeInvocationMutationText(cmd.OutOrStdout(), resolved, publicDispatch)
			}
			if format == "json" {
				return writeJSON(cmd.OutOrStdout(), createOutput{Resolved: resolved, Dispatch: &response.Dispatch})
			}
			return writeCancelText(cmd.OutOrStdout(), response)
		},
	}
	setFlagError(cmd, "cancel", &format)
	cmd.Flags().StringVar(&reason, "reason", "", "audit reason for cancellation")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func validateObservationArgs(operation, format string, args []string) (resolvedOutput, string, error) {
	resolved := resolvedOutput{Operation: operation}
	if err := validateFormat(format); err != nil {
		return resolved, "", err
	}
	if len(args) != 1 {
		return resolved, "", invalidArgs(operation + " requires exactly one dispatch ID")
	}
	dispatchID := strings.TrimSpace(args[0])
	resolved.DispatchID = dispatchID
	if !dispatchIDPattern.MatchString(dispatchID) {
		return resolved, "", invalidArgs("invalid dispatch ID")
	}
	return resolved, dispatchID, nil
}

func validateFormat(format string) error {
	if format != "text" && format != "json" {
		return invalidArgs("--format must be text or json")
	}
	return nil
}

func outputFailure(cmd *cobra.Command, format string, resolved resolvedOutput, err error) error {
	commandErr := asCommandError(err)
	if format == "json" {
		if outputErr := writeErrorJSON(cmd.OutOrStdout(), resolved, commandErr); outputErr != nil {
			return outputErr
		}
	} else if (resolved.Operation == "create" && resolved.Source != nil) || (resolved.Operation == "invoke" && resolved.Invocation != nil) {
		if outputErr := writeCreateErrorText(cmd.OutOrStdout(), resolved, commandErr); outputErr != nil {
			return outputErr
		}
	}
	return commandErr
}

func clientForObservation(deps Dependencies) (*dispatchClient, error) {
	client, _, err := clientAndConfigForObservation(deps)
	return client, err
}

func clientAndConfigForObservation(deps Dependencies) (*dispatchClient, clientConfig, error) {
	cwd, err := deps.Getwd()
	if err != nil {
		return nil, clientConfig{}, unexpected("resolve current directory", err)
	}
	projectRoot := findConfigRoot(cwd)
	config, err := loadClientConfig(deps, projectRoot)
	if err != nil {
		return nil, clientConfig{}, err
	}
	client, err := newDispatchClient(config.BaseURL, config.Token, deps.HTTPClient)
	if err != nil {
		return nil, clientConfig{}, err
	}
	return client, config, nil
}

func findConfigRoot(cwd string) string {
	current, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "synchestra.yaml")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func setFlagError(cmd *cobra.Command, operation string, format *string) {
	cmd.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		commandErr := invalidArgs(err.Error())
		if format != nil && *format == "json" {
			if outputErr := writeErrorJSON(command.OutOrStdout(), resolvedOutput{Operation: operation}, commandErr); outputErr != nil {
				return outputErr
			}
		}
		return commandErr
	})
}

func defaultDependencies() Dependencies {
	return Dependencies{
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		LookupEnv:   os.LookupEnv,
		HTTPClient:  &http.Client{Timeout: defaultHubTimeout},
		NewID:       randomID,
	}
}

func normalizeDependencies(deps Dependencies) Dependencies {
	defaults := defaultDependencies()
	if deps.Getwd == nil {
		deps.Getwd = defaults.Getwd
	}
	if deps.UserHomeDir == nil {
		deps.UserHomeDir = defaults.UserHomeDir
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = defaults.LookupEnv
	}
	if deps.HTTPClient == nil {
		deps.HTTPClient = defaults.HTTPClient
	}
	if deps.NewID == nil {
		deps.NewID = defaults.NewID
	}
	return deps
}

func randomID(prefix string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(random), nil
}
