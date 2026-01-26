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
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolvePipelineJobCreateTimePlaceholder(t *testing.T) {
	ctx := context.Background()

	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_create_time_utc}}"),
				},
			},
		},
	}

	opts := Options{
		PipelineJobCreateTimeUTC: "2026-01-09T12:34:56Z",
	}

	val, err := resolveInputParameter(
		ctx,
		nil, // task
		nil, // component spec
		opts,
		nil, // runtime config
		paramSpec,
		map[string]*structpb.Value{},
	)

	assert.NoError(t, err)
	assert.Equal(t, "2026-01-09T12:34:56Z", val.GetStringValue())
}

func TestResolvePipelineJobScheduleTimePlaceholder(t *testing.T) {
	ctx := context.Background()

	paramSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("{{$.pipeline_job_schedule_time_utc}}"),
				},
			},
		},
	}

	opts := Options{
		PipelineJobScheduleTimeUTC: "2026-01-09T13:00:00Z",
	}

	val, err := resolveInputParameter(
		ctx,
		nil,
		nil,
		opts,
		nil,
		paramSpec,
		map[string]*structpb.Value{},
	)

	assert.NoError(t, err)
	assert.Equal(t, "2026-01-09T13:00:00Z", val.GetStringValue())
}
