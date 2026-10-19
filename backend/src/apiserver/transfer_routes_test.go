// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferRoutesUseDedicatedHandlers(t *testing.T) {
	for _, path := range []string{"/apis/v2beta1/transfer/export", "/apis/v2beta1/transfer/import"} {
		t.Run(path, func(t *testing.T) {
			called := false
			handle := func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(http.StatusAccepted) }
			deps := newNoOpHTTPRouterDeps()
			deps.ExportTransfer = handle
			deps.ImportTransfer = handle
			gateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("transfer request reached gRPC gateway") })
			router := buildHTTPRouter(deps, gateway, "database")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
			require.True(t, called)
			require.Equal(t, http.StatusAccepted, w.Code)
		})
	}
}
