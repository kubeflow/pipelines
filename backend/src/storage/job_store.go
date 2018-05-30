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
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/model"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/jinzhu/gorm"
	k8sclient "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobStoreInterface interface {
	GetJob(pipelineId uint32, jobName string) (*model.JobDetail, error)
	ListJobs(pipelineId uint32, pageToken string, pageSize int, sortByFieldName string) ([]model.Job, string, error)
	CreateJob(pipelineId uint32, wf *v1alpha1.Workflow, scheduledAtInSec int64, createdAtInSec int64) (
		*model.JobDetail, error)
}

type JobStore struct {
	db       *gorm.DB
	wfClient workflowclient.WorkflowInterface
	time     util.TimeInterface
}

// ListJobs list the job metadata for a pipeline from DB
func (s *JobStore) ListJobs(pipelineId uint32, pageToken string, pageSize int, sortByFieldName string) ([]model.Job, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetJobTablePrimaryKeyColumn())
	if err != nil {
		return nil, "", err
	}
	queryJobTable := func(request *PaginationContext) ([]model.ListableDataModel, error) {
		return s.queryJobTable(pipelineId, request)
	}
	models, pageToken, err := listModel(paginationContext, queryJobTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed.")
	}
	return s.toJobs(models), pageToken, err
}

func (s *JobStore) queryJobTable(pipelineId uint32, context *PaginationContext) ([]model.ListableDataModel, error) {
	var jobs []model.Job
	query := s.db.Where("PipelineId = ?", pipelineId)
	paginationQuery, err := toPaginationQuery(query, context)
	if err != nil {
		return nil, util.Wrap(err, "Error creating pagination query when listing jobs.")
	}
	if r := paginationQuery.Limit(context.pageSize).Find(&jobs); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list jobs: %v", r.Error.Error())
	}
	return s.toListableModels(jobs), nil
}

// CreateJob create Workflow by calling CRD, and store the metadata to DB
func (s *JobStore) CreateJob(pipelineId uint32, wf *v1alpha1.Workflow, scheduledAtInSec int64,
	createdAtInSec int64) (*model.JobDetail, error) {

	// TODO: handle the case where the workflow is created but updating the DB fails.

	// Try to schedule the job once
	newWf, err := s.wfClient.Create(wf)
	if err != nil {
		return nil, util.NewInternalServerError(err,
			"Failed to create an Argo workflow for name (%s), generateName (%s).", wf.Name,
			wf.GenerateName)
	}

	// Store the result in the DB
	job := &model.Job{
		CreatedAtInSec:   createdAtInSec,
		UpdatedAtInSec:   createdAtInSec,
		Name:             newWf.Name,
		Status:           model.JobExecutionPending,
		PipelineID:       pipelineId,
		ScheduledAtInSec: scheduledAtInSec,
	}
	if r := s.db.Create(job); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to store job metadata: %v",
			r.Error.Error())
	}
	result := &model.JobDetail{Workflow: newWf, Job: job}
	return result, nil
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStore) GetJob(pipelineId uint32, jobName string) (*model.JobDetail, error) {
	// validate the the pipeline has the job.
	job, err := getJobMetadata(s.db, pipelineId, jobName)
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
	return &model.JobDetail{Workflow: wf, Job: job}, nil
}

// UpdateJobStatus update the job's status field. Only this field is supported to update for now.
func (s *JobStore) updateJobStatus(job *model.Job) (*model.Job, error) {
	newJob := *job
	newJob.UpdatedAtInSec = s.time.Now().Unix()
	r := s.db.Exec(`UPDATE jobs SET Status = ?, UpdatedAtInSec = ? WHERE Name = ?`, newJob.Status, newJob.UpdatedAtInSec, newJob.Name)
	if r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to update the job metadata: %s", r.Error.Error())
	}
	return &newJob, nil
}

func (s *JobStore) toListableModels(jobs []model.Job) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(jobs))
	for i := range models {
		models[i] = jobs[i]
	}
	return models
}

func (s *JobStore) toJobs(models []model.ListableDataModel) []model.Job {
	jobs := make([]model.Job, len(models))
	for i := range models {
		jobs[i] = models[i].(model.Job)
	}
	return jobs
}

func getJobMetadata(db *gorm.DB, pipelineId uint32, jobName string) (*model.Job, error) {
	job := &model.Job{}
	result := db.Raw("SELECT * FROM jobs where PipelineId = ? and Name = ?", pipelineId, jobName).Scan(job)
	if result.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Job", jobName)
	}
	if result.Error != nil {
		// TODO result can return multiple errors. log all of the errors when error handling v2 in place.
		return nil, util.NewInternalServerError(result.Error, "Failed to get job: %v", result.Error.Error())
	}
	return job, nil
}

// factory function for package store
func NewJobStore(db *gorm.DB, wfClient workflowclient.WorkflowInterface,
	time util.TimeInterface) *JobStore {
	return &JobStore{db: db, wfClient: wfClient, time: time}
}
