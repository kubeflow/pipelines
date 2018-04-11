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

package storage

import (
	"ml/src/message"
	"ml/src/util"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/jinzhu/gorm"
	k8sclient "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStoreInterface interface {
	GetJob(pipelineId uint, jobName string) (*message.JobDetail, error)
	ListJobs(pipelineId uint) ([]message.Job, error)
	CreateJob(pipelineId uint, wf *v1alpha1.Workflow, scheduledAtInSec int64) (
		*message.JobDetail, error)
}

type JobStore struct {
	db       *gorm.DB
	wfClient workflowclient.WorkflowInterface
	time     util.TimeInterface
}

// ListJobs list the job metadata for a pipeline from DB
func (s *JobStore) ListJobs(pipelineId uint) ([]message.Job, error) {
	var jobs []message.Job
	if r := s.db.Where("pipeline_id = ?", pipelineId).Find(&jobs); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list jobs: %v", r.Error.Error())
	}
	return jobs, nil
}

// CreateJob create Workflow by calling CRD, and store the metadata to DB
func (s *JobStore) CreateJob(pipelineId uint, wf *v1alpha1.Workflow, scheduledAtInSec int64) (
	*message.JobDetail, error) {
	newWf, err := s.wfClient.Create(wf)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create job . Error: %s", err.Error())
	}
	job := &message.Job{
		Name:             newWf.Name,
		PipelineID:       pipelineId,
		ScheduledAtInSec: scheduledAtInSec}
	if r := s.db.Create(job); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to store job metadata: %v",
			r.Error.Error())
	}
	return &message.JobDetail{Workflow: newWf, Job: job}, nil
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStore) GetJob(pipelineId uint, jobName string) (*message.JobDetail, error) {
	// validate the the pipeline has the job.
	job := &message.Job{}
	err := queryJob(s.db, pipelineId, jobName, job)
	if err != nil {
		return nil, err
	}

	wf, err := s.wfClient.Get(jobName, k8sclient.GetOptions{})
	if err != nil {
		// We should always expect the job to exist. In case the job is not found, it's an
		// unexpected scenario that implies something is wrong internally. So we don't differentiate
		// resource not found or other exceptions here, just always return internal error.
		return nil, util.NewInternalServerError(err,
			"Failed to get workflow %s from K8s CRD. Error: %s", jobName, err.Error())
	}
	return &message.JobDetail{Workflow: wf, Job: job}, nil
}

func queryJob(db *gorm.DB, pipelineId uint, jobName string, job *message.Job) error {
	result := db.Where("pipeline_id = ? and name = ?", pipelineId, jobName).First(job)
	if result.RecordNotFound() {
		return util.NewResourceNotFoundError("Job", jobName)
	}
	if result.Error != nil {
		// TODO result can return multiple errors. log all of the errors when error handling v2 in place.
		return util.NewInternalServerError(result.Error, "Failed to get job: %v", result.Error.Error())
	}
	return nil
}

// factory function for package store
func NewJobStore(db *gorm.DB, wfClient workflowclient.WorkflowInterface,
	time util.TimeInterface) *JobStore {
	return &JobStore{db: db, wfClient: wfClient, time: time}
}
