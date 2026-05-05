// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mlflow provides a shared Go HTTP client for the MLflow REST API.
// It is used by the API server, driver, and launcher to create experiments,
// create/update/search runs, log-batch metrics/params, and set tags.
package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
)

// Auth type constants.
const (
	AuthTypeKubernetes = "kubernetes"
)

// Retry policy defaults.
const (
	DefaultRetryInitial time.Duration = 500 * time.Millisecond
	DefaultRetryMax     time.Duration = 10 * time.Second
	DefaultRetryElapsed time.Duration = 60 * time.Second
)

// MLflow REST API paths.
const (
	pathExperimentsCreate    = "/api/2.0/mlflow/experiments/create"
	pathExperimentsGet       = "/api/2.0/mlflow/experiments/get"
	pathExperimentsGetByName = "/api/2.0/mlflow/experiments/get-by-name"
	pathRunsCreate           = "/api/2.0/mlflow/runs/create"
	pathRunsUpdate           = "/api/2.0/mlflow/runs/update"
	pathRunsSetTag           = "/api/2.0/mlflow/runs/set-tag"
	pathRunsSearch           = "/api/2.0/mlflow/runs/search"
	pathRunsLogBatch         = "/api/2.0/mlflow/runs/log-batch"
)

// Workspace header used when workspace-based multi-tenancy is enabled.
const workspaceHeader = "X-MLflow-Workspace"

// ParentRunTagKey is a key used to store parent run ID as a Tag.
const ParentRunTagKey = "mlflow.parentRunId"

// RetryPolicy configures the exponential backoff for retrying failed requests.
type RetryPolicy struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
	Multiplier      float64
}

// Config holds the configuration for creating a new Client.
type Config struct {
	Endpoint          string
	HTTPClient        *http.Client
	BearerToken       string
	WorkspacesEnabled bool
	Workspace         string
	Retry             RetryPolicy
}

// Param represents a single parameter key-value pair from a run, used in request to POST /api/2.0/mlflow/runs/log-batch
type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Metric represents a scalar metric recorded during a run, used in request to POST /api/2.0/mlflow/runs/log-batch
type Metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int64   `json:"step"`
}

// Tag represents a key-value tag to set on an MLflow run.
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LogBatchRequest represents the request to POST /api/2.0/mlflow/runs/log-batch
type LogBatchRequest struct {
	RunID   string   `json:"run_id"`
	Params  []Param  `json:"params"`
	Metrics []Metric `json:"metrics"`
	Tags    []Tag    `json:"tags"`
}

// MLflowExperiment represents an MLflow experiment as returned by the REST API.
type MLflowExperiment struct {
	ID   string `json:"experiment_id"`
	Name string `json:"name"`
}

// SearchRunsResponse is the parsed response from POST /api/2.0/mlflow/runs/search.
type SearchRunsResponse struct {
	Runs          []json.RawMessage `json:"runs"`
	NextPageToken string            `json:"next_page_token"`
}

// Client is a shared HTTP client for interacting with the MLflow REST API.
type Client struct {
	endpoint          *url.URL
	httpClient        *http.Client
	bearerToken       string
	workspacesEnabled bool
	workspace         string
}

// NewClient creates a new MLflow REST API client.
// The provided HTTPClient's transport is wrapped with a retryRoundTripper
// that retries transient (5xx / network) errors with exponential backoff.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("MLflow endpoint is required")
	}
	u, err := url.Parse(strings.TrimRight(cfg.Endpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid MLflow endpoint %q: %w", cfg.Endpoint, err)
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	// Wrap the transport with retry logic.
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = &retryRoundTripper{
		next:  base,
		retry: cfg.Retry,
	}
	return &Client{
		endpoint:          u,
		httpClient:        httpClient,
		bearerToken:       cfg.BearerToken,
		workspacesEnabled: cfg.WorkspacesEnabled,
		workspace:         cfg.Workspace,
	}, nil
}

// GetExperiment looks up an MLflowExperiment by ID.
func (c *Client) GetExperiment(ctx context.Context, experimentID string) (*MLflowExperiment, error) {
	reqURL := c.buildURL(pathExperimentsGet)
	q := reqURL.Query()
	q.Set("experiment_id", experimentID)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GetExperiment request: %w", err)
	}
	c.applyHeaders(req)

	respBody, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var result struct {
		Experiment MLflowExperiment `json:"experiment"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GetExperiment response: %w", err)
	}
	return &result.Experiment, nil
}

// GetExperimentByName looks up an MLflowExperiment by name.
func (c *Client) GetExperimentByName(ctx context.Context, name string) (*MLflowExperiment, error) {
	reqURL := c.buildURL(pathExperimentsGetByName)
	q := reqURL.Query()
	q.Set("experiment_name", name)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GetExperimentByName request: %w", err)
	}
	c.applyHeaders(req)

	respBody, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var result struct {
		Experiment MLflowExperiment `json:"experiment"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GetExperimentByName response: %w", err)
	}
	return &result.Experiment, nil
}

// CreateExperiment creates a new MLflow experiment. Returns the experiment ID.
func (c *Client) CreateExperiment(ctx context.Context, name string, description *string) (string, error) {
	body := map[string]interface{}{
		"name": name,
	}
	if description != nil {
		body["description"] = *description
	}
	respBody, err := c.postJSON(ctx, pathExperimentsCreate, body)
	if err != nil {
		return "", err
	}
	var result struct {
		ExperimentID string `json:"experiment_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse CreateExperiment response: %w", err)
	}
	return result.ExperimentID, nil
}

// CreateRun creates a new MLflow run under the given experiment.
func (c *Client) CreateRun(ctx context.Context, experimentID, runName string, tags []Tag) (string, error) {
	body := map[string]interface{}{
		"experiment_id": experimentID,
		"run_name":      runName,
		"start_time":    time.Now().UnixMilli(),
	}
	if len(tags) > 0 {
		tagList := make([]map[string]string, len(tags))
		for i, t := range tags {
			tagList[i] = map[string]string{"key": t.Key, "value": t.Value}
		}
		body["tags"] = tagList
	}
	respBody, err := c.postJSON(ctx, pathRunsCreate, body)
	if err != nil {
		return "", err
	}
	var result struct {
		Run struct {
			Info struct {
				RunID string `json:"run_id"`
			} `json:"info"`
		} `json:"run"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse CreateRun response: %w", err)
	}
	return result.Run.Info.RunID, nil
}

// UpdateRun updates the status of an MLflow run.
func (c *Client) UpdateRun(ctx context.Context, runID, status string, endTimeMs *int64) error {
	body := map[string]interface{}{
		"run_id": runID,
		"status": status,
	}
	if endTimeMs != nil {
		body["end_time"] = *endTimeMs
	}
	_, err := c.postJSON(ctx, pathRunsUpdate, body)
	return err
}

// SetTag sets a single tag on an MLflow run.
func (c *Client) SetTag(ctx context.Context, runID, key, value string) error {
	body := map[string]interface{}{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}
	_, err := c.postJSON(ctx, pathRunsSetTag, body)
	return err
}

// SearchRuns searches for runs matching a filter expression.
func (c *Client) SearchRuns(ctx context.Context, experimentIDs []string, filter string, maxResults int, pageToken string) (*SearchRunsResponse, error) {
	body := map[string]interface{}{
		"experiment_ids": experimentIDs,
		"filter":         filter,
		"max_results":    maxResults,
	}
	if pageToken != "" {
		body["page_token"] = pageToken
	}
	respBody, err := c.postJSON(ctx, pathRunsSearch, body)
	if err != nil {
		return nil, err
	}
	var result SearchRunsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SearchRuns response: %w", err)
	}
	return &result, nil
}

// LogBatch logs a batch of metrics, params, and tags to an MLflow run.
func (c *Client) LogBatch(ctx context.Context, request LogBatchRequest) error {
	_, err := c.postJSON(ctx, pathRunsLogBatch, request)
	if err != nil {
		return err
	}
	return nil
}

// APIError represents an error response from the MLflow REST API.
type APIError struct {
	StatusCode int
	ErrorCode  string `json:"error_code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("MLflow API error (HTTP %d, %s): %s", e.StatusCode, e.ErrorCode, e.Message)
}

// IsNotFoundError returns true if the error is an MLflow RESOURCE_DOES_NOT_EXIST error.
func IsNotFoundError(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	return apiErr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" || apiErr.StatusCode == http.StatusNotFound
}

// IsAlreadyExistsError returns true if the error is an MLflow RESOURCE_ALREADY_EXISTS error.
func IsAlreadyExistsError(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	return apiErr.ErrorCode == "RESOURCE_ALREADY_EXISTS"
}

func (c *Client) buildURL(path string) *url.URL {
	u := *c.endpoint
	u.Path = strings.TrimRight(u.Path, "/") + path
	return &u
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	if c.workspacesEnabled && c.workspace != "" {
		req.Header.Set(workspaceHeader, c.workspace)
	}
}

func (c *Client) postJSON(ctx context.Context, path string, body interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body for %s: %w", path, err)
	}
	reqURL := c.buildURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", path, err)
	}
	c.applyHeaders(req)
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MLflow request %s %s failed: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", req.URL.Path, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}
	_ = json.Unmarshal(body, apiErr)
	return nil, apiErr
}

// noRetryPaths lists MLflow API paths that are non-idempotent.  Retrying
// these after a timeout could create duplicate server-side state.
var noRetryPaths = map[string]bool{
	pathRunsCreate:        true,
	pathExperimentsCreate: true,
}

// retryRoundTripper wraps an http.RoundTripper with exponential backoff retry
// logic for transient (5xx and network) errors.  Non-idempotent endpoints
// listed in noRetryPaths are executed exactly once.
type retryRoundTripper struct {
	next  http.RoundTripper
	retry RetryPolicy
}

func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Skip retries for non-idempotent endpoints to avoid duplicates.
	if noRetryPaths[req.URL.Path] {
		return rt.next.RoundTrip(req)
	}

	// Ensure the request body can be replayed across retries.
	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to buffer request body for retry: %w", err)
		}
		req.Body.Close()
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.Body, _ = req.GetBody()
	}

	var resp *http.Response
	operation := func() error {
		// Reset the request body for each attempt.
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return backoff.Permanent(fmt.Errorf("failed to reset request body: %w", err))
			}
			req.Body = body
		}

		var err error
		resp, err = rt.next.RoundTrip(req)
		if err != nil {
			glog.Warningf("MLflow request %s %s failed (will retry): %v", req.Method, req.URL.Path, err)
			return err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		// 4xx — client error, not retryable.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return backoff.Permanent(&roundTripError{statusCode: resp.StatusCode})
		}

		// 5xx — server error, retryable. Drain and close the body so the
		// connection can be reused on the next attempt.
		statusCode := resp.StatusCode
		glog.Warningf("MLflow request %s %s returned %d (will retry)", req.Method, req.URL.Path, statusCode)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		resp = nil
		return fmt.Errorf("server returned %d", statusCode)
	}

	b := backoff.NewExponentialBackOff()
	if rt.retry.InitialInterval > 0 {
		b.InitialInterval = rt.retry.InitialInterval
	}
	if rt.retry.MaxInterval > 0 {
		b.MaxInterval = rt.retry.MaxInterval
	}
	if rt.retry.MaxElapsedTime > 0 {
		b.MaxElapsedTime = rt.retry.MaxElapsedTime
	}
	if rt.retry.Multiplier > 0 {
		b.Multiplier = rt.retry.Multiplier
	}

	if err := backoff.Retry(operation, b); err != nil {
		// If we have a response (4xx case), return it so the caller can parse the error body.
		if resp != nil {
			return resp, nil
		}
		return nil, err
	}
	return resp, nil
}

// roundTripError is a sentinel used by retryRoundTripper to signal a permanent
// (non-retryable) HTTP status to the backoff loop while preserving the response.
type roundTripError struct {
	statusCode int
}

func (e *roundTripError) Error() string {
	return fmt.Sprintf("non-retryable HTTP %d", e.statusCode)
}
