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

func Test_validateDAG(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "container spec present returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
			},
			wantErr: true,
			errMsg:  "container spec is unnecessary",
		},
		{
			name: "missing pipeline name returns error",
			opts: Options{
				PipelineName: "",
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "",
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "missing component spec returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    nil,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "missing task spec returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    &pipelinespec.ComponentSpec{},
				Task:         nil,
			},
			wantErr: true,
			errMsg:  "task spec is required",
		},
		{
			name: "missing DAG execution ID returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-1"}},
				DAGExecutionID: 0,
			},
			wantErr: true,
			errMsg:  "DAG execution ID is required",
		},
		{
			name: "valid DAG options pass validation",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-1"}},
				DAGExecutionID: 1,
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDAG(test.opts)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
