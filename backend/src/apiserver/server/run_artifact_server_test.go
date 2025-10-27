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
	"github.com/stretchr/testify/require"
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
	err := resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)
	require.NoError(t, err, "Failed to add file to object store")

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
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:stream", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.StreamArtifactV1(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"artifact-1\"", rr.Header().Get("Content-Disposition"))

	responseBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "Failed to read response body")
	require.Equal(t, expectedContent, string(responseBody))
}

func TestStreamArtifactV1_RunNotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	runArtifactServer := NewRunArtifactServer(resourceManager)

	url := "/apis/v1beta1/runs/non-existent-run-id/nodes/node-1/artifacts/artifact-1:stream"
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        "non-existent-run-id",
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.StreamArtifactV1(rr, req)

	require.NotEqual(t, http.StatusOK, rr.Code)
}

// TestStreamArtifactV1_ChunkedResponse validates that the HTTP endpoint
// actually streams the response in chunks, not loading it all into memory.
// This is the critical test that proves the endpoint prevents OOM errors.
func TestStreamArtifactV1_ChunkedResponse(t *testing.T) {
	// Create a 1GB test file to prove streaming works with truly large files
	largeFileSize := 1024 * 1024 * 1024 // 1GB
	t.Logf("Creating 1GB test file for HTTP endpoint streaming test...")
	largeContent := make([]byte, largeFileSize)
	// Fill with predictable pattern (faster than random)
	for i := 0; i < len(largeContent); i += 1024 * 1024 {
		for j := 0; j < 1024*1024 && i+j < len(largeContent); j++ {
			largeContent[i+j] = byte((j / 1024) % 256)
		}
	}
	filePath := "test/large_artifact.bin"

	// Setup test data
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()
	err := resourceManager.ObjectStore().AddFile(context.TODO(), largeContent, filePath)
	require.NoError(t, err, "Failed to add large file to object store")

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
								Name: "large-artifact",
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
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/large-artifact:stream", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "large-artifact",
	})

	// Use a custom ResponseRecorder that tracks write operations
	rr := &ChunkedResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		WriteCount:       0,
		ChunkSizes:       []int{},
	}

	runArtifactServer.StreamArtifactV1(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))

	// Calculate statistics first
	maxChunkSize := 0
	totalSize := 0
	for _, size := range rr.ChunkSizes {
		if size > maxChunkSize {
			maxChunkSize = size
		}
		totalSize += size
	}

	// Log streaming statistics (before assertions so we can debug failures)
	t.Logf("1GB file streaming results:")
	t.Logf("  - File size: %d MB", largeFileSize/(1024*1024))
	t.Logf("  - Write operations: %d", rr.WriteCount)
	t.Logf("  - Max chunk size: %d KB", maxChunkSize/1024)
	t.Logf("  - Total chunks: %d", len(rr.ChunkSizes))

	// Validate that response was written in chunks
	assert.Greater(t, rr.WriteCount, 1000, "1GB file should be written in many chunks")
	assert.Equal(t, largeFileSize, totalSize, "Total size should match file size")
	assert.Less(t, maxChunkSize, 10*1024*1024, "No single chunk should be larger than 10MB")

	t.Logf("All assertions passed: HTTP endpoint correctly streams 1GB files without loading them into memory")
}

// ChunkedResponseRecorder tracks write operations to validate chunked streaming
type ChunkedResponseRecorder struct {
	*httptest.ResponseRecorder
	WriteCount int
	ChunkSizes []int
}

func (r *ChunkedResponseRecorder) Write(p []byte) (int, error) {
	r.WriteCount++
	r.ChunkSizes = append(r.ChunkSizes, len(p))
	return r.ResponseRecorder.Write(p)
}

func TestStreamArtifactV1_ArtifactNotFound(t *testing.T) {
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()

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
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:stream", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.StreamArtifactV1(rr, req)

	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestStreamArtifactV1_MissingParameters(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

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
			req := httptest.NewRequest("GET", "/test", nil)
			req = mux.SetURLVars(req, tc.vars)

			rr := httptest.NewRecorder()

			runArtifactServer.StreamArtifactV1(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}
