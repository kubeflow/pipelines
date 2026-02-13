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

func TestResolveValuePlaceholders_StringPlaceholder(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--email": structpb.NewStringValue("user@example.com"),
	}

	input := structpb.NewStringValue("{{$.inputs.parameters['pipelinechannel--email']}}")
	result, err := resolveValuePlaceholders(input, resolvedParams)

	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", result.GetStringValue())
}

func TestResolveValuePlaceholders_ListWithPlaceholders(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--email": structpb.NewStringValue("user@example.com"),
	}

	input, _ := structpb.NewList([]interface{}{
		"{{$.inputs.parameters['pipelinechannel--email']}}",
	})
	inputValue := structpb.NewListValue(input)

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.GetListValue().GetValues()))
	assert.Equal(t, "user@example.com", result.GetListValue().GetValues()[0].GetStringValue())
}

func TestResolveValuePlaceholders_ListWithMixedValues(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--name": structpb.NewStringValue("Alice"),
	}

	input, _ := structpb.NewList([]interface{}{
		"static-value",
		"{{$.inputs.parameters['pipelinechannel--name']}}",
		42.0,
	})
	inputValue := structpb.NewListValue(input)

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	values := result.GetListValue().GetValues()
	assert.Equal(t, 3, len(values))
	assert.Equal(t, "static-value", values[0].GetStringValue())
	assert.Equal(t, "Alice", values[1].GetStringValue())
	assert.Equal(t, 42.0, values[2].GetNumberValue())
}

func TestResolveValuePlaceholders_StructWithPlaceholders(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--host": structpb.NewStringValue("db.example.com"),
	}

	input, _ := structpb.NewStruct(map[string]interface{}{
		"host": "{{$.inputs.parameters['pipelinechannel--host']}}",
		"port": 5432.0,
	})
	inputValue := structpb.NewStructValue(input)

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	fields := result.GetStructValue().GetFields()
	assert.Equal(t, "db.example.com", fields["host"].GetStringValue())
	assert.Equal(t, 5432.0, fields["port"].GetNumberValue())
}

func TestResolveValuePlaceholders_NoPlaceholders(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{}

	input := structpb.NewStringValue("plain-string")
	result, err := resolveValuePlaceholders(input, resolvedParams)

	assert.NoError(t, err)
	assert.Equal(t, "plain-string", result.GetStringValue())
}

func TestResolveValuePlaceholders_MissingParam(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{}

	input := structpb.NewStringValue("{{$.inputs.parameters['pipelinechannel--missing']}}")
	_, err := resolveValuePlaceholders(input, resolvedParams)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in resolved inputs")
}

func TestResolveValuePlaceholders_NilValue(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{}

	result, err := resolveValuePlaceholders(nil, resolvedParams)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestResolveValuePlaceholders_ListOfStructsWithPlaceholders(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--email": structpb.NewStringValue("user@example.com"),
	}

	inputValue := structpb.NewListValue(&structpb.ListValue{
		Values: []*structpb.Value{
			structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"email": structpb.NewStringValue("{{$.inputs.parameters['pipelinechannel--email']}}"),
				},
			}),
			structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"email": structpb.NewStringValue("static@example.com"),
				},
			}),
		},
	})

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	values := result.GetListValue().GetValues()
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "user@example.com", values[0].GetStructValue().GetFields()["email"].GetStringValue())
	assert.Equal(t, "static@example.com", values[1].GetStructValue().GetFields()["email"].GetStringValue())
}

func TestResolveValuePlaceholders_StructWithListPlaceholders(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--primary": structpb.NewStringValue("primary@example.com"),
	}

	inputValue := structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"recipients": structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("{{$.inputs.parameters['pipelinechannel--primary']}}"),
					structpb.NewStringValue("backup@example.com"),
				},
			}),
		},
	})

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	recipients := result.GetStructValue().GetFields()["recipients"].GetListValue().GetValues()
	assert.Equal(t, 2, len(recipients))
	assert.Equal(t, "primary@example.com", recipients[0].GetStringValue())
	assert.Equal(t, "backup@example.com", recipients[1].GetStringValue())
}

func TestResolveValuePlaceholders_DeeplyNestedStructures(t *testing.T) {
	resolvedParams := map[string]*structpb.Value{
		"pipelinechannel--email": structpb.NewStringValue("deep@example.com"),
	}

	inputValue := structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"notification": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"recipients": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"email": structpb.NewStringValue("{{$.inputs.parameters['pipelinechannel--email']}}"),
								},
							}),
						},
					}),
					"enabled": structpb.NewBoolValue(true),
				},
			}),
		},
	})

	result, err := resolveValuePlaceholders(inputValue, resolvedParams)

	assert.NoError(t, err)
	notification := result.GetStructValue().GetFields()["notification"].GetStructValue().GetFields()
	assert.Equal(t, true, notification["enabled"].GetBoolValue())
	recipients := notification["recipients"].GetListValue().GetValues()
	assert.Equal(t, 1, len(recipients))
	assert.Equal(t, "deep@example.com", recipients[0].GetStructValue().GetFields()["email"].GetStringValue())
}
