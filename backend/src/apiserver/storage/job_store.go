package storage

import (
	"bytes"
	"fmt"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/jinzhu/gorm"
	"k8s.io/apimachinery/pkg/util/json"
)

type JobStoreInterface interface {
	GetJob(pipelineId string, jobId string) (*model.JobDetail, error)
	ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Job, string, error)
	UpdateJob(workflow *util.Workflow) (err error)
}

type JobStore struct {
	db   *gorm.DB
	time util.TimeInterface
}

// ListJobs list the job metadata for a pipeline from DB
func (s *JobStore) ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Job, string, error) {
	paginationContext, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetJobTablePrimaryKeyColumn(), isDesc)
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
	return s.toJobMetadatas(models), pageToken, err
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStore) GetJob(pipelineId string, jobId string) (*model.JobDetail, error) {
	var job model.JobDetail
	r := s.db.Raw(`SELECT * FROM job_details WHERE PipelineId=? AND UUID=? LIMIT 1`, pipelineId, jobId).Scan(&job)
	if r.RecordNotFound() {
		return nil, util.NewResourceNotFoundError("Job", fmt.Sprint(jobId))
	}
	if r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to get job: %v", r.Error.Error())
	}
	return &job, nil
}

func (s *JobStore) createJob(
	ownerUID string,
	name string,
	namespace string,
	workflowUID string,
	scheduledAtInSec int64,
	condition string,
	marshalled string,
	workflow *util.Workflow) (err error) {
	job := &model.JobDetail{
		Job: model.Job{
			UUID:             workflowUID,
			Name:             name,
			Namespace:        namespace,
			PipelineID:       ownerUID,
			CreatedAtInSec:   workflow.CreationTimestamp.Unix(),
			ScheduledAtInSec: scheduledAtInSec,
			Conditions:       condition,
		},
		Workflow: marshalled,
	}

	r := s.db.Create(job)
	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Error while creating job using workflow: %v, %+v",
			r.Error, workflow.Workflow)
	}

	return nil
}

func (s *JobStore) UpdateJob(workflow *util.Workflow) (err error) {
	if workflow.Name == "" {
		return util.NewInvalidInputError("The workflow must have a name: %+v", workflow.Workflow)
	}
	if workflow.Namespace == "" {
		return util.NewInvalidInputError("The workflow must have a namespace: %+v", workflow.Workflow)
	}
	ownerUID := workflow.ScheduledWorkflowUUIDAsStringOrEmpty()
	if ownerUID == "" {
		return util.NewInvalidInputError("The workflow must have a valid owner: %+v", workflow.Workflow)
	}

	marshalled, err := json.Marshal(workflow.Workflow)
	if err != nil {
		return util.NewInternalServerError(err, "Unable to marshal a workflow: %+v", workflow.Workflow)
	}

	if workflow.UID == "" {
		return util.NewInvalidInputError("The workflow must have a UID: %+v", workflow.Workflow)
	}

	scheduledAtInSec := workflow.ScheduledAtInSecOr0()

	condition := workflow.Condition()

	r := s.db.Exec(`UPDATE job_details SET 
		Name = ?,
		Namespace = ?,
		PipelineID = ?,
		CreatedAtInSec = ?,
		ScheduledAtInSec = ?,
		Conditions = ?,
		Workflow = ?
		WHERE UUID = ?`,
		workflow.Name,
		workflow.Namespace,
		ownerUID,
		workflow.CreationTimestamp.Unix(),
		scheduledAtInSec,
		condition,
		string(marshalled),
		string(workflow.UID))

	if r.Error != nil {
		return util.NewInternalServerError(r.Error, "Error while updating job using workflow: %v, %+v",
			r.Error, workflow.Workflow)
	}

	if r.RowsAffected <= 0 {
		return s.createJob(
			ownerUID,
			workflow.Name,
			workflow.Namespace,
			string(workflow.UID),
			scheduledAtInSec,
			condition,
			string(marshalled),
			workflow)
	}

	return nil
}

func (s *JobStore) queryJobTable(pipelineId string, context *PaginationContext) ([]model.ListableDataModel, error) {
	var jobs []model.JobDetail
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM job_details WHERE PipelineID = '%s'", pipelineId))
	toPaginationQuery("AND", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))
	if r := s.db.Raw(query.String()).Scan(&jobs); r.Error != nil {
		return nil, util.NewInternalServerError(r.Error, "Failed to list jobs: %v", r.Error.Error())
	}
	return s.toListableModels(jobs), nil
}

func (s *JobStore) toListableModels(jobs []model.JobDetail) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(jobs))
	for i := range models {
		models[i] = jobs[i].Job
	}
	return models
}

func (s *JobStore) toJobMetadatas(models []model.ListableDataModel) []model.Job {
	jobMetadatas := make([]model.Job, len(models))
	for i := range models {
		jobMetadatas[i] = models[i].(model.Job)
	}
	return jobMetadatas
}

// factory function for job store
func NewJobStore(db *gorm.DB, time util.TimeInterface) *JobStore {
	return &JobStore{db: db, time: time}
}
