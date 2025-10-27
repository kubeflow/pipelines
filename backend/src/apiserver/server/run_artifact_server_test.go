// Copyright 2024 The Kubeflow Authors
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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestStreamArtifactV1_Succeed(t *testing.T) {
	expectedContent := "test artifact content"
	filePath := "test/artifact.txt"

	// Setup test data
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()
	resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)

	// Create workflow with artifact
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(run.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				"node-1": {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: "artifact-1",
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: filePath,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	// Create HTTP test server
	runArtifactServer := NewRunArtifactServer(manager)

	// Setup HTTP request
	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:stream", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	// Setup mux variables for the request
	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	runArtifactServer.StreamArtifactV1(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"artifact-1\"", rr.Header().Get("Content-Disposition"))

	// Read and verify response body
	responseBody, err := io.ReadAll(rr.Body)
	assert.Nil(t, err)
	assert.Equal(t, expectedContent, string(responseBody))
}

func TestStreamArtifactV1_RunNotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	// Create HTTP test server
	runArtifactServer := NewRunArtifactServer(resourceManager)

	// Setup HTTP request with non-existent run ID
	url := "/apis/v1beta1/runs/non-existent-run-id/nodes/node-1/artifacts/artifact-1:stream"
	req := httptest.NewRequest("GET", url, nil)

	// Setup mux variables for the request
	req = mux.SetURLVars(req, map[string]string{
		"run_id":        "non-existent-run-id",
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	runArtifactServer.StreamArtifactV1(rr, req)

	// Verify response - should return an error (not 200)
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestStreamArtifactV1_ArtifactNotFound(t *testing.T) {
	// Setup test data without adding the artifact file
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()

	// Create workflow with artifact reference but no actual file
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: run.UUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(run.UUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				"node-1": {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: "artifact-1",
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: "test/nonexistent.txt",
									},
								},
							},
						},
					},
				},
			},
		},
	})
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	assert.Nil(t, err)

	// Create HTTP test server
	runArtifactServer := NewRunArtifactServer(manager)

	// Setup HTTP request
	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:stream", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	// Setup mux variables for the request
	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	runArtifactServer.StreamArtifactV1(rr, req)

	// Verify response - should return an error for missing artifact
	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestStreamArtifactV1_MissingParameters(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	// Create HTTP test server
	runArtifactServer := NewRunArtifactServer(resourceManager)

	testCases := []struct {
		name        string
		vars        map[string]string
		expectedMsg string
	}{
		{
			name:        "Missing run_id",
			vars:        map[string]string{"node_id": "node-1", "artifact_name": "artifact-1"},
			expectedMsg: "missing path parameter: 'run_id'",
		},
		{
			name:        "Missing node_id",
			vars:        map[string]string{"run_id": "run-1", "artifact_name": "artifact-1"},
			expectedMsg: "missing path parameter: 'node_id'",
		},
		{
			name:        "Missing artifact_name",
			vars:        map[string]string{"run_id": "run-1", "node_id": "node-1"},
			expectedMsg: "missing path parameter: 'artifact_name'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup HTTP request
			req := httptest.NewRequest("GET", "/test", nil)
			req = mux.SetURLVars(req, tc.vars)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			runArtifactServer.StreamArtifactV1(rr, req)

			// Verify response
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

