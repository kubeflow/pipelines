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

package main

import (
	"encoding/json"
	"ml/backend/api"
	"ml/backend/src/model"
	"ml/backend/src/util"

	"github.com/golang/protobuf/ptypes/timestamp"
)

func ToApiJob(job *model.Job) *api.Job {
	// We don't expose the status of the job for now, since the Sync service is not in place yet to
	// sync the status of a job from K8s CRD to the DB. We only use Status column to track whether
	// K8s resource is created successfully.
	return &api.Job{
		Name:        job.Name,
		CreatedAt:   &timestamp.Timestamp{Seconds: job.CreatedAtInSec},
		ScheduledAt: &timestamp.Timestamp{Seconds: job.ScheduledAtInSec},
	}
}

func ToApiJobs(jobs []model.Job) []*api.Job {
	apiJobs := make([]*api.Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, ToApiJob(&job))
	}
	return apiJobs
}

func ToApiJobDetail(jobDetail *model.JobDetail) (*api.JobDetail, error) {
	workflow, err := json.Marshal(jobDetail.Workflow)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal job details back to client.")
	}
	return &api.JobDetail{
		Job:      ToApiJob(jobDetail.Job),
		Workflow: string(workflow),
	}, nil
}

func ToApiPackage(pkg *model.Package) *api.Package {
	return &api.Package{
		Id:          pkg.ID,
		CreatedAt:   &timestamp.Timestamp{Seconds: pkg.CreatedAtInSec},
		Name:        pkg.Name,
		Description: pkg.Description,
		Parameters:  toApiParameters(pkg.Parameters),
	}
}

func ToApiPackages(pkgs []model.Package) []*api.Package {
	apiPkgs := make([]*api.Package, 0)
	for _, pkg := range pkgs {
		apiPkgs = append(apiPkgs, ToApiPackage(&pkg))
	}
	return apiPkgs
}

func ToApiPipeline(pipeline *model.Pipeline) *api.Pipeline {
	return &api.Pipeline{
		Id:          pipeline.ID,
		CreatedAt:   &timestamp.Timestamp{Seconds: pipeline.CreatedAtInSec},
		Name:        pipeline.Name,
		Description: pipeline.Description,
		PackageId:   pipeline.PackageId,
		Schedule:    pipeline.Schedule,
		Enabled:     pipeline.Enabled,
		EnabledAt:   &timestamp.Timestamp{Seconds: pipeline.EnabledAtInSec},
		Parameters:  toApiParameters(pipeline.Parameters),
	}
}

func ToApiPipelines(pipelines []model.Pipeline) []*api.Pipeline {
	apiPipelines := make([]*api.Pipeline, 0)
	for _, pipeline := range pipelines {
		apiPipelines = append(apiPipelines, ToApiPipeline(&pipeline))
	}
	return apiPipelines
}

func ToModelPipeline(pipeline *api.Pipeline) *model.Pipeline {
	return &model.Pipeline{
		ID:          pipeline.Id,
		Name:        pipeline.Name,
		Description: pipeline.Description,
		PackageId:   pipeline.PackageId,
		Schedule:    pipeline.Schedule,
		Enabled:     pipeline.Enabled,
		Parameters:  toModelParameters(pipeline.Parameters),
	}
}

func toApiParameters(params []model.Parameter) []*api.Parameter {
	apiParams := make([]*api.Parameter, 0)
	for _, param := range params {
		var value string
		if param.Value != nil {
			value = *param.Value
		}
		apiParam := api.Parameter{
			Name:  param.Name,
			Value: value,
		}
		apiParams = append(apiParams, &apiParam)
	}
	return apiParams
}

func toModelParameters(params []*api.Parameter) []model.Parameter {
	modelParams := make([]model.Parameter, 0)
	for _, param := range params {
		modelParam := model.Parameter{
			Name:  param.Name,
			Value: &param.Value,
		}
		modelParams = append(modelParams, modelParam)
	}
	return modelParams
}
