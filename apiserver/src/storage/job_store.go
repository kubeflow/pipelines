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
	"ml/apiserver/src/util"

	"ml/apiserver/src/message/pipelinemanager"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/jinzhu/gorm"
	k8sclient "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStoreInterface interface {
	GetJob(pipelineId uint, jobName string) (*v1alpha1.Workflow, error)
	ListJobs(pipelineId uint) ([]pipelinemanager.Job, error)
	CreateJob(pipelineId uint, wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error)
}

type JobStore struct {
	db       *gorm.DB
	wfClient workflowclient.WorkflowInterface
}

// ListJobs list the job metadata for a pipeline from DB
func (s *JobStore) ListJobs(pipelineId uint) ([]pipelinemanager.Job, error) {
	var jobs []pipelinemanager.Job
	if r := s.db.Where("pipeline_id = ?", pipelineId).Find(&jobs); r.Error != nil {
		return nil, util.NewInternalError("Failed to list jobs.", r.Error.Error())
	}
	return jobs, nil
}

// CreateJob create Workflow by calling CRD, and store the metadata to DB
func (s *JobStore) CreateJob(pipelineId uint, wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	newWf, err := s.wfClient.Create(wf)
	if err != nil {
		return newWf, util.NewInternalError("Failed to create job",
			"Failed to create job . Error: %s", err.Error())
	}
	job := &pipelinemanager.Job{Name: newWf.Name, PipelineID: pipelineId}
	if r := s.db.Create(job); r.Error != nil {
		return newWf, util.NewInternalError("Failed to store job metadata", r.Error.Error())
	}
	return newWf, nil
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStore) GetJob(pipelineId uint, jobName string) (*v1alpha1.Workflow, error) {
	// validate the the pipeline has the job.
	if r := s.db.Where("pipeline_id = ? and name = ?", pipelineId, jobName).
		First(&pipelinemanager.Job{}); r.Error != nil {
		return nil, util.NewResourceNotFoundError("Job", jobName)
	}

	job, err := s.wfClient.Get(jobName, k8sclient.GetOptions{})
	if err != nil {
		return job, util.NewInternalError("Failed to get a job",
			"Failed to get workflow %s from K8s CRD. Error: %s", jobName, err.Error())
	}
	return job, nil
}

// factory function for package store
func NewJobStore(db *gorm.DB, wfClient workflowclient.WorkflowInterface) *JobStore {
	return &JobStore{db: db, wfClient: wfClient}
}
