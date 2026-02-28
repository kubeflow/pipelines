// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/cachekey"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// mockCacheClient implements cacheutils.Client for testing.
type mockCacheClient struct {
	getExecutionCacheFunc    func(fingerPrint, pipelineName, namespace string) (string, error)
	createExecutionCacheFunc func(ctx context.Context, task *api.Task) error
	generateCacheKeyFunc     func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error)
	generateFingerPrintFunc  func(cacheKey *cachekey.CacheKey) (string, error)
}

var _ cacheutils.Client = &mockCacheClient{}

func (m *mockCacheClient) GetExecutionCache(fingerPrint, pipelineName, namespace string) (string, error) {
	if m.getExecutionCacheFunc != nil {
		return m.getExecutionCacheFunc(fingerPrint, pipelineName, namespace)
	}
	return "", nil
}

func (m *mockCacheClient) CreateExecutionCache(ctx context.Context, task *api.Task) error {
	if m.createExecutionCacheFunc != nil {
		return m.createExecutionCacheFunc(ctx, task)
	}
	return nil
}

func (m *mockCacheClient) GenerateCacheKey(
	inputs *pipelinespec.ExecutorInput_Inputs,
	outputs *pipelinespec.ExecutorInput_Outputs,
	outputParametersTypeMap map[string]string,
	cmdArgs []string, image string,
	pvcNames []string,
) (*cachekey.CacheKey, error) {
	if m.generateCacheKeyFunc != nil {
		return m.generateCacheKeyFunc(inputs, outputs, outputParametersTypeMap, cmdArgs, image, pvcNames)
	}
	return &cachekey.CacheKey{}, nil
}

func (m *mockCacheClient) GenerateFingerPrint(cacheKey *cachekey.CacheKey) (string, error) {
	if m.generateFingerPrintFunc != nil {
		return m.generateFingerPrintFunc(cacheKey)
	}
	return "mock-fingerprint", nil
}

func Test_getFingerPrint(t *testing.T) {
	tests := []struct {
		name            string
		opts            Options
		executorInput   *pipelinespec.ExecutorInput
		pvcNames        []string
		mockClient      *mockCacheClient
		wantFingerPrint string
		wantErr         bool
	}{
		{
			name: "basic fingerprint generation",
			opts: Options{
				Component: &pipelinespec.ComponentSpec{
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
							"output1": {ParameterType: pipelinespec.ParameterType_STRING},
						},
					},
				},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "test-image:latest",
					Command: []string{"python", "main.py"},
					Args:    []string{"--flag"},
				},
			},
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"input1": structpb.NewStringValue("value1"),
					},
				},
			},
			pvcNames: nil,
			mockClient: &mockCacheClient{
				generateCacheKeyFunc: func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error) {
					assert.Equal(t, "test-image:latest", image)
					assert.Equal(t, []string{"python", "main.py", "--flag"}, cmdArgs)
					return &cachekey.CacheKey{}, nil
				},
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "abc123", nil
				},
			},
			wantFingerPrint: "abc123",
			wantErr:         false,
		},
		{
			name: "PVC names are deduplicated and sorted",
			opts: Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image:latest",
				},
			},
			executorInput: &pipelinespec.ExecutorInput{},
			pvcNames:      []string{"pvc-b", "pvc-a", "pvc-b", "pvc-c", "pvc-a"},
			mockClient: &mockCacheClient{
				generateCacheKeyFunc: func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error) {
					// Verify PVC names are deduplicated and sorted
					assert.Equal(t, []string{"pvc-a", "pvc-b", "pvc-c"}, pvcNames)
					return &cachekey.CacheKey{}, nil
				},
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "sorted-fingerprint", nil
				},
			},
			wantFingerPrint: "sorted-fingerprint",
			wantErr:         false,
		},
		{
			name: "empty inputs produce valid fingerprint",
			opts: Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image:latest",
				},
			},
			executorInput: &pipelinespec.ExecutorInput{},
			pvcNames:      nil,
			mockClient: &mockCacheClient{
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "empty-fingerprint", nil
				},
			},
			wantFingerPrint: "empty-fingerprint",
			wantErr:         false,
		},
		{
			name: "GenerateCacheKey error propagates",
			opts: Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image:latest",
				},
			},
			executorInput: &pipelinespec.ExecutorInput{},
			mockClient: &mockCacheClient{
				generateCacheKeyFunc: func(inputs *pipelinespec.ExecutorInput_Inputs, outputs *pipelinespec.ExecutorInput_Outputs, outputParametersTypeMap map[string]string, cmdArgs []string, image string, pvcNames []string) (*cachekey.CacheKey, error) {
					return nil, fmt.Errorf("cache key generation failed")
				},
			},
			wantErr: true,
		},
		{
			name: "GenerateFingerPrint error propagates",
			opts: Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image:latest",
				},
			},
			executorInput: &pipelinespec.ExecutorInput{},
			mockClient: &mockCacheClient{
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "", fmt.Errorf("fingerprint generation failed")
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fingerPrint, err := getFingerPrint(test.opts, test.executorInput, test.mockClient, test.pvcNames)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.wantFingerPrint, fingerPrint)
			}
		})
	}
}

func Test_getFingerPrintsAndID(t *testing.T) {
	tests := []struct {
		name            string
		execution       *Execution
		opts            *Options
		mockClient      *mockCacheClient
		pvcNames        []string
		wantFingerPrint string
		wantExecutionID string
		wantErr         bool
	}{
		{
			name: "caching disabled globally returns empty",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				CacheDisabled: true,
				Task: &pipelinespec.PipelineTaskSpec{
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: true,
					},
				},
			},
			mockClient:      &mockCacheClient{},
			wantFingerPrint: "",
			wantExecutionID: "",
			wantErr:         false,
		},
		{
			name: "cache not enabled on task returns empty",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				Task: &pipelinespec.PipelineTaskSpec{
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: false,
					},
				},
			},
			mockClient:      &mockCacheClient{},
			wantFingerPrint: "",
			wantExecutionID: "",
			wantErr:         false,
		},
		{
			name: "nil caching options returns empty",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				Task: &pipelinespec.PipelineTaskSpec{},
			},
			mockClient:      &mockCacheClient{},
			wantFingerPrint: "",
			wantExecutionID: "",
			wantErr:         false,
		},
		{
			name: "cache hit returns fingerprint and execution ID",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "my-task"},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: true,
					},
				},
				PipelineName: "my-pipeline",
				Namespace:    "default",
			},
			mockClient: &mockCacheClient{
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "fingerprint-123", nil
				},
				getExecutionCacheFunc: func(fingerPrint, pipelineName, namespace string) (string, error) {
					assert.Equal(t, "fingerprint-123", fingerPrint)
					assert.Equal(t, "pipeline/my-pipeline", pipelineName)
					assert.Equal(t, "default", namespace)
					return "cached-exec-456", nil
				},
			},
			wantFingerPrint: "fingerprint-123",
			wantExecutionID: "cached-exec-456",
			wantErr:         false,
		},
		{
			name: "cache miss returns fingerprint with empty execution ID",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "my-task"},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: true,
					},
				},
				PipelineName: "my-pipeline",
				Namespace:    "default",
			},
			mockClient: &mockCacheClient{
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "fingerprint-789", nil
				},
				getExecutionCacheFunc: func(fingerPrint, pipelineName, namespace string) (string, error) {
					return "", nil
				},
			},
			wantFingerPrint: "fingerprint-789",
			wantExecutionID: "",
			wantErr:         false,
		},
		{
			name: "execution will not trigger returns empty",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
				Condition:     proto.Bool(false),
			},
			opts: &Options{
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "my-task"},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: true,
					},
				},
			},
			mockClient:      &mockCacheClient{},
			wantFingerPrint: "",
			wantExecutionID: "",
			wantErr:         false,
		},
		{
			name: "GetExecutionCache error propagates",
			execution: &Execution{
				ExecutorInput: &pipelinespec.ExecutorInput{},
			},
			opts: &Options{
				Component: &pipelinespec.ComponentSpec{},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "my-task"},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
						EnableCache: true,
					},
				},
				PipelineName: "my-pipeline",
				Namespace:    "default",
			},
			mockClient: &mockCacheClient{
				generateFingerPrintFunc: func(cacheKey *cachekey.CacheKey) (string, error) {
					return "fingerprint-error-test", nil
				},
				getExecutionCacheFunc: func(fingerPrint, pipelineName, namespace string) (string, error) {
					return "", fmt.Errorf("cache server unavailable")
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fingerPrint, cachedExecutionID, err := getFingerPrintsAndID(test.execution, test.opts, test.mockClient, test.pvcNames)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.wantFingerPrint, fingerPrint)
				assert.Equal(t, test.wantExecutionID, cachedExecutionID)
			}
		})
	}
}

func Test_createCache(t *testing.T) {
	tests := []struct {
		name        string
		execution   *metadata.Execution
		opts        *Options
		fingerPrint string
		mockClient  *mockCacheClient
		wantErr     bool
	}{
		{
			name:      "successful cache creation",
			execution: metadata.NewExecution(&pb.Execution{Id: proto.Int64(123)}),
			opts: &Options{
				PipelineName: "my-pipeline",
				Namespace:    "default",
				RunID:        "run-001",
			},
			fingerPrint: "fingerprint-abc",
			mockClient: &mockCacheClient{
				createExecutionCacheFunc: func(ctx context.Context, task *api.Task) error {
					assert.Equal(t, "pipeline/my-pipeline", task.PipelineName)
					assert.Equal(t, "default", task.Namespace)
					assert.Equal(t, "run-001", task.RunId)
					assert.Equal(t, "123", task.MlmdExecutionID)
					assert.Equal(t, "fingerprint-abc", task.Fingerprint)
					return nil
				},
			},
			wantErr: false,
		},
		{
			name:      "error when execution ID is 0",
			execution: metadata.NewExecution(&pb.Execution{Id: proto.Int64(0)}),
			opts: &Options{
				PipelineName: "my-pipeline",
				Namespace:    "default",
				RunID:        "run-001",
			},
			fingerPrint: "fingerprint-abc",
			mockClient:  &mockCacheClient{},
			wantErr:     true,
		},
		{
			name:      "CreateExecutionCache error propagates",
			execution: metadata.NewExecution(&pb.Execution{Id: proto.Int64(456)}),
			opts: &Options{
				PipelineName: "my-pipeline",
				Namespace:    "default",
				RunID:        "run-001",
			},
			fingerPrint: "fingerprint-def",
			mockClient: &mockCacheClient{
				createExecutionCacheFunc: func(ctx context.Context, task *api.Task) error {
					return fmt.Errorf("rpc error: connection refused")
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := createCache(
				context.Background(),
				test.execution,
				test.opts,
				1000,
				test.fingerPrint,
				test.mockClient,
			)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
