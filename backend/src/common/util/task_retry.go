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

// CopyDriverRetryGeneration identifies the child authorizing a parent update.
// Unclaimed children still carry their identity; the runtime adapter supplies
// their immutable workflow generation separately from target ownership.
func CopyDriverRetryGeneration(target, source *api.PipelineTask) {
	if target == nil || source == nil || source.GetTaskId() == "" {
		return
	}
	if target.StatusMetadata == nil {
		target.StatusMetadata = &api.PipelineTask_StatusMetadata{}
	}
	if target.StatusMetadata.CustomProperties == nil {
		target.StatusMetadata.CustomProperties = make(map[string]*structpb.Value)
	}
	properties := target.StatusMetadata.CustomProperties
	properties[DriverRetrySourceTaskKey] = structpb.NewStringValue(source.GetTaskId())
	delete(properties, DriverRetrySourceAttemptKey)
	// Generation belongs to the caller, never to the refreshed parent.
	delete(properties, DriverRetryGenerationKey)
	if value := source.GetStatusMetadata().GetCustomProperties()[DriverRetryGenerationKey]; value != nil {
		properties[DriverRetryGenerationKey] = proto.Clone(value).(*structpb.Value)
	}
	if attempt := source.GetStatusMetadata().GetCustomProperties()[DriverRetryAttemptKey]; attempt != nil {
		properties[DriverRetrySourceAttemptKey] = proto.Clone(attempt).(*structpb.Value)
	}
}
