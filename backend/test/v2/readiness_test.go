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

package test

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/stretchr/testify/require"
)

func TestWaitForReadyUsesConfiguredV2Transport(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tls       bool
		trusted   bool
		status    int
		wantError bool
	}{
		{name: "HTTP endpoint", status: http.StatusOK},
		{name: "HTTPS with configured CA", tls: true, trusted: true, status: http.StatusOK},
		{name: "untrusted HTTPS", tls: true, status: http.StatusOK, wantError: true},
		{name: "unavailable is not ready", status: http.StatusServiceUnavailable, wantError: true},
		{name: "missing endpoint is not ready", status: http.StatusNotFound, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previousURL, previousTLS, previousCA := *config.ApiUrl, *config.TLSEnabled, *config.CaCertPath
			previousInCluster, previousSkipVerify := *config.InClusterRun, *config.DisableTLSCheck
			t.Cleanup(func() {
				*config.ApiUrl, *config.TLSEnabled, *config.CaCertPath = previousURL, previousTLS, previousCA
				*config.InClusterRun, *config.DisableTLSCheck = previousInCluster, previousSkipVerify
			})
			requests := make(chan string, 10)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"multi_user":false,"pipeline_store":"database"}`))
			}))
			if tc.tls {
				server.StartTLS()
			} else {
				server.Start()
			}
			t.Cleanup(server.Close)
			*config.ApiUrl, *config.TLSEnabled, *config.CaCertPath = server.URL, tc.tls, ""
			*config.InClusterRun, *config.DisableTLSCheck = false, false
			if tc.trusted {
				*config.CaCertPath = filepath.Join(t.TempDir(), "ca.pem")
				require.NoError(t, os.WriteFile(*config.CaCertPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
			}
			err := WaitForReady(time.Millisecond)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if !tc.tls || tc.trusted {
				select {
				case path := <-requests:
					require.Equal(t, "/apis/v2beta1/healthz", path)
				default:
					t.Fatal("readiness did not contact the configured endpoint")
				}
			}
		})
	}
}
