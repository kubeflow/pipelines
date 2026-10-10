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

package executorplugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/driver/executorplugin"
	"github.com/stretchr/testify/require"
)

func TestHandlerRejectsNilServiceContext(t *testing.T) {
	handler, err := executorplugin.NewHandlerWithContext(nil, "unused", nil)
	require.ErrorContains(t, err, "provide a context tied to the service lifetime")
	require.Nil(t, handler)
}

func TestHandlerAllowsUnownedTemplatesWithoutClusterAccess(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("test-token"), 0600))
	handler, err := executorplugin.NewHandlerWithContext(context.Background(), tokenPath, nil)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/template.execute", strings.NewReader(`{"template":{"plugin":{"another-plugin":{}}}}`))
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "{}", response.Body.String())
}
