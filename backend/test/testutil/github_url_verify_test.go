// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadWithGitHubAuthAndRetry_RetriesThenSucceeds(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First two attempts are rate limited, the third succeeds.
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := headWithGitHubAuthAndRetry(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "should have retried the 429s")
}

func TestHeadWithGitHubAuthAndRetry_SetsBearerTokenWhenPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token-123")
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := headWithGitHubAuthAndRetry(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "Bearer test-token-123", gotAuth)
}

func TestHeadWithGitHubAuthAndRetry_NoTokenNoHeader(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	sawAuth := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := headWithGitHubAuthAndRetry(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.False(t, sawAuth, "no Authorization header should be sent without GITHUB_TOKEN")
}

func TestHeadWithGitHubAuthAndRetry_DoesNotRetryNotFound(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := headWithGitHubAuthAndRetry(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "a genuine 404 must not be retried")
}
