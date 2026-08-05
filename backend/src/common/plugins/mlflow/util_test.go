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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOpenRunStatus(t *testing.T) {
	assert.False(t, IsOpenRunStatus("FINISHED"))
	assert.False(t, IsOpenRunStatus("FAILED"))
	assert.False(t, IsOpenRunStatus("KILLED"))
	assert.False(t, IsOpenRunStatus("finished"))
	assert.True(t, IsOpenRunStatus("RUNNING"))
	assert.True(t, IsOpenRunStatus("SCHEDULED"))
	assert.True(t, IsOpenRunStatus(""))
}

func TestCloseOpenChildRuns_NilClient(t *testing.T) {
	errs := CloseOpenChildRuns(context.Background(), nil, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "MLflow client is required")
}

func TestCloseOpenChildRuns_MissingIDs(t *testing.T) {
	c := newTestClient(t, "http://mlflow.example.com")

	errs := CloseOpenChildRuns(context.Background(), c, "", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "experimentID and parentRunID are required")

	errs = CloseOpenChildRuns(context.Background(), c, "exp-1", "", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "experimentID and parentRunID are required")
}

func TestCloseOpenChildRuns_ClosesOnlyOpenDirectChildren(t *testing.T) {
	var updatedRunIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[
				{"info":{"run_id":"child-running","status":"RUNNING"}},
				{"info":{"run_id":"child-finished","status":"FINISHED"}},
				{"info":{"run_uuid":"child-scheduled-uuid","status":"SCHEDULED"}}
			]}`))
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			updatedRunIDs = append(updatedRunIDs, body["run_id"].(string))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Empty(t, errs)
	assert.ElementsMatch(t, []string{"child-running", "child-scheduled-uuid"}, updatedRunIDs)
}

func TestCloseOpenChildRuns_NoOpenChildren_NoUpdates(t *testing.T) {
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"child-finished","status":"FINISHED"}}]}`))
		case pathRunsUpdate:
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Empty(t, errs)
	assert.False(t, updateCalled)
}

func TestCloseOpenChildRuns_SkipsSelfReferencingRun(t *testing.T) {
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"parent-run","status":"RUNNING"}}]}`))
		case pathRunsUpdate:
			updateCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Empty(t, errs)
	assert.False(t, updateCalled)
}

func TestCloseOpenChildRuns_PaginatesAcrossMultiplePages(t *testing.T) {
	var updatedRunIDs []string
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			page++
			w.WriteHeader(http.StatusOK)
			if page == 1 {
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"child-1","status":"RUNNING"}}],"next_page_token":"page-2"}`))
			} else {
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"child-2","status":"RUNNING"}}]}`))
			}
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			updatedRunIDs = append(updatedRunIDs, body["run_id"].(string))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Empty(t, errs)
	assert.Equal(t, 2, page)
	assert.ElementsMatch(t, []string{"child-1", "child-2"}, updatedRunIDs)
}

func TestCloseOpenChildRuns_SearchError_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"boom"}`))
	}))
	defer server.Close()

	c := newTestClientWithFastRetry(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "failed to search child runs")
}

func TestCloseOpenChildRuns_UpdateError_ReportedButOthersStillClosed(t *testing.T) {
	var updatedRunIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[
				{"info":{"run_id":"child-bad","status":"RUNNING"}},
				{"info":{"run_id":"child-good","status":"RUNNING"}}
			]}`))
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			runID := body["run_id"].(string)
			if runID == "child-bad" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"boom"}`))
				return
			}
			updatedRunIDs = append(updatedRunIDs, runID)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClientWithFastRetry(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "failed to close child run child-bad")
	assert.Equal(t, []string{"child-good"}, updatedRunIDs)
}

func TestCloseOpenChildRuns_DecodeError_ReportedButOthersStillClosed(t *testing.T) {
	var updatedRunIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":["not-an-object",{"info":{"run_id":"child-good","status":"RUNNING"}}]}`))
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			updatedRunIDs = append(updatedRunIDs, body["run_id"].(string))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "failed to decode child run payload")
	assert.Equal(t, []string{"child-good"}, updatedRunIDs)
}

func TestCloseOpenChildRuns_BeforeCloseCalledBeforeUpdate(t *testing.T) {
	var events []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[
				{"info":{"run_id":"child-1","status":"RUNNING"}},
				{"info":{"run_id":"child-finished","status":"FINISHED"}}
			]}`))
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			events = append(events, "update:"+body["run_id"].(string))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, func(childRunID string) {
		events = append(events, "beforeClose:"+childRunID)
	})
	require.Empty(t, errs)
	// beforeClose runs for every valid direct child; UpdateRun only for open ones.
	assert.Equal(t, []string{"beforeClose:child-1", "update:child-1", "beforeClose:child-finished"}, events)
}

func TestCloseOpenChildRuns_BeforeCloseTraversesTerminalChildren(t *testing.T) {
	var updatedRunIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
			if strings.Contains(string(body), "terminal-child") {
				_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"orphan-grandchild","status":"RUNNING"}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"terminal-child","status":"FINISHED"}}]}`))
		case pathRunsUpdate:
			body := decodeUpdateRunBody(t, r)
			updatedRunIDs = append(updatedRunIDs, body["run_id"].(string))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, func(childRunID string) {
		if childRunID != "terminal-child" {
			return
		}
		subErrs := CloseOpenChildRuns(context.Background(), c, "exp-1", childRunID, "FAILED", nil, nil)
		require.Empty(t, subErrs)
	})
	require.Empty(t, errs)
	assert.Equal(t, []string{"orphan-grandchild"}, updatedRunIDs)
}

func TestCloseOpenChildRuns_PaginationLimitReached(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == pathRunsSearch {
			page++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[],"next_page_token":"more"}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "pagination limit (10 pages) reached while closing child runs of parent-run")
	assert.Equal(t, maxChildSearchPages, page)
}

func TestCloseOpenChildRuns_NilBeforeClose_NoPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathRunsSearch:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"child-1","status":"RUNNING"}}]}`))
		case pathRunsUpdate:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	require.NotPanics(t, func() {
		errs := CloseOpenChildRuns(context.Background(), c, "exp-1", "parent-run", "FAILED", nil, nil)
		require.Empty(t, errs)
	})
}

// decodeUpdateRunBody decodes a /runs/update request body for assertions.
func decodeUpdateRunBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	rawBody, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rawBody, &body))
	return body
}
