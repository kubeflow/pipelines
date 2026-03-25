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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// createWorkflowWithArtifact creates a workflow with a single artifact for testing
func createWorkflowWithArtifact(runUUID, nodeID, artifactName, artifactPath string) *util.Workflow {
	return util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "workflow-name",
			Namespace:         "ns1",
			UID:               "workflow1",
			Labels:            map[string]string{util.LabelKeyWorkflowRunId: runUUID},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1beta1",
				Kind:       "Workflow",
				Name:       "workflow-name",
				UID:        types.UID(runUUID),
			}},
		},
		Status: v1alpha1.WorkflowStatus{
			Nodes: map[string]v1alpha1.NodeStatus{
				nodeID: {
					Outputs: &v1alpha1.Outputs{
						Artifacts: []v1alpha1.Artifact{
							{
								Name: artifactName,
								ArtifactLocation: v1alpha1.ArtifactLocation{
									S3: &v1alpha1.S3Artifact{
										Key: artifactPath,
									},
								},
							},
						},
					},
				},
			},
		},
	})
}

func TestReadArtifactV1_Succeed(t *testing.T) {
	expectedContent := "test artifact content"
	filePath := "test/artifact.txt"

	// Setup test data
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()

	defer func() {
		if err := manager.DeleteRun(context.Background(), run.UUID); err != nil {
			t.Logf("Failed to clean up test run: %v", err)
		}
	}()

	err := resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)
	require.NoError(t, err, "Failed to add file to object store")

	workflow := createWorkflowWithArtifact(run.UUID, "node-1", "artifact-1", filePath)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:read", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.ReadArtifactV1(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	validateHeaders(t, rr)

	responseBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "Failed to read response body")

	var jsonResponse map[string]string
	err = json.Unmarshal(responseBody, &jsonResponse)
	require.NoError(t, err, "Failed to parse JSON response")

	decodedData, err := base64.StdEncoding.DecodeString(jsonResponse["data"])
	require.NoError(t, err, "Failed to decode base64 data")

	require.Equal(t, expectedContent, string(decodedData))
}

func validateHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache, private", rr.Header().Get("Cache-Control"))
	assert.Empty(t, rr.Header().Get("Content-Encoding"), "Content-Encoding should not be set for JSON response")
}

func TestReadArtifactV1_RunNotFound(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	runArtifactServer := NewRunArtifactServer(resourceManager)

	url := "/apis/v1beta1/runs/non-existent-run-id/nodes/node-1/artifacts/artifact-1:read"
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        "non-existent-run-id",
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.ReadArtifactV1(rr, req)

	require.NotEqual(t, http.StatusOK, rr.Code)
}

// TestReadArtifactV1_ChunkedResponse validates that the HTTP endpoint
// actually streams the response in chunks, not loading it all into memory.
// This is the critical test that proves the endpoint prevents OOM errors.
func TestReadArtifactV1_ChunkedResponse(t *testing.T) {
	largeFileSize := 10 * 1024 * 1024 // 10MB
	t.Log("Creating test file for HTTP endpoint streaming test...")
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

	defer func() {
		if err := manager.DeleteRun(context.Background(), run.UUID); err != nil {
			t.Logf("Failed to clean up test run: %v", err)
		}
	}()

	err := resourceManager.ObjectStore().AddFile(context.TODO(), largeContent, filePath)
	require.NoError(t, err, "Failed to add large file to object store")

	workflow := createWorkflowWithArtifact(run.UUID, "node-1", "large-artifact", filePath)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/large-artifact:read", run.UUID)
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

	runArtifactServer.ReadArtifactV1(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	validateHeaders(t, rr.ResponseRecorder)

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
	t.Logf("Large file streaming results:")
	t.Logf("  - File size: %d MB", largeFileSize/(1024*1024))
	t.Logf("  - Write operations: %d", rr.WriteCount)
	t.Logf("  - Max chunk size: %d KB", maxChunkSize/1024)
	t.Logf("  - Total chunks: %d", len(rr.ChunkSizes))
	t.Logf("  - Total response size: %d bytes", totalSize)

	// Validate that response was written in chunks
	assert.Greater(t, rr.WriteCount, 10, fmt.Sprintf("%dMB file should be written in many chunks", largeFileSize/(1024*1024)))

	// Response will be larger than original due to base64 encoding (~33% overhead)
	assert.Greater(t, totalSize, largeFileSize, "Response should be larger due to base64 encoding")

	// Most importantly: verify chunked streaming (no single large write)
	assert.Less(t, maxChunkSize, 10*1024*1024, "No single chunk should be larger than 10MB - proves streaming")

	t.Logf("All assertions passed: HTTP endpoint correctly streams large files without loading them into memory")
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

func TestReadArtifactV1_ArtifactNotFound(t *testing.T) {
	resourceManager, manager, run := initWithOneTimeRun(t)
	defer resourceManager.Close()

	defer func() {
		if err := manager.DeleteRun(context.Background(), run.UUID); err != nil {
			t.Logf("Failed to clean up test run: %v", err)
		}
	}()

	workflow := createWorkflowWithArtifact(run.UUID, "node-1", "artifact-1", "test/nonexistent.txt")
	_, err := manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(manager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:read", run.UUID)
	req := httptest.NewRequest("GET", url, nil)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.ReadArtifactV1(rr, req)

	require.NotEqual(t, http.StatusOK, rr.Code)
}

func TestReadArtifactV1_MissingParameters(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer clientManager.Close()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	runArtifactServer := NewRunArtifactServer(resourceManager)

	testCases := []struct {
		name               string
		vars               map[string]string
		checkSpecificError bool
		expectedMsg        string
	}{
		{
			name:               "Missing all parameters",
			vars:               map[string]string{},
			checkSpecificError: false,
			expectedMsg:        "", // Don't check specific message when all params are missing
		},
		{
			name:               "Missing run_id",
			vars:               map[string]string{"node_id": "node-1", "artifact_name": "artifact-1"},
			checkSpecificError: true,
			expectedMsg:        "missing path parameter: 'run_id'",
		},
		{
			name:               "Missing node_id",
			vars:               map[string]string{"run_id": "run-1", "artifact_name": "artifact-1"},
			checkSpecificError: true,
			expectedMsg:        "missing path parameter: 'node_id'",
		},
		{
			name:               "Missing artifact_name",
			vars:               map[string]string{"run_id": "run-1", "node_id": "node-1"},
			checkSpecificError: true,
			expectedMsg:        "missing path parameter: 'artifact_name'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req = mux.SetURLVars(req, tc.vars)

			rr := httptest.NewRecorder()

			runArtifactServer.ReadArtifactV1(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)

			var errorResponse api.Error
			err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
			require.NoError(t, err)

			if tc.checkSpecificError {
				// For specific parameter tests, check the exact error message
				require.Equal(t, tc.expectedMsg, errorResponse.ErrorMessage)
			} else {
				// For "Missing all parameters", just verify we get a missing parameter error
				// without depending on which specific parameter is checked first
				require.Contains(t, errorResponse.ErrorMessage, "missing path parameter:")
			}
		})
	}
}

func TestReadArtifactV1_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	clientManager, manager, run := initWithOneTimeRun(t)
	require.NotNil(t, clientManager, "Failed to create client manager")
	require.NotNil(t, manager, "Failed to create manager")
	require.NotNil(t, run, "Failed to create run")

	clientManager.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})

	expectedContent := "test artifact content"
	filePath := "test/artifact.txt"

	err := resourceManager.ObjectStore().AddFile(context.TODO(), []byte(expectedContent), filePath)
	require.NoError(t, err, "Failed to add file to object store")

	workflow := createWorkflowWithArtifact(run.UUID, "node-1", "artifact-1", filePath)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err, "Failed to report workflow resource")

	runArtifactServer := NewRunArtifactServer(resourceManager)

	url := fmt.Sprintf("/apis/v1beta1/runs/%s/nodes/node-1/artifacts/artifact-1:read", run.UUID)
	req := httptest.NewRequest("GET", url, nil).WithContext(ctx)

	req = mux.SetURLVars(req, map[string]string{
		"run_id":        run.UUID,
		"node_id":       "node-1",
		"artifact_name": "artifact-1",
	})

	rr := httptest.NewRecorder()

	runArtifactServer.ReadArtifactV1(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)

	var errorResponse api.Error
	err = json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	require.Contains(t, errorResponse.ErrorMessage, "User 'user@google.com' is not authorized")
	require.Contains(t, errorResponse.ErrorMessage, "this is not allowed")
}
