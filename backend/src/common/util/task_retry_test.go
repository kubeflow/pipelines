// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCopyDriverRetryGeneration(t *testing.T) {
	source := &api.PipelineTask{StatusMetadata: &api.PipelineTask_StatusMetadata{CustomProperties: map[string]*structpb.Value{
		DriverRetryGenerationKey: structpb.NewStringValue("7"),
		"_kfp_driver_checkpoint": structpb.NewStringValue("child-only"),
	}}}
	target := &api.PipelineTask{StatusMetadata: &api.PipelineTask_StatusMetadata{
		Message: "parent message",
		CustomProperties: map[string]*structpb.Value{
			DriverRetryGenerationKey: structpb.NewStringValue("8"),
			"_kfp_driver_checkpoint": structpb.NewStringValue("parent-only"),
		},
	}}
	CopyDriverRetryGeneration(target, source)
	assert.Equal(t, "7", target.GetStatusMetadata().GetCustomProperties()[DriverRetryGenerationKey].GetStringValue())
	assert.Equal(t, "parent-only", target.GetStatusMetadata().GetCustomProperties()["_kfp_driver_checkpoint"].GetStringValue())
	assert.Equal(t, "parent message", target.GetStatusMetadata().GetMessage())
	target.StatusMetadata.CustomProperties[DriverRetryGenerationKey].Kind = &structpb.Value_StringValue{StringValue: "9"}
	assert.Equal(t, "7", source.GetStatusMetadata().GetCustomProperties()[DriverRetryGenerationKey].GetStringValue())
	CopyDriverRetryGeneration(target, &api.PipelineTask{})
	assert.Equal(t, "9", target.GetStatusMetadata().GetCustomProperties()[DriverRetryGenerationKey].GetStringValue())
	CopyDriverRetryGeneration(nil, source)
	CopyDriverRetryGeneration(target, nil)
}

func TestCopyDriverRetryGenerationFencesSourceWithoutReplacingTargetClaim(t *testing.T) {
	source := &api.PipelineTask{TaskId: "child", StatusMetadata: &api.PipelineTask_StatusMetadata{
		CustomProperties: map[string]*structpb.Value{
			DriverRetryGenerationKey: structpb.NewStringValue("7"),
			DriverRetryAttemptKey:    structpb.NewStringValue("2"),
		},
	}}
	target := &api.PipelineTask{TaskId: "parent", StatusMetadata: &api.PipelineTask_StatusMetadata{
		CustomProperties: map[string]*structpb.Value{
			DriverRetryGenerationKey: structpb.NewStringValue("7"),
			DriverRetryAttemptKey:    structpb.NewStringValue("0"),
			"_kfp_driver_checkpoint": structpb.NewStringValue("parent-handoff"),
		},
	}}
	CopyDriverRetryGeneration(target, source)
	properties := target.GetStatusMetadata().GetCustomProperties()
	assert.Equal(t, "child", properties[DriverRetrySourceTaskKey].GetStringValue())
	assert.Equal(t, "2", properties[DriverRetrySourceAttemptKey].GetStringValue())
	assert.Equal(t, "0", properties[DriverRetryAttemptKey].GetStringValue())
	assert.Equal(t, "parent-handoff", properties["_kfp_driver_checkpoint"].GetStringValue())
	properties[DriverRetrySourceAttemptKey].Kind = &structpb.Value_StringValue{StringValue: "3"}
	assert.Equal(t, "2", source.GetStatusMetadata().GetCustomProperties()[DriverRetryAttemptKey].GetStringValue())

	delete(source.StatusMetadata.CustomProperties, DriverRetryAttemptKey)
	CopyDriverRetryGeneration(target, source)
	assert.NotContains(t, properties, DriverRetrySourceTaskKey)
	assert.NotContains(t, properties, DriverRetrySourceAttemptKey)
	assert.Equal(t, "0", properties[DriverRetryAttemptKey].GetStringValue())
}
