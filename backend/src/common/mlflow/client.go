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
	DefaultAuthType   = "kubernetes"
	AuthTypeNone      = "none"
	AuthTypeBearer    = "bearer"
	AuthTypeBasicAuth = "basic-auth"
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
	pathExperimentsGetByName = "/api/2.0/mlflow/experiments/get-by-name"
	pathRunsCreate           = "/api/2.0/mlflow/runs/create"
	pathRunsUpdate           = "/api/2.0/mlflow/runs/update"
	pathRunsSetTag           = "/api/2.0/mlflow/runs/set-tag"
	pathRunsSearch           = "/api/2.0/mlflow/runs/search"
	pathRunsLogBatch         = "/api/2.0/mlflow/runs/log-batch"
)

// Workspace header used when workspace-based multi-tenancy is enabled.
const workspaceHeader = "X-MLflow-Workspace"

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
	AuthType          string
	BearerToken       string
	BasicAuthUsername string
	BasicAuthPassword string
	WorkspacesEnabled bool
	Workspace         string
	Retry             RetryPolicy
}

// Tag represents a key-value tag to set on an MLflow run.
type Tag struct {
	Key   string
	Value string
}

// MLflowExperiment represents an MLflow experiment as returned by the REST API.
type MLflowExperiment struct {
	ID   string `json:"experiment_id"`
	Name string `json:"name"`
}

// SearchRunsResponse is the parsed response from POST /api/2.0/mlflow/runs/search.
type SearchRunsResponse struct {
	// Runs are kept as raw JSON so the caller can decode per their needs.
	Runs          []json.RawMessage `json:"runs"`
	NextPageToken string            `json:"next_page_token"`
}

// Client is a shared HTTP client for interacting with the MLflow REST API.
type Client struct {
	endpoint          *url.URL
	httpClient        *http.Client
	authType          string
	bearerToken       string
	basicAuthUsername string
	basicAuthPassword string
	workspacesEnabled bool
	workspace         string
	retry             RetryPolicy
}

// NewClient creates a new MLflow REST API client.
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
	return &Client{
		endpoint:          u,
		httpClient:        httpClient,
		authType:          cfg.AuthType,
		bearerToken:       cfg.BearerToken,
		basicAuthUsername: cfg.BasicAuthUsername,
		basicAuthPassword: cfg.BasicAuthPassword,
		workspacesEnabled: cfg.WorkspacesEnabled,
		workspace:         cfg.Workspace,
		retry:             cfg.Retry,
	}, nil
}

// GetExperimentByName looks up an experiment by name.
// Returns an MLflowExperiment or an error. Returns a not-found error if
// the experiment does not exist.
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

	respBody, err := c.doWithRetry(req)
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
// tags is optional and can be nil.
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
// endTimeMs may be nil if the run is not yet complete.
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
func (c *Client) LogBatch(ctx context.Context, runID string, metrics []map[string]interface{}, params []map[string]string, tags []Tag) error {
	body := map[string]interface{}{
		"run_id": runID,
	}
	if len(metrics) > 0 {
		body["metrics"] = metrics
	}
	if len(params) > 0 {
		body["params"] = params
	}
	if len(tags) > 0 {
		tagList := make([]map[string]string, len(tags))
		for i, t := range tags {
			tagList[i] = map[string]string{"key": t.Key, "value": t.Value}
		}
		body["tags"] = tagList
	}
	_, err := c.postJSON(ctx, pathRunsLogBatch, body)
	return err
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

	switch c.authType {
	case AuthTypeNone:
		// No authentication headers.
	case AuthTypeBearer:
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}
	case AuthTypeBasicAuth:
		if c.basicAuthUsername != "" {
			req.SetBasicAuth(c.basicAuthUsername, c.basicAuthPassword)
		}
	default:
		// DefaultAuthType ("kubernetes") — use bearer token (SA token).
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}
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
	return c.doWithRetry(req)
}

func (c *Client) doWithRetry(req *http.Request) ([]byte, error) {
	var respBody []byte
	operation := func() error {
		// Reset the request body for retries.
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return backoff.Permanent(fmt.Errorf("failed to reset request body: %w", err))
			}
			req.Body = body
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			glog.Warningf("MLflow request %s %s failed (will retry): %v", req.Method, req.URL.Path, err)
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("failed to read response body from %s: %w", req.URL.Path, err))
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			respBody = body
			return nil
		}

		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.Unmarshal(body, apiErr) // best-effort parse

		// Don't retry 4xx (client errors) — they won't succeed on retry.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return backoff.Permanent(apiErr)
		}

		// Retry 5xx.
		glog.Warningf("MLflow request %s %s returned %d (will retry): %s", req.Method, req.URL.Path, resp.StatusCode, apiErr.Message)
		return apiErr
	}

	b := backoff.NewExponentialBackOff()
	if c.retry.InitialInterval > 0 {
		b.InitialInterval = c.retry.InitialInterval
	}
	if c.retry.MaxInterval > 0 {
		b.MaxInterval = c.retry.MaxInterval
	}
	if c.retry.MaxElapsedTime > 0 {
		b.MaxElapsedTime = c.retry.MaxElapsedTime
	}
	if c.retry.Multiplier > 0 {
		b.Multiplier = c.retry.Multiplier
	}

	if err := backoff.Retry(operation, b); err != nil {
		return nil, err
	}
	return respBody, nil
}
