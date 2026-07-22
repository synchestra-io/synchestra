package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  dispatch

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

const (
	exitRunnerNotFound       = 80
	exitNoEligibleWorker     = 81
	exitIncompatibleProtocol = 82
	exitUnauthenticated      = 101
	exitHubUnreachable       = 102
)

// commandError carries both the process exit code and the stable JSON error.
type commandError struct {
	exitCode int
	apiError dispatchcontract.APIError
	cause    error
}

func (e *commandError) Error() string {
	return e.apiError.Message
}

func (e *commandError) ExitCode() int {
	return e.exitCode
}

func (e *commandError) Unwrap() error {
	return e.cause
}

func newCommandError(exitCode int, code, message string) *commandError {
	return &commandError{
		exitCode: exitCode,
		apiError: dispatchcontract.APIError{Code: code, Message: message},
	}
}

func wrapCommandError(exitCode int, code, message string, cause error) *commandError {
	err := newCommandError(exitCode, code, message)
	err.cause = cause
	return err
}

func invalidArgs(message string) *commandError {
	return newCommandError(2, dispatchcontract.CodeInvalidRequest, message)
}

func notFound(message string) *commandError {
	return newCommandError(3, dispatchcontract.CodeNotFound, message)
}

func unexpected(message string, cause error) *commandError {
	return wrapCommandError(10, "UNEXPECTED", message, cause)
}

func asCommandError(err error) *commandError {
	var result *commandError
	if errors.As(err, &result) {
		return result
	}
	return unexpected(err.Error(), err)
}

func mapAPIError(status int, apiError dispatchcontract.APIError) *commandError {
	if apiError.Code == "" {
		apiError.Code = "HUB_ERROR"
	}
	if apiError.Message == "" {
		apiError.Message = fmt.Sprintf("Hub request failed with HTTP %d", status)
	}

	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		apiError.Code = dispatchcontract.CodeUnauthenticated
		return &commandError{exitCode: exitUnauthenticated, apiError: apiError}
	}

	exitCode := 10
	switch apiError.Code {
	case dispatchcontract.CodeConflict:
		exitCode = 1
	case dispatchcontract.CodeInvalidRequest, dispatchcontract.CodeUnsupportedSelector:
		exitCode = 2
	case dispatchcontract.CodeNotFound:
		exitCode = 3
		if apiError.Details["resource"] == "runner" {
			exitCode = exitRunnerNotFound
		}
	case "RUNNER_NOT_FOUND":
		exitCode = exitRunnerNotFound
	case dispatchcontract.CodeInvalidState:
		exitCode = 4
	case dispatchcontract.CodeNoEligibleWorker:
		exitCode = exitNoEligibleWorker
	case dispatchcontract.CodeIncompatibleProtocol:
		exitCode = exitIncompatibleProtocol
		if !strings.Contains(strings.ToLower(apiError.Message), "upgrade") {
			apiError.Message += "; upgrade the older CLI or Hub component"
		}
	case dispatchcontract.CodeUnauthenticated:
		exitCode = exitUnauthenticated
	default:
		switch status {
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			exitCode = 2
		case http.StatusNotFound:
			exitCode = 3
		case http.StatusConflict:
			exitCode = 1
		}
	}

	return &commandError{exitCode: exitCode, apiError: apiError}
}
