package runner

// Features implemented: cli/runner/invoke, wb-session-transport
// Features depended on:  cli/auth, cli/runner/dispatch, dispatch, repo-config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

type invokeOptions struct {
	runner       string
	handler      string
	invocationID string
	deadline     string
	format       string
}

func invokeCommand(deps Dependencies) *cobra.Command {
	options := invokeOptions{}
	cmd := &cobra.Command{
		Use:   "invoke @<payload-file>",
		Short: "Invoke a registered handler through a durable dispatch",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvoke(cmd, deps, options, args)
		},
	}
	setFlagError(cmd, "invoke", &options.format)
	cmd.Flags().StringVar(&options.runner, "runner", "", "required target runner")
	cmd.Flags().StringVar(&options.handler, "handler", "", "required registered handler name")
	cmd.Flags().StringVar(&options.invocationID, "invocation-id", "", "required caller-owned invocation identifier")
	cmd.Flags().StringVar(&options.deadline, "deadline", "", "optional RFC3339 invocation deadline")
	cmd.Flags().StringVar(&options.format, "format", "text", "output format: text or json")
	return cmd
}

func runInvoke(cmd *cobra.Command, deps Dependencies, options invokeOptions, args []string) error {
	resolved := resolvedOutput{Operation: "invoke", Runner: strings.TrimSpace(options.runner)}
	fail := func(err error) error {
		return outputFailure(cmd, options.format, resolved, err)
	}
	if err := validateFormat(options.format); err != nil {
		return fail(err)
	}
	if len(args) != 1 || !strings.HasPrefix(args[0], "@") || len(args[0]) == 1 {
		return fail(invalidArgs("invoke requires exactly one @<payload-file> argument"))
	}
	if resolved.Runner == "" {
		return fail(invalidArgs("--runner is required"))
	}
	handler := dispatchcontract.HandlerName(strings.TrimSpace(options.handler))
	if handler == "" {
		return fail(invalidArgs("--handler is required"))
	}
	if !dispatchcontract.IsSupportedHandler(handler) {
		return fail(invalidArgs("--handler names an unsupported registered handler"))
	}
	invocationID := strings.TrimSpace(options.invocationID)
	if invocationID == "" {
		return fail(invalidArgs("--invocation-id is required"))
	}
	deadline, err := parseInvocationDeadline(options.deadline)
	if err != nil {
		return fail(err)
	}

	cwd, err := deps.Getwd()
	if err != nil {
		return fail(unexpected("resolve current directory", err))
	}
	payload, err := readInvocationPayload(cwd, strings.TrimPrefix(args[0], "@"))
	if err != nil {
		return fail(err)
	}
	invocation, err := dispatchcontract.NewHandlerInvocation(invocationID, handler, payload, deadline)
	if err != nil {
		return fail(invalidArgs(err.Error()))
	}
	resolved.Invocation = newInvocationMetadata(invocation, time.Time{})

	repository, err := resolveRepository(cmd.Context(), cwd)
	if err != nil {
		return fail(err)
	}
	resolved.Repository = &repository.Snapshot
	requested, err := dispatchcontract.HandlerRequestedExecution(handler)
	if err != nil {
		return fail(invalidArgs(err.Error()))
	}
	requiredCapability, err := dispatchcontract.HandlerRequiredCapability(handler)
	if err != nil {
		return fail(invalidArgs(err.Error()))
	}
	source, err := dispatchcontract.EncodeHandlerInvocation(invocation)
	if err != nil {
		return fail(invalidArgs(err.Error()))
	}
	intent := dispatchcontract.DispatchIntent{
		Source:     source,
		Repository: repository.Snapshot,
		Requested:  requested,
		Constraints: dispatchcontract.WorkerConstraints{
			RunnerID:             resolved.Runner,
			RequiredCapabilities: []string{requiredCapability},
		},
	}
	if err := intent.Validate(); err != nil {
		return fail(invalidArgs(fmt.Sprintf("invalid handler dispatch intent: %v", err)))
	}

	config, err := loadClientConfig(deps, repository.ProjectRoot)
	if err != nil {
		return fail(err)
	}
	client, err := newDispatchClient(config.BaseURL, config.Token, deps.HTTPClient)
	if err != nil {
		return fail(err)
	}
	idempotencyKey, err := invocationIdempotencyKey(deps, handler, invocationID)
	if err != nil {
		return fail(err)
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
	returnedInvocation, ok, err := dispatchcontract.ParseHandlerInvocation(response.Dispatch.Intent.Source)
	if err != nil || !ok || !sameInvocationRequest(invocation, returnedInvocation) {
		return fail(unexpected("Hub create response did not contain the requested handler invocation", err))
	}
	dispatch := response.Dispatch
	attempts := []dispatchcontract.Attempt{}
	if !response.Created {
		status, statusErr := client.Status(cmd.Context(), dispatch.ID)
		if statusErr != nil {
			return fail(statusErr)
		}
		statusInvocation, statusOK, statusParseErr := dispatchcontract.ParseHandlerInvocation(status.Dispatch.Intent.Source)
		if statusParseErr != nil || !statusOK || !sameInvocationRequest(invocation, statusInvocation) {
			return fail(unexpected("Hub status response did not contain the requested handler invocation", statusParseErr))
		}
		dispatch = status.Dispatch
		returnedInvocation = statusInvocation
		attempts = status.Attempts
	}
	resolved = applyInvocationResolution(resolved, dispatch, returnedInvocation)

	publicDispatch := newInvocationDispatchOutput(dispatch)
	publicAttempts := newInvocationAttemptOutputs(attempts)
	if options.format == "json" {
		return writeJSON(cmd.OutOrStdout(), invocationCreateOutput{
			Resolved: resolved,
			Dispatch: &publicDispatch,
			Attempts: publicAttempts,
			Created:  response.Created,
		})
	}
	return writeInvocationCreateText(cmd.OutOrStdout(), resolved, publicDispatch, response.Created, publicAttempts)
}

func parseInvocationDeadline(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	deadline, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, invalidArgs("--deadline must be an RFC3339 timestamp")
	}
	deadline = deadline.UTC()
	return &deadline, nil
}

func readInvocationPayload(cwd, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, invalidArgs("@<payload-file> path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, filepath.FromSlash(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, invalidArgs("cannot read @<payload-file>")
	}
	payload, readErr := io.ReadAll(io.LimitReader(file, dispatchcontract.MaxHandlerPayloadBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, invalidArgs("cannot read @<payload-file>")
	}
	if len(payload) > dispatchcontract.MaxHandlerPayloadBytes {
		return nil, invalidArgs("@<payload-file> exceeds the 1 MiB limit")
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, invalidArgs("@<payload-file> is empty")
	}
	if !json.Valid(payload) {
		return nil, invalidArgs("@<payload-file> must contain one valid JSON value")
	}
	return payload, nil
}

func invocationIdempotencyKey(deps Dependencies, handler dispatchcontract.HandlerName, invocationID string) (string, error) {
	if handler == dispatchcontract.HandlerNameWBSessionAcceptV1 {
		key, err := dispatchcontract.WBHandoffIdempotencyKey(invocationID)
		if err != nil {
			return "", invalidArgs(err.Error())
		}
		return key, nil
	}
	key, err := deps.NewID("idem_")
	if err != nil {
		return "", unexpected("generate invocation idempotency key", err)
	}
	return key, nil
}

func sameInvocationRequest(left, right dispatchcontract.HandlerInvocation) bool {
	if left.ProtocolVersion != right.ProtocolVersion || left.ID != right.ID || left.Handler != right.Handler ||
		left.PayloadDigest != right.PayloadDigest || left.PayloadSize != right.PayloadSize || !bytes.Equal(left.Payload, right.Payload) {
		return false
	}
	if left.Deadline == nil || right.Deadline == nil {
		return left.Deadline == nil && right.Deadline == nil
	}
	return left.Deadline.Equal(*right.Deadline)
}

func responseInvocation(dispatch dispatchcontract.Dispatch) (dispatchcontract.HandlerInvocation, bool, error) {
	invocation, ok, err := dispatchcontract.ParseHandlerInvocation(dispatch.Intent.Source)
	if err != nil {
		return dispatchcontract.HandlerInvocation{}, true, unexpected("Hub response contained an invalid handler invocation", err)
	}
	return invocation, ok, nil
}
