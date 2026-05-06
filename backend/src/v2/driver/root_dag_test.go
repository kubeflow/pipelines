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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
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
				opts.ParentTask = &apiv2beta1.PipelineTaskDetail{}
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
