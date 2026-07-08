// Copyright 2025 The Kubeflow Authors
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

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestNewRunLogServer(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)
	assert.NotNil(t, server)
	assert.NotNil(t, server.resourceManager)
	assert.NotNil(t, server.httpClient)
}

func TestRunLogServer_writeErrorToResponse(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)

	recorder := httptest.NewRecorder()
	server.writeErrorToResponse(recorder, http.StatusBadRequest, assert.AnError)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var errorResponse api.Error
	err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	assert.Nil(t, err)
	assert.Contains(t, errorResponse.ErrorMessage, assert.AnError.Error())
}

func TestReadRunLogV1_MissingRunId(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)

	// URL path is irrelevant here — mux.SetURLVars overrides variable extraction.
	req := httptest.NewRequest("GET", "/test", nil)
	req = mux.SetURLVars(req, map[string]string{})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), RunKey)
}

func TestReadRunLogV1_MissingNodeId(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)

	// URL path is irrelevant here — mux.SetURLVars overrides variable extraction.
	req := httptest.NewRequest("GET", "/test", nil)
	req = mux.SetURLVars(req, map[string]string{
		RunKey: "some-run-id",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), NodeKey)
}

func TestReadRunLogV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, _, run := initWithOneTimeRun(t)
	defer clients.Close()

	// Deny access for the log read while keeping the seeded run intact, so the
	// request reaches the authorization check rather than failing earlier.
	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := NewRunLogServer(manager)

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "node-1",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Failed to authorize")
}
