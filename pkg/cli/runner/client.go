package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

const (
	maxHubResponseBytes = 8 << 20
	defaultHubTimeout   = 30 * time.Second
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type dispatchClient struct {
	baseURL *url.URL
	token   string
	http    httpDoer
}

func newDispatchClient(baseURL, token string, doer httpDoer) (*dispatchClient, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, invalidArgs("invalid Hub URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, invalidArgs("Hub URL must use http or https")
	}
	if parsed.User != nil {
		return nil, invalidArgs("Hub URL must not contain inline credentials")
	}
	if doer == nil {
		doer = &http.Client{Timeout: defaultHubTimeout}
	}
	return &dispatchClient{baseURL: parsed, token: token, http: doer}, nil
}

func (c *dispatchClient) Create(ctx context.Context, request dispatchcontract.CreateDispatchRequest) (dispatchcontract.CreateDispatchResponse, error) {
	var response dispatchcontract.CreateDispatchResponse
	err := c.doJSON(ctx, http.MethodPost, c.dispatchesURL(), request, &response)
	if err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.ProtocolVersion); err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.Dispatch.ProtocolVersion); err != nil {
		return response, err
	}
	if response.Dispatch.ID == "" {
		return response, unexpected("Hub create response did not contain a durable dispatch ID", nil)
	}
	return response, nil
}

func (c *dispatchClient) Status(ctx context.Context, dispatchID string) (dispatchcontract.GetDispatchResponse, error) {
	var response dispatchcontract.GetDispatchResponse
	err := c.doJSON(ctx, http.MethodGet, c.dispatchURL(dispatchID, ""), nil, &response)
	if err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.ProtocolVersion); err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.Dispatch.ProtocolVersion); err != nil {
		return response, err
	}
	if response.Dispatch.ID != dispatchID {
		return response, unexpected("Hub status response returned a different dispatch ID", nil)
	}
	for _, attempt := range response.Attempts {
		if err := requireResponseProtocol(attempt.ProtocolVersion); err != nil {
			return response, err
		}
	}
	return response, nil
}

func (c *dispatchClient) Logs(ctx context.Context, dispatchID string, cursor int64) (dispatchcontract.GetLogsResponse, error) {
	endpoint := c.dispatchURL(dispatchID, "logs")
	query := endpoint.Query()
	query.Set("cursor", strconv.FormatInt(cursor, 10))
	endpoint.RawQuery = query.Encode()
	var response dispatchcontract.GetLogsResponse
	err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &response)
	if err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.ProtocolVersion); err != nil {
		return response, err
	}
	return response, nil
}

func (c *dispatchClient) Retry(ctx context.Context, dispatchID string, request dispatchcontract.RetryDispatchRequest) (dispatchcontract.RetryDispatchResponse, error) {
	var response dispatchcontract.RetryDispatchResponse
	err := c.doJSON(ctx, http.MethodPost, c.dispatchURL(dispatchID, "retry"), request, &response)
	if err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.ProtocolVersion); err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.Dispatch.ProtocolVersion); err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.Attempt.ProtocolVersion); err != nil {
		return response, err
	}
	if response.Dispatch.ID != dispatchID || response.Attempt.DispatchID != dispatchID {
		return response, unexpected("Hub retry response returned a different dispatch ID", nil)
	}
	return response, nil
}

func (c *dispatchClient) Cancel(ctx context.Context, dispatchID string, request dispatchcontract.CancelDispatchRequest) (dispatchcontract.CancelDispatchResponse, error) {
	var response dispatchcontract.CancelDispatchResponse
	err := c.doJSON(ctx, http.MethodPost, c.dispatchURL(dispatchID, "cancel"), request, &response)
	if err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.ProtocolVersion); err != nil {
		return response, err
	}
	if err := requireResponseProtocol(response.Dispatch.ProtocolVersion); err != nil {
		return response, err
	}
	if response.Dispatch.ID != dispatchID {
		return response, unexpected("Hub cancel response returned a different dispatch ID", nil)
	}
	return response, nil
}

func (c *dispatchClient) dispatchesURL() *url.URL {
	result := *c.baseURL
	result.Path = strings.TrimRight(result.Path, "/") + "/v1/dispatches"
	return &result
}

func (c *dispatchClient) dispatchURL(dispatchID, operation string) *url.URL {
	result := c.dispatchesURL()
	result.Path += "/" + url.PathEscape(dispatchID)
	if operation != "" {
		result.Path += "/" + operation
	}
	return result
}

func (c *dispatchClient) doJSON(ctx context.Context, method string, endpoint *url.URL, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return unexpected("encode Hub request", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return unexpected("construct Hub request", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("X-Synchestra-Dispatch-Protocol", dispatchcontract.ProtocolVersionV1)
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return wrapCommandError(exitHubUnreachable, "HUB_UNREACHABLE", "Hub is unreachable", err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxHubResponseBytes+1))
	if err != nil {
		return unexpected("read Hub response", err)
	}
	if len(data) > maxHubResponseBytes {
		return unexpected("Hub response exceeds 8 MiB limit", nil)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiError dispatchcontract.APIError
		if len(data) > 0 {
			if err := json.Unmarshal(data, &apiError); err != nil {
				apiError = dispatchcontract.APIError{
					Code:    "HUB_ERROR",
					Message: fmt.Sprintf("Hub request failed with HTTP %d", response.StatusCode),
				}
			}
		}
		return mapAPIError(response.StatusCode, apiError)
	}
	if len(data) == 0 {
		return unexpected("Hub returned an empty response", nil)
	}
	if err := json.Unmarshal(data, responseBody); err != nil {
		return unexpected("decode Hub response", err)
	}
	return nil
}

func requireResponseProtocol(version string) error {
	if err := dispatchcontract.RequireCompatibleProtocol(version); err != nil {
		return wrapCommandError(
			exitIncompatibleProtocol,
			dispatchcontract.CodeIncompatibleProtocol,
			fmt.Sprintf("Hub protocol %q is incompatible with CLI protocol %q; upgrade the older component", version, dispatchcontract.ProtocolVersionV1),
			err,
		)
	}
	return nil
}
