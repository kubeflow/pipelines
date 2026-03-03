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

func Test_validateContainer(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil container spec returns error",
			opts: Options{
				Container: nil,
			},
			wantErr: true,
			errMsg:  "container spec is required",
		},
		{
			name: "missing pipeline name returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "",
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "pipeline-1",
				RunID:        "",
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "missing component spec returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    nil,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "valid container options pass validation",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
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
			err := validateContainer(test.opts)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
