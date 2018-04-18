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

import (
	"ml/src/model"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Job metadata of a job.
type Job struct {
	Name             string `json:"name"`
	CreatedAtInSec   int64  `json:"createdAt"`
	ScheduledAtInSec int64  `json:"scheduledAt"`
}

// JobDetail a detailed view of a job, including templates, job status etc.
type JobDetail struct {
	Workflow *v1alpha1.Workflow `json:"jobDetail"`
	Job      *Job               `json:"metadata"`
}

func ToApiJob(job *model.Job) *Job {
	// We don't expose the status of the job for now, since the Sync service is not in place yet to
	// sync the status of a job from K8s CRD to the DB. We only use Status column to track whether
	// K8s resource is created successfully.
	return &Job{
		Name:             job.Name,
		CreatedAtInSec:   job.CreatedAtInSec,
		ScheduledAtInSec: job.ScheduledAtInSec,
	}
}

func ToApiJobs(jobs []model.Job) []Job {
	apiJobs := make([]Job, 0)
	for _, job := range jobs {
		apiJobs = append(apiJobs, *ToApiJob(&job))
	}
	return apiJobs
}

func ToApiJobDetail(jobDetail *model.JobDetail) *JobDetail {
	return &JobDetail{
		Workflow: jobDetail.Workflow,
		Job:      ToApiJob(jobDetail.Job),
	}
}
