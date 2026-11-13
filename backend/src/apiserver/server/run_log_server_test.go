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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	req := httptest.NewRequest("GET", "/apis/v1alpha1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "node-1",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, "application/json", recorder.Result().Header.Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "Check if you have access to namespace")
}

// Shared read mode auto-approves the get and list verbs, so log reads use the
// dedicated readLog verb to stay behind a SubjectAccessReview. This pins that:
// an unauthorized caller must still get a 403 with shared read enabled.
func TestReadRunLogV1_SharedReadModeStillDenied(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.MultiUserModeSharedReadAccess, "true")
	defer viper.Set(common.MultiUserModeSharedReadAccess, "false")

	clients, _, run := initWithOneTimeRun(t)
	defer clients.Close()

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v1alpha1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "node-1",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Check if you have access to namespace")
}

// A pod whose run id label does not match the requested run is rejected inside
// ReadLog, after the handler has already started its response. This pins that
// the handler does not commit a 200 before that validation: the failure must
// still surface to the client as a non-2xx JSON response.
func TestReadRunLogV1_PodNotFromRun(t *testing.T) {
	clients, _, run := initWithOneTimeRun(t)
	defer clients.Close()

	foreignPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "victim-pod",
			Namespace: "ns1",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "some-other-run"},
		},
	}
	clients.KubernetesCoreClientFake = client.NewFakeKubernetesCoreClientWithPod(foreignPod)
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v1alpha1/runs/"+run.UUID+"/nodes/victim-pod/log", nil)
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "victim-pod",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLogV1(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "application/json", recorder.Result().Header.Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "Failed to read logs for run "+run.UUID)
}

func TestReadRunLogV1_AuthorizedOverPlainHTTP(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, run := initWithOneTimeRun(t) // default SAR fake authorizes
	defer clients.Close()
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v1alpha1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")

	assert.NoError(t, server.authorize(req, run.UUID))
}

func TestRunLogArchiveContextAuthorized(t *testing.T) {
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()
	server := NewRunLogServer(manager)
	req := httptest.NewRequest("GET", "/log?archive_context=true", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")
	req = mux.SetURLVars(req, map[string]string{RunKey: run.UUID, NodeKey: "nonexistent-pod"})
	response := httptest.NewRecorder()
	server.ReadRunLogV1(response, req)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	var identity resource.RunLogArchiveContext
	assert.NoError(t, json.Unmarshal(response.Body.Bytes(), &identity))
	assert.Equal(t, run.UUID, identity.RunID)
	assert.Equal(t, run.Namespace, identity.Namespace)
	assert.Equal(t, run.K8SName, identity.WorkflowName)
	// Imported histories must not produce a local archive authorization context.
	_, err := clients.DB().Exec("UPDATE run_details SET ImportedFrom = ? WHERE UUID = ?", "foreign-installation", run.UUID)
	assert.NoError(t, err)
	response = httptest.NewRecorder()
	server.ReadRunLogV1(response, req)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.NotContains(t, response.Body.String(), "workflowName")
	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	server = NewRunLogServer(resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false}))
	response = httptest.NewRecorder()
	server.ReadRunLogV1(response, req)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.NotContains(t, response.Body.String(), "workflowName")
	assert.Contains(t, response.Body.String(), "Failed to authorize the request")
}
