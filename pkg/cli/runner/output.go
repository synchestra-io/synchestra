package runner

// Features implemented: cli/runner/dispatch, cli/runner/invoke, wb-session-transport
// Features depended on:  dispatch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

type resolvedOutput struct {
	Operation          string                               `json:"operation"`
	DispatchID         string                               `json:"dispatch_id,omitempty"`
	Cursor             int64                                `json:"cursor,omitempty"`
	Source             *dispatchcontract.DispatchSource     `json:"source,omitempty"`
	Repository         *dispatchcontract.RepositorySnapshot `json:"repository,omitempty"`
	RequestedExecution *dispatchcontract.RequestedExecution `json:"requested_execution,omitempty"`
	Runner             string                               `json:"runner,omitempty"`
	Invocation         *invocationMetadataOutput            `json:"invocation,omitempty"`
}

type invocationMetadataOutput struct {
	ProtocolVersion string                       `json:"protocol_version"`
	ID              string                       `json:"id"`
	Handler         dispatchcontract.HandlerName `json:"handler"`
	PayloadDigest   string                       `json:"payload_digest"`
	PayloadSize     int64                        `json:"payload_size"`
	CreatedAt       *time.Time                   `json:"created_at,omitempty"`
	Deadline        *time.Time                   `json:"deadline,omitempty"`
}

type invocationDispatchOutput struct {
	ProtocolVersion string                          `json:"protocol_version"`
	ID              string                          `json:"id"`
	Status          dispatchcontract.DispatchStatus `json:"status"`
	AttemptIDs      []string                        `json:"attempt_ids"`
	ActiveAttemptID string                          `json:"active_attempt_id,omitempty"`
	Cancellation    *invocationCancellationOutput   `json:"cancellation,omitempty"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

type invocationCancellationOutput struct {
	RequestedAt    time.Time  `json:"requested_at"`
	RequestedBy    string     `json:"requested_by"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

type invocationLeaseOutput struct {
	Owner           dispatchcontract.WorkerIdentity `json:"owner"`
	Generation      int64                           `json:"generation"`
	AcquiredAt      time.Time                       `json:"acquired_at"`
	ExpiresAt       time.Time                       `json:"expires_at"`
	LastHeartbeatAt time.Time                       `json:"last_heartbeat_at"`
}

type invocationResultOutput struct {
	ArtifactReferences []string  `json:"artifact_references"`
	PublishedAt        time.Time `json:"published_at"`
}

type invocationFailureOutput struct {
	Stage     string                         `json:"stage"`
	Code      string                         `json:"code"`
	Retryable bool                           `json:"retryable"`
	Logs      *dispatchcontract.LogReference `json:"logs,omitempty"`
}

type invocationAttemptOutput struct {
	ProtocolVersion string                             `json:"protocol_version"`
	ID              string                             `json:"id"`
	DispatchID      string                             `json:"dispatch_id"`
	Number          int                                `json:"number"`
	Status          dispatchcontract.AttemptStatus     `json:"status"`
	Worker          *dispatchcontract.WorkerIdentity   `json:"worker,omitempty"`
	Lease           *invocationLeaseOutput             `json:"lease,omitempty"`
	Session         *dispatchcontract.SessionReference `json:"session,omitempty"`
	Logs            *dispatchcontract.LogReference     `json:"logs,omitempty"`
	Result          *invocationResultOutput            `json:"result,omitempty"`
	Failure         *invocationFailureOutput           `json:"failure,omitempty"`
	CancellationAck *time.Time                         `json:"cancellation_acknowledged_at,omitempty"`
	CreatedAt       time.Time                          `json:"created_at"`
	StartedAt       *time.Time                         `json:"started_at,omitempty"`
	FinishedAt      *time.Time                         `json:"finished_at,omitempty"`
}

type invocationCreateOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *invocationDispatchOutput  `json:"dispatch,omitempty"`
	Attempts []invocationAttemptOutput  `json:"attempts"`
	Created  bool                       `json:"created"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type invocationStatusOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *invocationDispatchOutput  `json:"dispatch,omitempty"`
	Attempts []invocationAttemptOutput  `json:"attempts"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type invocationRetryOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *invocationDispatchOutput  `json:"dispatch,omitempty"`
	Attempt  *invocationAttemptOutput   `json:"attempt,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type invocationMutationOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *invocationDispatchOutput  `json:"dispatch,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type createOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *dispatchcontract.Dispatch `json:"dispatch,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type statusOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *dispatchcontract.Dispatch `json:"dispatch,omitempty"`
	Attempts []dispatchcontract.Attempt `json:"attempts"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type logsPayload struct {
	Reference  dispatchcontract.LogReference `json:"reference"`
	Events     []dispatchcontract.LogEvent   `json:"events"`
	NextCursor int64                         `json:"next_cursor"`
}

type logsOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Logs     *logsPayload               `json:"logs,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type invocationLogEventOutput struct {
	Sequence  int64                     `json:"sequence"`
	Timestamp time.Time                 `json:"timestamp"`
	Level     dispatchcontract.LogLevel `json:"level"`
	Stage     string                    `json:"stage,omitempty"`
}

type invocationLogsPayload struct {
	Reference  dispatchcontract.LogReference `json:"reference"`
	Events     []invocationLogEventOutput    `json:"events"`
	NextCursor int64                         `json:"next_cursor"`
}

type invocationLogsOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Logs     *invocationLogsPayload     `json:"logs,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

type retryOutput struct {
	Resolved resolvedOutput             `json:"resolved"`
	Dispatch *dispatchcontract.Dispatch `json:"dispatch,omitempty"`
	Attempt  *dispatchcontract.Attempt  `json:"attempt,omitempty"`
	Error    *dispatchcontract.APIError `json:"error,omitempty"`
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return unexpected("write JSON output", err)
	}
	return nil
}

func writeCreateText(writer io.Writer, resolved resolvedOutput, dispatch dispatchcontract.Dispatch) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nDispatch:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  dispatch-id: %s\n", displayValue(dispatch.ID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  status:      %s\n", displayValue(string(dispatch.Status))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  created-at:  %s\n", formatTime(dispatch.CreatedAt)); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeResolvedText(writer io.Writer, resolved resolvedOutput) error {
	if _, err := fmt.Fprintln(writer, "Resolved:"); err != nil {
		return err
	}
	if resolved.Source != nil {
		if resolved.Source.Kind == dispatchcontract.SourceKindAdHoc {
			if _, err := fmt.Fprintln(writer, "  source:       ad_hoc"); err != nil {
				return err
			}
		} else if resolved.Source.SpecScore != nil {
			source := resolved.Source.SpecScore
			if _, err := fmt.Fprintf(writer, "  source:       specscore %s %s (%s)\n", source.TargetKind, source.TargetID, source.TargetPath); err != nil {
				return err
			}
		}
	}
	if resolved.Invocation != nil {
		invocation := resolved.Invocation
		if _, err := fmt.Fprintf(writer, "  invocation:   %s\n", displayValue(invocation.ID)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  handler:      %s\n", displayValue(string(invocation.Handler))); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  payload:      %s (%d bytes)\n", displayValue(invocation.PayloadDigest), invocation.PayloadSize); err != nil {
			return err
		}
		if invocation.CreatedAt != nil {
			if _, err := fmt.Fprintf(writer, "  created-at:   %s\n", formatTime(*invocation.CreatedAt)); err != nil {
				return err
			}
		}
		if invocation.Deadline != nil {
			if _, err := fmt.Fprintf(writer, "  deadline:     %s\n", formatTime(*invocation.Deadline)); err != nil {
				return err
			}
		}
	}
	if resolved.Repository != nil {
		repository := resolved.Repository
		if _, err := fmt.Fprintf(writer, "  repository:   %s\n", displayValue(repository.CanonicalID)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  revision:     %s\n", displayValue(repository.BaseRevision)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  base-ref:     %s\n", displayValue(repository.BaseRef)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  subdirectory: %s\n", displayValue(repository.Subdirectory)); err != nil {
			return err
		}
	}
	if resolved.RequestedExecution != nil {
		requested := resolved.RequestedExecution
		if _, err := fmt.Fprintf(writer, "  profile:      %s\n", displayValue(string(requested.Profile))); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  agent:        %s\n", displayValue(requested.Agent)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  model:        %s\n", displayValue(requested.ModelSelector)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "  effort:       %s\n", displayValue(requested.Effort)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "  runner:       %s\n", displayRunner(resolved.Runner)); err != nil {
		return err
	}
	return nil
}

func newInvocationMetadata(invocation dispatchcontract.HandlerInvocation, createdAt time.Time) *invocationMetadataOutput {
	result := &invocationMetadataOutput{
		ProtocolVersion: invocation.ProtocolVersion,
		ID:              invocation.ID,
		Handler:         invocation.Handler,
		PayloadDigest:   invocation.PayloadDigest,
		PayloadSize:     invocation.PayloadSize,
		Deadline:        invocation.Deadline,
	}
	if !createdAt.IsZero() {
		canonical := createdAt.UTC()
		result.CreatedAt = &canonical
	}
	return result
}

func newInvocationDispatchOutput(dispatch dispatchcontract.Dispatch) invocationDispatchOutput {
	attemptIDs := append([]string(nil), dispatch.AttemptIDs...)
	if attemptIDs == nil {
		attemptIDs = []string{}
	}
	result := invocationDispatchOutput{
		ProtocolVersion: dispatch.ProtocolVersion,
		ID:              dispatch.ID,
		Status:          dispatch.Status,
		AttemptIDs:      attemptIDs,
		ActiveAttemptID: dispatch.ActiveAttemptID,
		CreatedAt:       dispatch.CreatedAt,
		UpdatedAt:       dispatch.UpdatedAt,
	}
	if dispatch.Cancellation != nil {
		result.Cancellation = &invocationCancellationOutput{
			RequestedAt:    dispatch.Cancellation.RequestedAt,
			RequestedBy:    dispatch.Cancellation.RequestedBy,
			AcknowledgedAt: dispatch.Cancellation.AcknowledgedAt,
		}
	}
	return result
}

func newInvocationAttemptOutput(attempt dispatchcontract.Attempt) invocationAttemptOutput {
	result := invocationAttemptOutput{
		ProtocolVersion: attempt.ProtocolVersion,
		ID:              attempt.ID,
		DispatchID:      attempt.DispatchID,
		Number:          attempt.Number,
		Status:          attempt.Status,
		Session:         attempt.Session,
		Logs:            attempt.Logs,
		CancellationAck: attempt.CancellationAck,
		CreatedAt:       attempt.CreatedAt,
		StartedAt:       attempt.StartedAt,
		FinishedAt:      attempt.FinishedAt,
	}
	if attempt.Failure != nil {
		result.Failure = &invocationFailureOutput{
			Stage:     attempt.Failure.Stage,
			Code:      attempt.Failure.Code,
			Retryable: attempt.Failure.Retryable,
			Logs:      attempt.Failure.Logs,
		}
	}
	if attempt.Worker != nil {
		worker := attempt.Worker.Identity
		result.Worker = &worker
	}
	if attempt.Lease != nil {
		result.Lease = &invocationLeaseOutput{
			Owner:           attempt.Lease.Owner,
			Generation:      attempt.Lease.Generation,
			AcquiredAt:      attempt.Lease.AcquiredAt,
			ExpiresAt:       attempt.Lease.ExpiresAt,
			LastHeartbeatAt: attempt.Lease.LastHeartbeatAt,
		}
	}
	if attempt.Result != nil {
		artifactReferences := make([]string, 0, len(attempt.Result.Validation))
		for _, evidence := range attempt.Result.Validation {
			if strings.TrimSpace(evidence.ArtifactRef) != "" {
				artifactReferences = append(artifactReferences, evidence.ArtifactRef)
			}
		}
		result.Result = &invocationResultOutput{
			ArtifactReferences: artifactReferences,
			PublishedAt:        attempt.Result.PublishedAt,
		}
	}
	return result
}

func newInvocationLogsPayload(response dispatchcontract.GetLogsResponse) *invocationLogsPayload {
	events := make([]invocationLogEventOutput, 0, len(response.Events))
	for _, event := range response.Events {
		events = append(events, invocationLogEventOutput{
			Sequence:  event.Sequence,
			Timestamp: event.Timestamp,
			Level:     event.Level,
			Stage:     event.Stage,
		})
	}
	return &invocationLogsPayload{
		Reference:  response.Reference,
		Events:     events,
		NextCursor: response.NextCursor,
	}
}

func newInvocationAttemptOutputs(attempts []dispatchcontract.Attempt) []invocationAttemptOutput {
	result := make([]invocationAttemptOutput, 0, len(attempts))
	for _, attempt := range attempts {
		result = append(result, newInvocationAttemptOutput(attempt))
	}
	return result
}

func applyInvocationResolution(resolved resolvedOutput, dispatch dispatchcontract.Dispatch, invocation dispatchcontract.HandlerInvocation) resolvedOutput {
	repository := dispatch.Intent.Repository
	resolved.Repository = &repository
	resolved.Runner = dispatch.Intent.Constraints.RunnerID
	resolved.Invocation = newInvocationMetadata(invocation, dispatch.CreatedAt)
	resolved.Source = nil
	resolved.RequestedExecution = nil
	return resolved
}

func writeInvocationCreateText(writer io.Writer, resolved resolvedOutput, dispatch invocationDispatchOutput, created bool, attempts []invocationAttemptOutput) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nDispatch:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  dispatch-id: %s\n", displayValue(dispatch.ID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  status:      %s\n", displayValue(string(dispatch.Status))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  created:     %t\n", created); err != nil {
		return err
	}
	if err := writeInvocationAttemptsText(buffered, attempts); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeInvocationStatusText(writer io.Writer, resolved resolvedOutput, dispatch invocationDispatchOutput, attempts []invocationAttemptOutput) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered); err != nil {
		return err
	}
	if err := writeInvocationDispatchText(buffered, dispatch); err != nil {
		return err
	}
	if err := writeInvocationAttemptsText(buffered, attempts); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeInvocationDispatchText(writer io.Writer, dispatch invocationDispatchOutput) error {
	if _, err := fmt.Fprintln(writer, "Dispatch:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  dispatch-id: %s\n", displayValue(dispatch.ID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  status:      %s\n", displayValue(string(dispatch.Status))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  updated-at:  %s\n", formatTime(dispatch.UpdatedAt)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  attempt-id:  %s\n", displayValue(dispatch.ActiveAttemptID)); err != nil {
		return err
	}
	return nil
}

func writeInvocationAttemptsText(writer io.Writer, attempts []invocationAttemptOutput) error {
	if _, err := fmt.Fprintln(writer, "\nAttempts:"); err != nil {
		return err
	}
	if len(attempts) == 0 {
		if _, err := fmt.Fprintln(writer, "  none"); err != nil {
			return err
		}
	}
	for _, attempt := range attempts {
		if _, err := fmt.Fprintf(writer, "  %d  %s  %s\n", attempt.Number, displayValue(attempt.ID), displayValue(string(attempt.Status))); err != nil {
			return err
		}
		if attempt.Worker != nil {
			if _, err := fmt.Fprintf(writer, "    runner:           %s\n", displayRunner(attempt.Worker.RunnerID)); err != nil {
				return err
			}
		}
		if attempt.Lease != nil {
			if _, err := fmt.Fprintf(writer, "    lease-generation: %d\n", attempt.Lease.Generation); err != nil {
				return err
			}
		}
		if attempt.Result != nil {
			if _, err := fmt.Fprintln(writer, "    Result:"); err != nil {
				return err
			}
			for _, artifact := range attempt.Result.ArtifactReferences {
				if _, err := fmt.Fprintf(writer, "      artifact: %s\n", displayLine(artifact)); err != nil {
					return err
				}
			}
		}
		if attempt.Failure != nil {
			if _, err := fmt.Fprintf(writer, "    failure: %s (%s, retryable=%t)\n", displayLine(attempt.Failure.Code), displayLine(attempt.Failure.Stage), attempt.Failure.Retryable); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeStatusText(writer io.Writer, response dispatchcontract.GetDispatchResponse) error {
	buffered := bufio.NewWriter(writer)
	if err := writeDispatchText(buffered, response.Dispatch); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nAttempts:"); err != nil {
		return err
	}
	if len(response.Attempts) == 0 {
		if _, err := fmt.Fprintln(buffered, "  none"); err != nil {
			return err
		}
	}
	for _, attempt := range response.Attempts {
		if _, err := fmt.Fprintf(buffered, "  %d  %s  %s\n", attempt.Number, displayValue(attempt.ID), displayValue(string(attempt.Status))); err != nil {
			return err
		}
		if err := writeAttemptDetails(buffered, attempt); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func writeAttemptDetails(writer io.Writer, attempt dispatchcontract.Attempt) error {
	if attempt.Resolved != nil {
		resolved := attempt.Resolved
		if _, err := fmt.Fprintln(writer, "    Resolved execution:"); err != nil {
			return err
		}
		fields := []struct{ label, value string }{
			{"profile", string(resolved.Profile)},
			{"agent", resolved.Agent},
			{"model", resolved.Model},
			{"effort", resolved.Effort},
			{"mapping-version", resolved.MappingVersion},
			{"mapping-reason", resolved.RoutingReason},
		}
		for _, field := range fields {
			if _, err := fmt.Fprintf(writer, "      %-16s %s\n", field.label+":", displayLine(field.value)); err != nil {
				return err
			}
		}
	}
	if attempt.Result != nil {
		result := attempt.Result
		if _, err := fmt.Fprintln(writer, "    Result:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      branch:          %s\n", displayLine(result.Branch)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      commit:          %s\n", displayLine(result.Commit)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      summary:         %s\n", displayLine(result.Summary)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, "      validation:"); err != nil {
			return err
		}
		if len(result.Validation) == 0 {
			if _, err := fmt.Fprintln(writer, "        none"); err != nil {
				return err
			}
		}
		for _, evidence := range result.Validation {
			if _, err := fmt.Fprintf(writer, "        - %s: %s (exit %d)\n", displayLine(evidence.Name), displayLine(string(evidence.Status)), evidence.ExitCode); err != nil {
				return err
			}
			if evidence.Command != "" {
				if _, err := fmt.Fprintf(writer, "          command:  %s\n", displayLine(evidence.Command)); err != nil {
					return err
				}
			}
			if evidence.Summary != "" {
				if _, err := fmt.Fprintf(writer, "          summary:  %s\n", displayLine(evidence.Summary)); err != nil {
					return err
				}
			}
			if evidence.ArtifactRef != "" {
				if _, err := fmt.Fprintf(writer, "          artifact: %s\n", displayLine(evidence.ArtifactRef)); err != nil {
					return err
				}
			}
		}
	}
	if attempt.Failure != nil {
		failure := attempt.Failure
		if _, err := fmt.Fprintln(writer, "    Failure:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      stage:           %s\n", displayLine(failure.Stage)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      code:            %s\n", displayLine(failure.Code)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      message:         %s\n", displayLine(failure.Message)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "      retryable:       %t\n", failure.Retryable); err != nil {
			return err
		}
		if failure.Logs == nil {
			if _, err := fmt.Fprintln(writer, "      logs:            -"); err != nil {
				return err
			}
		} else {
			logs := failure.Logs
			if _, err := fmt.Fprintln(writer, "      logs:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "        href:          %s\n", displayLine(logs.Href)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "        stream-id:     %s\n", displayLine(logs.StreamID)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "        session-id:    %s\n", displayLine(logs.SessionID)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "        last-sequence: %d\n", logs.LastSequence); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeDispatchText(writer io.Writer, dispatch dispatchcontract.Dispatch) error {
	if _, err := fmt.Fprintln(writer, "Dispatch:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  dispatch-id: %s\n", displayValue(dispatch.ID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  status:      %s\n", displayValue(string(dispatch.Status))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  updated-at:  %s\n", formatTime(dispatch.UpdatedAt)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  attempt-id:  %s\n", displayValue(dispatch.ActiveAttemptID)); err != nil {
		return err
	}
	return nil
}

func writeLogsText(writer io.Writer, response dispatchcontract.GetLogsResponse) error {
	buffered := bufio.NewWriter(writer)
	if _, err := fmt.Fprintln(buffered, "Logs:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  stream-id:   %s\n", displayValue(response.Reference.StreamID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  next-cursor: %d\n", response.NextCursor); err != nil {
		return err
	}
	for _, event := range response.Events {
		stage := displayValue(event.Stage)
		message := strings.ReplaceAll(event.Message, "\n", "\\n")
		if _, err := fmt.Fprintf(buffered, "%s\t%s\t%s\t%s\n", formatTime(event.Timestamp), event.Level, stage, message); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func writeInvocationLogsText(writer io.Writer, resolved resolvedOutput, response dispatchcontract.GetLogsResponse) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nLogs:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  stream-id:   %s\n", displayValue(response.Reference.StreamID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  next-cursor: %d\n", response.NextCursor); err != nil {
		return err
	}
	for _, event := range response.Events {
		if _, err := fmt.Fprintf(buffered, "%s\t%s\t%s\n", formatTime(event.Timestamp), event.Level, displayValue(event.Stage)); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func writeInvocationRetryText(writer io.Writer, resolved resolvedOutput, dispatch invocationDispatchOutput, attempt invocationAttemptOutput) error {
	return writeInvocationStatusText(writer, resolved, dispatch, []invocationAttemptOutput{attempt})
}

func writeInvocationMutationText(writer io.Writer, resolved resolvedOutput, dispatch invocationDispatchOutput) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered); err != nil {
		return err
	}
	if err := writeInvocationDispatchText(buffered, dispatch); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeRetryText(writer io.Writer, response dispatchcontract.RetryDispatchResponse) error {
	buffered := bufio.NewWriter(writer)
	if err := writeDispatchText(buffered, response.Dispatch); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nRetry:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  attempt-id: %s\n", displayValue(response.Attempt.ID)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  number:     %d\n", response.Attempt.Number); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  status:     %s\n", displayValue(string(response.Attempt.Status))); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeCancelText(writer io.Writer, response dispatchcontract.CancelDispatchResponse) error {
	return writeDispatchText(writer, response.Dispatch)
}

func writeErrorJSON(writer io.Writer, resolved resolvedOutput, err error) error {
	commandErr := asCommandError(err)
	return writeJSON(writer, createOutput{Resolved: resolved, Error: &commandErr.apiError})
}

func writeCreateErrorText(writer io.Writer, resolved resolvedOutput, commandErr *commandError) error {
	buffered := bufio.NewWriter(writer)
	if err := writeResolvedText(buffered, resolved); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buffered, "\nError:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  code:    %s\n", commandErr.apiError.Code); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buffered, "  message: %s\n", commandErr.apiError.Message); err != nil {
		return err
	}
	return buffered.Flush()
}

func displayValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func displayLine(value string) string {
	value = strings.NewReplacer("\r", "\\r", "\n", "\\n", "\t", "\\t").Replace(value)
	return displayValue(value)
}

func displayRunner(value string) string {
	if strings.TrimSpace(value) == "" {
		return "any"
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339Nano)
}
