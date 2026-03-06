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

package mlflow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_EmptyEndpoint_ReturnsError(t *testing.T) {
	_, err := NewClient(Config{Endpoint: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

func TestNewClient_InvalidEndpoint_ReturnsError(t *testing.T) {
	_, err := NewClient(Config{Endpoint: "://bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MLflow endpoint")
}

func TestNewClient_ValidEndpoint_Success(t *testing.T) {
	c, err := NewClient(Config{Endpoint: "http://mlflow.example.com"})
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "http", c.endpoint.Scheme)
	assert.Equal(t, "mlflow.example.com", c.endpoint.Host)
}

func TestNewClient_TrailingSlashTrimmed(t *testing.T) {
	c, err := NewClient(Config{Endpoint: "http://mlflow.example.com/"})
	require.NoError(t, err)
	assert.Equal(t, "", c.endpoint.Path)
}

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	c, err := NewClient(Config{Endpoint: "http://mlflow.example.com"})
	require.NoError(t, err)
	require.NotNil(t, c.httpClient)
}

func TestNewClient_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	c, err := NewClient(Config{Endpoint: "http://mlflow.example.com", HTTPClient: custom})
	require.NoError(t, err)
	assert.Equal(t, custom, c.httpClient)
}

func TestApplyHeaders_BearerToken(t *testing.T) {
	c, _ := NewClient(Config{
		Endpoint:    "http://mlflow.example.com",
		BearerToken: "sa-token",
	})
	req, _ := http.NewRequest("GET", "http://mlflow.example.com", nil)
	c.applyHeaders(req)
	assert.Equal(t, "Bearer sa-token", req.Header.Get("Authorization"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestApplyHeaders_NoBearerToken(t *testing.T) {
	c, _ := NewClient(Config{
		Endpoint: "http://mlflow.example.com",
	})
	req, _ := http.NewRequest("GET", "http://mlflow.example.com", nil)
	c.applyHeaders(req)
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestApplyHeaders_WorkspaceHeader(t *testing.T) {
	c, _ := NewClient(Config{
		Endpoint:          "http://mlflow.example.com",
		WorkspacesEnabled: true,
		Workspace:         "my-ns",
	})
	req, _ := http.NewRequest("GET", "http://mlflow.example.com", nil)
	c.applyHeaders(req)
	assert.Equal(t, "my-ns", req.Header.Get(workspaceHeader))
}

func TestApplyHeaders_WorkspaceHeader_DisabledWhenFalse(t *testing.T) {
	c, _ := NewClient(Config{
		Endpoint:          "http://mlflow.example.com",
		WorkspacesEnabled: false,
		Workspace:         "my-ns",
	})
	req, _ := http.NewRequest("GET", "http://mlflow.example.com", nil)
	c.applyHeaders(req)
	assert.Empty(t, req.Header.Get(workspaceHeader))
}

func TestGetExperimentByName_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, pathExperimentsGetByName, r.URL.Path)
		assert.Equal(t, "my-experiment", r.URL.Query().Get("experiment_name"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"42","name":"my-experiment"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	exp, err := c.GetExperimentByName(context.Background(), "my-experiment")
	require.NoError(t, err)
	assert.Equal(t, "42", exp.ID)
	assert.Equal(t, "my-experiment", exp.Name)
}

func TestGetExperimentByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetExperimentByName(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, IsNotFoundError(err))
}

func TestCreateExperiment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, pathExperimentsCreate, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "test-exp", payload["name"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment_id":"99"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	id, err := c.CreateExperiment(context.Background(), "test-exp", nil)
	require.NoError(t, err)
	assert.Equal(t, "99", id)
}

func TestCreateExperiment_WithDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "test-exp", payload["name"])
		assert.Equal(t, "my description", payload["description"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"experiment_id":"100"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	desc := "my description"
	id, err := c.CreateExperiment(context.Background(), "test-exp", &desc)
	require.NoError(t, err)
	assert.Equal(t, "100", id)
}

func TestCreateExperiment_AlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error_code":"RESOURCE_ALREADY_EXISTS","message":"experiment already exists"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.CreateExperiment(context.Background(), "test-exp", nil)
	require.Error(t, err)
	assert.True(t, IsAlreadyExistsError(err))
}

func TestCreateRun_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, pathRunsCreate, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "exp-1", payload["experiment_id"])
		assert.Equal(t, "my-run", payload["run_name"])
		assert.NotNil(t, payload["start_time"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"run-abc"}}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	runID, err := c.CreateRun(context.Background(), "exp-1", "my-run", nil)
	require.NoError(t, err)
	assert.Equal(t, "run-abc", runID)
}

func TestCreateRun_WithTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))

		tags, ok := payload["tags"].([]interface{})
		require.True(t, ok)
		assert.Len(t, tags, 2)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"run-tagged"}}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	tags := []Tag{
		{Key: "k1", Value: "v1"},
		{Key: "k2", Value: "v2"},
	}
	runID, err := c.CreateRun(context.Background(), "exp-1", "tagged-run", tags)
	require.NoError(t, err)
	assert.Equal(t, "run-tagged", runID)
}

func TestCreateRun_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"bad param"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.CreateRun(context.Background(), "exp-1", "fail-run", nil)
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "INVALID_PARAMETER_VALUE", apiErr.ErrorCode)
}

func TestUpdateRun_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, pathRunsUpdate, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "run-1", payload["run_id"])
		assert.Equal(t, "FINISHED", payload["status"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	err := c.UpdateRun(context.Background(), "run-1", "FINISHED", nil)
	require.NoError(t, err)
}

func TestUpdateRun_WithEndTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "run-1", payload["run_id"])
		assert.NotNil(t, payload["end_time"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	endTime := int64(1700000000000)
	err := c.UpdateRun(context.Background(), "run-1", "FAILED", &endTime)
	require.NoError(t, err)
}

func TestSetTag_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, pathRunsSetTag, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "run-1", payload["run_id"])
		assert.Equal(t, "my-key", payload["key"])
		assert.Equal(t, "my-value", payload["value"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	err := c.SetTag(context.Background(), "run-1", "my-key", "my-value")
	require.NoError(t, err)
}

func TestSearchRuns_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, pathRunsSearch, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))

		ids := payload["experiment_ids"].([]interface{})
		assert.Len(t, ids, 1)
		assert.Equal(t, "exp-1", ids[0])
		assert.Equal(t, "tags.status = 'RUNNING'", payload["filter"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"r1"}},{"info":{"run_id":"r2"}}],"next_page_token":"tok2"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	result, err := c.SearchRuns(context.Background(), []string{"exp-1"}, "tags.status = 'RUNNING'", 100, "")
	require.NoError(t, err)
	assert.Len(t, result.Runs, 2)
	assert.Equal(t, "tok2", result.NextPageToken)
}

func TestSearchRuns_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	result, err := c.SearchRuns(context.Background(), []string{"exp-1"}, "", 10, "")
	require.NoError(t, err)
	assert.Empty(t, result.Runs)
	assert.Empty(t, result.NextPageToken)
}

func TestSearchRuns_WithPageToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "page-2", payload["page_token"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.SearchRuns(context.Background(), []string{"exp-1"}, "", 10, "page-2")
	require.NoError(t, err)
}

func TestIsNotFoundError_True(t *testing.T) {
	err := &APIError{StatusCode: 404, ErrorCode: "RESOURCE_DOES_NOT_EXIST", Message: "not found"}
	assert.True(t, IsNotFoundError(err))
}

func TestIsNotFoundError_ByStatusCode(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound, ErrorCode: "UNKNOWN"}
	assert.True(t, IsNotFoundError(err))
}

func TestIsNotFoundError_False(t *testing.T) {
	err := &APIError{StatusCode: 500, ErrorCode: "INTERNAL_ERROR"}
	assert.False(t, IsNotFoundError(err))
}

func TestIsNotFoundError_NonAPIError(t *testing.T) {
	assert.False(t, IsNotFoundError(assert.AnError))
}

func TestIsAlreadyExistsError_True(t *testing.T) {
	err := &APIError{StatusCode: 409, ErrorCode: "RESOURCE_ALREADY_EXISTS"}
	assert.True(t, IsAlreadyExistsError(err))
}

func TestIsAlreadyExistsError_False(t *testing.T) {
	err := &APIError{StatusCode: 409, ErrorCode: "OTHER"}
	assert.False(t, IsAlreadyExistsError(err))
}

func TestIsAlreadyExistsError_NonAPIError(t *testing.T) {
	assert.False(t, IsAlreadyExistsError(assert.AnError))
}

func TestAPIError_ErrorString(t *testing.T) {
	err := &APIError{StatusCode: 400, ErrorCode: "BAD_REQUEST", Message: "invalid param"}
	assert.Equal(t, "MLflow API error (HTTP 400, BAD_REQUEST): invalid param", err.Error())
}

func TestDoWithRetry_RetriesOn5xx(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error_code":"SERVICE_UNAVAILABLE","message":"retry later"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Use UpdateRun (idempotent) to verify that retryable endpoints are retried.
	c := newTestClientWithFastRetry(t, server.URL)
	err := c.UpdateRun(context.Background(), "run-1", "FINISHED", nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(3))
}

func TestDoWithRetry_NoRetryOn4xx(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"bad"}`))
	}))
	defer server.Close()

	// Use SetTag (idempotent, retryable) to verify 4xx stops retries.
	c := newTestClientWithFastRetry(t, server.URL)
	err := c.SetTag(context.Background(), "run-1", "key", "val")
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestDoWithRetry_SkipsRetryForRunsCreate(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error_code":"SERVICE_UNAVAILABLE","message":"retry later"}`))
	}))
	defer server.Close()

	c := newTestClientWithFastRetry(t, server.URL)
	_, err := c.CreateRun(context.Background(), "exp-1", "test-run", nil)
	require.Error(t, err)
	// Non-idempotent endpoint: should be called exactly once, no retries.
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestDoWithRetry_SkipsRetryForExperimentsCreate(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error_code":"SERVICE_UNAVAILABLE","message":"retry later"}`))
	}))
	defer server.Close()

	c := newTestClientWithFastRetry(t, server.URL)
	_, err := c.CreateExperiment(context.Background(), "test-exp", nil)
	require.Error(t, err)
	// Non-idempotent endpoint: should be called exactly once, no retries.
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestBuildURL_AppendsPath(t *testing.T) {
	c, _ := NewClient(Config{Endpoint: "http://mlflow.example.com/prefix"})
	u := c.buildURL("/api/2.0/mlflow/experiments/create")
	assert.Equal(t, "http://mlflow.example.com/prefix/api/2.0/mlflow/experiments/create", u.String())
}

func TestBuildURL_TrimsTrailingSlash(t *testing.T) {
	c, _ := NewClient(Config{Endpoint: "http://mlflow.example.com/prefix/"})
	u := c.buildURL("/api/2.0/mlflow/runs/create")
	assert.Equal(t, "http://mlflow.example.com/prefix/api/2.0/mlflow/runs/create", u.String())
}

// ---- Helpers ----

func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := NewClient(Config{
		Endpoint: endpoint,
		Retry: RetryPolicy{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     5 * time.Millisecond,
			MaxElapsedTime:  50 * time.Millisecond,
		},
	})
	require.NoError(t, err)
	return c
}

func newTestClientWithFastRetry(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := NewClient(Config{
		Endpoint: endpoint,
		Retry: RetryPolicy{
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
			Multiplier:      1.5,
		},
	})
	require.NoError(t, err)
	return c
}

// TestMLflowRuntimeConfig_JSONFieldAlignment verifies that the JSON keys produced by
// marshalling MLflowRuntimeConfig.
func TestMLflowRuntimeConfig_JSONFieldAlignment(t *testing.T) {
	cfg := MLflowRuntimeConfig{
		Endpoint:           "http://mlflow:5000",
		Workspace:          "ns1",
		ParentRunID:        "parent-1",
		ExperimentID:       "exp-1",
		AuthType:           "kubernetes",
		Timeout:            "30s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedKeys := []string{
		"endpoint",
		"workspace",
		"parentRunId",
		"experimentId",
		"authType",
		"timeout",
		"insecureSkipVerify",
		"injectUserEnvVars",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "MLflowRuntimeConfig JSON must contain KEP key %q", key)
	}
	assert.Len(t, raw, len(expectedKeys), "MLflowRuntimeConfig JSON must contain exactly the KEP-defined keys")
}

// TestMLflowRuntimeConfig_OmitEmptyFields verifies that omitempty fields are excluded
// when their zero values are set.
func TestMLflowRuntimeConfig_OmitEmptyFields(t *testing.T) {
	cfg := MLflowRuntimeConfig{
		Endpoint:    "http://mlflow:5000",
		ParentRunID: "parent-1",
		AuthType:    "kubernetes",
	}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	// Fields with omitempty and zero values should be absent.
	assert.NotContains(t, raw, "workspace", "workspace should be omitted when empty")
	assert.NotContains(t, raw, "timeout", "timeout should be omitted when empty")
	assert.NotContains(t, raw, "insecureSkipVerify", "insecureSkipVerify should be omitted when false")
	assert.NotContains(t, raw, "injectUserEnvVars", "injectUserEnvVars should be omitted when false")

	// Required fields should always be present.
	assert.Contains(t, raw, "endpoint")
	assert.Contains(t, raw, "parentRunId")
	assert.Contains(t, raw, "experimentId") // present but empty string
	assert.Contains(t, raw, "authType")
}

// TestMLflowRuntimeConfig_RoundTrip verifies that marshalling and unmarshalling
// MLflowRuntimeConfig produces an identical struct.
func TestMLflowRuntimeConfig_RoundTrip(t *testing.T) {
	original := MLflowRuntimeConfig{
		Endpoint:           "http://mlflow:5000",
		Workspace:          "workspace-1",
		ParentRunID:        "parent-run-123",
		ExperimentID:       "exp-456",
		AuthType:           "kubernetes",
		Timeout:            "15s",
		InsecureSkipVerify: true,
		InjectUserEnvVars:  true,
	}
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded MLflowRuntimeConfig
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, original, decoded)
}
