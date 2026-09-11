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
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_validateRootDAG(t *testing.T) {
	validOpts := func() common.Options {
		return common.Options{
			PipelineName:   "pipeline-1",
			Run:            &apiv2beta1.Run{RunId: "run-1"},
			Component:      &pipelinespec.ComponentSpec{},
			RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
			Namespace:      "default",
			IterationIndex: -1,
		}
	}
	tests := []struct {
		name    string
		opts    common.Options
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing pipeline name returns error",
			opts:    common.Options{IterationIndex: -1},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: common.Options{
				PipelineName:   "pipeline-1",
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "nil component spec returns error",
			opts: common.Options{
				PipelineName:   "pipeline-1",
				Run:            &apiv2beta1.Run{RunId: "run-1"},
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "nil runtime config returns error",
			opts: common.Options{
				PipelineName:   "pipeline-1",
				Run:            &apiv2beta1.Run{RunId: "run-1"},
				Component:      &pipelinespec.ComponentSpec{},
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "runtime config is required",
		},
		{
			name: "missing namespace returns error",
			opts: common.Options{
				PipelineName:   "pipeline-1",
				Run:            &apiv2beta1.Run{RunId: "run-1"},
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "namespace is required",
		},
		{
			name: "task spec present returns error",
			opts: func() common.Options {
				opts := validOpts()
				opts.Task = &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "some-task"}}
				return opts
			}(),
			wantErr: true,
			errMsg:  "task spec is unnecessary",
		},
		{
			name: "parent task without id returns error",
			opts: func() common.Options {
				opts := validOpts()
				opts.ParentTask = &apiv2beta1.PipelineTask{}
				return opts
			}(),
			wantErr: true,
			errMsg:  "parent task is required",
		},
		{
			name: "container spec present returns error",
			opts: func() common.Options {
				opts := validOpts()
				opts.Container = &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "test-image"}
				return opts
			}(),
			wantErr: true,
			errMsg:  "container spec is unnecessary",
		},
		{
			name: "non-negative iteration index returns error",
			opts: func() common.Options {
				opts := validOpts()
				opts.IterationIndex = 0
				return opts
			}(),
			wantErr: true,
			errMsg:  "iteration index is unnecessary",
		},
		{
			name:    "valid minimal root DAG options pass validation",
			opts:    validOpts(),
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRootDAG(test.opts)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRootDAG_RetryReusesExistingTask(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	clientManager := clientmanager.NewFakeClientManager(fake.NewSimpleClientset(), mockAPI)
	run := &apiv2beta1.Run{RunId: "run-1"}
	mockAPI.AddRun(run)

	pipelineSpec := &pipelinespec.PipelineSpec{Root: &pipelinespec.ComponentSpec{}}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))
	scopePath, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	require.NoError(t, scopePath.Push("root"))

	opts := common.Options{
		PipelineName:   "pipeline-1",
		Run:            run,
		Component:      pipelineSpec.Root,
		RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
		Namespace:      "default",
		IterationIndex: -1,
		ScopePath:      scopePath,
		PodName:        "root-driver-pod",
		PodUID:         "root-driver-uid",
	}

	firstExecution, err := RootDAG(context.Background(), opts, clientManager)
	require.NoError(t, err)
	secondExecution, err := RootDAG(context.Background(), opts, clientManager)
	require.NoError(t, err)
	assert.Equal(t, firstExecution.TaskID, secondExecution.TaskID)

	refreshedRun, err := mockAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)
	require.Len(t, refreshedRun.GetTasks(), 1)
}
