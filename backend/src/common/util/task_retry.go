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
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// CopyDriverRetryGeneration carries the originating attempt's fence onto a
// dependent task write without copying task-specific recovery checkpoints or
// replacing the target's own attempt claim.
func CopyDriverRetryGeneration(target, source *api.PipelineTask) {
	value, present := source.GetStatusMetadata().GetCustomProperties()[DriverRetryGenerationKey]
	if target == nil || !present {
		return
	}
	if target.StatusMetadata == nil {
		target.StatusMetadata = &api.PipelineTask_StatusMetadata{}
	}
	if target.StatusMetadata.CustomProperties == nil {
		target.StatusMetadata.CustomProperties = make(map[string]*structpb.Value)
	}
	if value != nil {
		value = proto.Clone(value).(*structpb.Value)
	}
	target.StatusMetadata.CustomProperties[DriverRetryGenerationKey] = value
	delete(target.StatusMetadata.CustomProperties, DriverRetrySourceTaskKey)
	delete(target.StatusMetadata.CustomProperties, DriverRetrySourceAttemptKey)
	if attempt, present := source.GetStatusMetadata().GetCustomProperties()[DriverRetryAttemptKey]; present {
		target.StatusMetadata.CustomProperties[DriverRetrySourceTaskKey] = structpb.NewStringValue(source.GetTaskId())
		if attempt != nil {
			attempt = proto.Clone(attempt).(*structpb.Value)
		}
		target.StatusMetadata.CustomProperties[DriverRetrySourceAttemptKey] = attempt
	}
}
