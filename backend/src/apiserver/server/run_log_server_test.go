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

	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/gorilla/mux"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	var errorResponse apiError
	err := json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	assert.Nil(t, err)
	assert.Contains(t, errorResponse.ErrorMessage, assert.AnError.Error())
}

func TestReadRunLog_MissingRunId(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)

	// URL path is irrelevant here — mux.SetURLVars overrides variable extraction.
	req := httptest.NewRequest("GET", "/test", nil)
	req = mux.SetURLVars(req, map[string]string{})

	recorder := httptest.NewRecorder()
	server.ReadRunLog(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), RunKey)
}

func TestReadRunLog_MissingNodeId(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewRunLogServer(manager)

	// URL path is irrelevant here — mux.SetURLVars overrides variable extraction.
	req := httptest.NewRequest("GET", "/test", nil)
	req = mux.SetURLVars(req, map[string]string{
		RunKey: "some-run-id",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLog(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), NodeKey)
}

func TestReadRunLog_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, _, run := initWithOneTimeRun(t)
	defer clients.Close()

	// Deny access for the log read while keeping the seeded run intact, so the
	// request reaches the authorization check rather than failing earlier.
	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v2beta1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "node-1",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLog(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Equal(t, "application/json", recorder.Result().Header.Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "Check if you have access to namespace")
}

// Shared read mode auto-approves the get and list verbs, so log reads use the
// dedicated readLog verb to stay behind a SubjectAccessReview. This pins that:
// an unauthorized caller must still get a 403 with shared read enabled.
func TestReadRunLog_SharedReadModeStillDenied(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.MultiUserModeSharedReadAccess, "true")
	defer viper.Set(common.MultiUserModeSharedReadAccess, "false")

	clients, _, run := initWithOneTimeRun(t)
	defer clients.Close()

	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v2beta1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "node-1",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLog(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Check if you have access to namespace")
}

// A pod whose run id label does not match the requested run is rejected inside
// ReadLog, after the handler has already started its response. This pins that
// the handler does not commit a 200 before that validation: the failure must
// still surface to the client as a non-2xx JSON response.
func TestReadRunLog_PodNotFromRun(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/apis/v2beta1/runs/"+run.UUID+"/nodes/victim-pod/log", nil)
	req = mux.SetURLVars(req, map[string]string{
		RunKey:  run.UUID,
		NodeKey: "victim-pod",
	})

	recorder := httptest.NewRecorder()
	server.ReadRunLog(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "application/json", recorder.Result().Header.Get("Content-Type"))
	assert.Contains(t, recorder.Body.String(), "Failed to read logs for run "+run.UUID)
}

func TestReadRunLog_AuthorizedOverPlainHTTP(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, run := initWithOneTimeRun(t) // default SAR fake authorizes
	defer clients.Close()
	server := NewRunLogServer(manager)

	req := httptest.NewRequest("GET", "/apis/v2beta1/runs/"+run.UUID+"/nodes/node-1/log", nil)
	req.Header.Set(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+"user@google.com")

	assert.NoError(t, server.authorize(req, run.UUID))
}

type logTestKubernetesCore struct {
	kubernetes.Interface
}

func (c *logTestKubernetesCore) PodClient(namespace string) typedcorev1.PodInterface {
	return c.CoreV1().Pods(namespace)
}

func (c *logTestKubernetesCore) GetClientSet() kubernetes.Interface {
	return c.Interface
}

func TestReadRunLog_ForwardsFollowQueryToPodLogs(t *testing.T) {
	for _, tc := range []struct {
		query      string
		wantFollow bool
	}{
		{query: "", wantFollow: false},
		{query: "?follow=false", wantFollow: false},
		{query: "?follow=true", wantFollow: true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			logRequests := make(chan string, 1)
			kubeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/namespaces/ns1/pods/node-1":
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(&corev1.Pod{
						TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
						ObjectMeta: metav1.ObjectMeta{Name: "node-1", Namespace: "ns1", Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID}},
					})
				case "/api/v1/namespaces/ns1/pods/node-1/log":
					logRequests <- r.URL.Query().Get("follow")
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write([]byte("IR run log\n"))
				default:
					http.NotFound(w, r)
				}
			}))
			defer kubeServer.Close()
			kubeClient, err := kubernetes.NewForConfig(&rest.Config{Host: kubeServer.URL})
			require.NoError(t, err)
			clients.KubernetesCoreClientFake = &logTestKubernetesCore{Interface: kubeClient}
			manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{})
			server := NewRunLogServer(manager)
			router := mux.NewRouter()
			router.HandleFunc("/apis/v2beta1/runs/{run_id}/nodes/{node_id}/log", server.ReadRunLog).Methods(http.MethodGet)
			req := httptest.NewRequest(http.MethodGet, "/apis/v2beta1/runs/"+run.UUID+"/nodes/node-1/log"+tc.query, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, "IR run log\n", response.Body.String())
			require.Equal(t, tc.wantFollow, response.Flushed)
			require.Equal(t, "text/plain", response.Header().Get("Content-Type"))
			require.Equal(t, "no-cache, private", response.Header().Get("Cache-Control"))
			select {
			case follow := <-logRequests:
				require.Equal(t, tc.wantFollow, follow == "true")
			default:
				t.Fatal("the request did not reach pod logs")
			}
		})
	}
}
