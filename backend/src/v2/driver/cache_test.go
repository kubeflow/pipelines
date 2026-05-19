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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetFingerPrintIsDeterministicForEquivalentPVCSets(t *testing.T) {
	opts := common.Options{
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
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"input1": structpb.NewStringValue("value1"),
			},
		},
	}

	fingerprintA, err := getFingerPrint(opts, executorInput, []string{"pvc-b", "pvc-a", "pvc-b"})
	require.NoError(t, err)
	require.NotEmpty(t, fingerprintA)

	fingerprintB, err := getFingerPrint(opts, executorInput, []string{"pvc-a", "pvc-b"})
	require.NoError(t, err)
	require.NotEmpty(t, fingerprintB)

	assert.Equal(t, fingerprintA, fingerprintB)
}

func TestGetFingerPrintsAndIDReturnsEarlyWhenCachingDisabled(t *testing.T) {
	execution := &Execution{
		ExecutorInput: &pipelinespec.ExecutorInput{},
	}
	opts := &common.Options{
		CacheDisabled: true,
		Task: &pipelinespec.PipelineTaskSpec{
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
				EnableCache: true,
			},
		},
	}

	fingerprint, task, err := getFingerPrintsAndID(context.Background(), execution, nil, opts, nil)
	require.NoError(t, err)
	assert.Empty(t, fingerprint)
	assert.Nil(t, task)
}

func TestGetFingerPrintsAndIDReturnsEarlyWhenExecutionWillNotTrigger(t *testing.T) {
	execution := &Execution{
		ExecutorInput: &pipelinespec.ExecutorInput{},
		Condition:     proto.Bool(false),
	}
	opts := &common.Options{
		Task: &pipelinespec.PipelineTaskSpec{
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
				EnableCache: true,
			},
		},
	}

	fingerprint, task, err := getFingerPrintsAndID(context.Background(), execution, nil, opts, nil)
	require.NoError(t, err)
	assert.Empty(t, fingerprint)
	assert.Nil(t, task)
}

func TestGetFingerPrintsAndIDReturnsEarlyWhenTaskCachingDisabled(t *testing.T) {
	execution := &Execution{
		ExecutorInput: &pipelinespec.ExecutorInput{},
	}
	opts := &common.Options{
		Task: &pipelinespec.PipelineTaskSpec{
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{
				EnableCache: false,
			},
		},
	}

	fingerprint, task, err := getFingerPrintsAndID(context.Background(), execution, nil, opts, nil)
	require.NoError(t, err)
	assert.Empty(t, fingerprint)
	assert.Nil(t, task)
}

func TestGetFingerPrintsAndIDReturnsEarlyWhenTaskCachingOptionsMissing(t *testing.T) {
	execution := &Execution{
		ExecutorInput: &pipelinespec.ExecutorInput{},
	}
	opts := &common.Options{
		Task: &pipelinespec.PipelineTaskSpec{},
	}

	fingerprint, task, err := getFingerPrintsAndID(context.Background(), execution, nil, opts, nil)
	require.NoError(t, err)
	assert.Empty(t, fingerprint)
	assert.Nil(t, task)
}

func TestGetFingerPrintsAndIDReturnsNilTaskForCacheMissWithoutAPIError(t *testing.T) {
	execution := &Execution{
		ExecutorInput: &pipelinespec.ExecutorInput{},
	}
	opts := &common.Options{
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
		Namespace: "default",
	}
	api := &fakeCacheLookupAPI{}

	fingerprint, task, err := getFingerPrintsAndID(context.Background(), execution, api, opts, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fingerprint)
	assert.Nil(t, task)
}

type fakeCacheLookupAPI struct{}

func (f *fakeCacheLookupAPI) GetRun(context.Context, *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) CreateTask(context.Context, *apiv2beta1.CreateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) UpdateTask(context.Context, *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) UpdateTasksBulk(context.Context, *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) GetTask(context.Context, *apiv2beta1.GetTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) ListTasks(context.Context, *apiv2beta1.ListTasksRequest) (*apiv2beta1.ListTasksResponse, error) {
	return &apiv2beta1.ListTasksResponse{}, nil
}

func (f *fakeCacheLookupAPI) CreateArtifact(context.Context, *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) CreateArtifactsBulk(context.Context, *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) ListArtifactsByURI(context.Context, string, string) ([]*apiv2beta1.Artifact, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) ListArtifactTasks(context.Context, *apiv2beta1.ListArtifactTasksRequest) (*apiv2beta1.ListArtifactTasksResponse, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) CreateArtifactTask(context.Context, *apiv2beta1.CreateArtifactTaskRequest) (*apiv2beta1.ArtifactTask, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) CreateArtifactTasks(context.Context, *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) GetPipelineVersion(context.Context, *apiv2beta1.GetPipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) FetchPipelineSpecFromRun(context.Context, *apiv2beta1.Run) (*structpb.Struct, error) {
	return nil, nil
}

func (f *fakeCacheLookupAPI) UpdateStatuses(context.Context, *apiv2beta1.Run, *structpb.Struct, *apiv2beta1.PipelineTaskDetail) error {
	return nil
}
