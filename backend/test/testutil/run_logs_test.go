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

package testutil

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureRunLogTest(t *testing.T, address string) {
	t.Helper()
	apiURL, token, ca, skipTLS := *config.ApiUrl, *config.AuthToken, *config.CaCertPath, *config.DisableTLSCheck
	kubeflow, multiUser := *config.KubeflowMode, *config.MultiUserMode
	t.Cleanup(func() {
		*config.ApiUrl, *config.AuthToken, *config.CaCertPath, *config.DisableTLSCheck = apiURL, token, ca, skipTLS
		*config.KubeflowMode, *config.MultiUserMode = kubeflow, multiUser
	})
	*config.ApiUrl, *config.AuthToken, *config.CaCertPath, *config.DisableTLSCheck = address, "task-token", "", false
	*config.KubeflowMode, *config.MultiUserMode = false, false
}

func TestReadRunLogUsesScopedTaskRouteAndAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/apis/v2/runs/run%2Fid/nodes/task%2Fid/log", request.URL.EscapedPath())
		assert.Equal(t, "false", request.URL.Query().Get("follow"))
		assert.Equal(t, "Bearer task-token", request.Header.Get("Authorization"))
		_, _ = writer.Write([]byte("retained task output"))
	}))
	defer server.Close()
	configureRunLogTest(t, server.URL)
	logs, err := ReadRunLog(context.Background(), "run/id", "task/id", "task-token")
	require.NoError(t, err)
	require.Equal(t, "retained task output", logs)
}

func TestReadRunLogReportsErrorsAndRejectsRedirects(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.NotEqual(t, "/login", request.URL.Path)
				writer.Header().Set("Location", "/login")
				writer.WriteHeader(status)
				_, _ = writer.Write([]byte("log retrieval failed"))
			}))
			defer server.Close()
			configureRunLogTest(t, server.URL)
			logs, err := ReadRunLog(context.Background(), "run", "task", "task-token")
			require.ErrorContains(t, err, http.StatusText(status))
			require.Equal(t, "log retrieval failed", logs)
		})
	}
}

func TestReadRunLogTrustsConfiguredAPICertificate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("TLS task output"))
	}))
	defer server.Close()
	configureRunLogTest(t, server.URL)
	caPath := filepath.Join(t.TempDir(), "api-ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE", Bytes: server.Certificate().Raw,
	}), 0600))
	*config.CaCertPath = caPath
	logs, err := ReadRunLog(context.Background(), "run", "task", "task-token")
	require.NoError(t, err)
	require.Equal(t, "TLS task output", logs)
}

func TestReadRunLogHonorsCancellation(t *testing.T) {
	configureRunLogTest(t, "http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ReadRunLog(ctx, "run", "task", "")
	require.ErrorIs(t, err, context.Canceled)
	_, err = ReadRunLog(context.Background(), "run", "", "")
	require.ErrorContains(t, err, "IDs are required")
}

func TestReadRunLogUsesGeneratedSuiteTokenWithoutAuthFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "Bearer generated-suite-token", request.Header.Get("Authorization"))
		_, _ = writer.Write([]byte("generated-token task logs"))
	}))
	defer server.Close()
	configureRunLogTest(t, server.URL)
	*config.AuthToken = ""
	*config.MultiUserMode = true
	logs, err := ReadRunLog(context.Background(), "run", "task", "generated-suite-token")
	require.NoError(t, err)
	require.Equal(t, "generated-token task logs", logs)
}

func TestReadRunLogRefreshesProjectedInClusterToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.Header.Get("Authorization")))
	}))
	defer server.Close()
	configureRunLogTest(t, server.URL)
	*config.KubeflowMode = true
	tokenPath := filepath.Join(t.TempDir(), "projected-token")
	t.Setenv("KF_PIPELINES_SA_TOKEN_PATH", tokenPath)
	for _, token := range []string{"initial-token", "rotated-token"} {
		require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0600))
		logs, err := ReadRunLog(context.Background(), "run", "task", "not-the-projected-token")
		require.NoError(t, err)
		require.Equal(t, "Bearer "+token, logs)
	}
	require.NoError(t, os.Remove(tokenPath))
	_, err := ReadRunLog(context.Background(), "run", "task", "")
	require.ErrorContains(t, err, "projected SA token")
}
