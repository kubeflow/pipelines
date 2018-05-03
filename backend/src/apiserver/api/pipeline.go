// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import "ml/backend/src/model"

type Pipeline struct {
	ID             uint        `json:"id"`
	CreatedAtInSec int64       `json:"createdAt"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	PackageId      uint        `json:"packageId"`
	Schedule       string      `json:"schedule"`
	Enabled        bool        `json:"enabled"`
	EnabledAtInSec int64       `json:"enabledAt"`
	Parameters     []Parameter `json:"parameters,omitempty"`
}

func ToApiPipeline(pipeline *model.Pipeline) *Pipeline {
	return &Pipeline{
		ID:             pipeline.ID,
		CreatedAtInSec: pipeline.CreatedAtInSec,
		Name:           pipeline.Name,
		Description:    pipeline.Description,
		PackageId:      pipeline.PackageId,
		Schedule:       pipeline.Schedule,
		Enabled:        pipeline.Enabled,
		EnabledAtInSec: pipeline.EnabledAtInSec,
		Parameters:     toApiParameters(pipeline.Parameters),
	}
}

func ToApiPipelines(pipelines []model.Pipeline) []Pipeline {
	apiPipelines := make([]Pipeline, 0)
	for _, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, *ToApiPipeline(&pipeline))
	}
	return apiPipelines
}

func ToModelPipeline(pipeline *Pipeline) *model.Pipeline {
	return &model.Pipeline{
		ID:             pipeline.ID,
		CreatedAtInSec: pipeline.CreatedAtInSec,
		Name:           pipeline.Name,
		Description:    pipeline.Description,
		PackageId:      pipeline.PackageId,
		Schedule:       pipeline.Schedule,
		Enabled:        pipeline.Enabled,
		EnabledAtInSec: pipeline.EnabledAtInSec,
		Parameters:     toModelParameters(pipeline.Parameters),
	}
}
