package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  dispatch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra-servers/pkg/dispatch-contract"
)

type resolvedOutput struct {
	Operation          string                               `json:"operation"`
	DispatchID         string                               `json:"dispatch_id,omitempty"`
	Cursor             int64                                `json:"cursor,omitempty"`
	Source             *dispatchcontract.DispatchSource     `json:"source,omitempty"`
	Repository         *dispatchcontract.RepositorySnapshot `json:"repository,omitempty"`
	RequestedExecution *dispatchcontract.RequestedExecution `json:"requested_execution,omitempty"`
	Runner             string                               `json:"runner,omitempty"`
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
	}
	return buffered.Flush()
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
