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
	"github.com/stretchr/testify/assert"
)

func Test_validateRootDAG(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing pipeline name returns error",
			opts: Options{
				PipelineName:   "",
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "",
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "nil component spec returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      nil,
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "nil runtime config returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  nil,
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "runtime config is required",
		},
		{
			name: "missing namespace returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:      "",
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "namespace is required",
		},
		{
			name: "task spec present returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:      "default",
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "some-task"}},
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "task spec is unnecessary",
		},
		{
			name: "non-zero DAG execution ID returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:      "default",
				DAGExecutionID: 42,
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "DAG execution ID is unnecessary",
		},
		{
			name: "container spec present returns error",
			opts: Options{
				PipelineName:  "pipeline-1",
				RunID:         "run-1",
				Component:     &pipelinespec.ComponentSpec{},
				RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:     "default",
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				IterationIndex: -1,
			},
			wantErr: true,
			errMsg:  "container spec is unnecessary",
		},
		{
			name: "non-negative iteration index returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:      "default",
				IterationIndex: 0,
			},
			wantErr: true,
			errMsg:  "iteration index is unnecessary",
		},
		{
			name: "valid minimal root DAG options pass validation",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				Namespace:      "default",
				IterationIndex: -1,
			},
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
