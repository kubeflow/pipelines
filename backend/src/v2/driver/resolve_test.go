// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolvePipelineJobCreateTimePlaceholder(t *testing.T) {
	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_create_time_utc}}"),
				},
			},
		},
	}

	opts := common.Options{
		PipelineJobCreateTimeUTC: "2026-01-09T12:34:56Z",
		ParentTask:               &apiv2beta1.PipelineTaskDetail{Name: "parent"},
		Task:                     &pipelinespec.PipelineTaskSpec{Inputs: &pipelinespec.TaskInputsSpec{}},
	}

	val, _, err := resolver.ResolveInputParameter(opts, paramSpec, nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-09T12:34:56Z", val.GetValue().GetStringValue())
}

func TestResolvePipelineJobScheduleTimePlaceholder(t *testing.T) {
	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_schedule_time_utc}}"),
				},
			},
		},
	}

	opts := common.Options{
		PipelineJobScheduleTimeUTC: "2026-01-09T13:00:00Z",
		ParentTask:                 &apiv2beta1.PipelineTaskDetail{Name: "parent"},
		Task:                       &pipelinespec.PipelineTaskSpec{Inputs: &pipelinespec.TaskInputsSpec{}},
	}

	val, _, err := resolver.ResolveInputParameter(opts, paramSpec, nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-09T13:00:00Z", val.GetValue().GetStringValue())
}
