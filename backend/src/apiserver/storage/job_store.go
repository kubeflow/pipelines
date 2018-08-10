package storage

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/json"
)

type JobStoreInterface interface {
	GetJob(pipelineId string, jobId string) (*model.JobDetail, error)
	ListJobs(pipelineId string, pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Job, string, error)
	CreateOrUpdateJob(workflow *util.Workflow) (err error)
}

type JobStore struct {
	db   *sql.DB
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

func (s *JobStore) queryJobTable(pipelineId string, context *PaginationContext) ([]model.ListableDataModel, error) {
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM job_details WHERE PipelineID = '%s'", pipelineId))
	toPaginationQuery("AND", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))
	r, err := s.db.Query(query.String())
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list jobs: %v", err.Error())
	}
	defer r.Close()
	jobs, err := s.scanRows(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list jobs: %v", err.Error())
	}

	return s.toListableModels(jobs), nil
}

// GetJob Get the job manifest from Workflow CRD
func (s *JobStore) GetJob(pipelineId string, jobId string) (*model.JobDetail, error) {
	r, err := s.db.Query(`SELECT * FROM job_details WHERE PipelineId=? AND uuid=? LIMIT 1`, pipelineId, jobId)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get job: %v", err.Error())
	}
	defer r.Close()
	jobs, err := s.scanRows(r)

	if err != nil || len(jobs) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get job: %v", err.Error())
	}
	if len(jobs) == 0 {
		return nil, util.NewResourceNotFoundError("Job", fmt.Sprint(jobId))
	}
	return &jobs[0], nil
}

func (s *JobStore) scanRows(rows *sql.Rows) ([]model.JobDetail, error) {
	var jobs []model.JobDetail
	for rows.Next() {
		var uuid, name, namespace, pipelineID, conditions, workflow string
		var CreatedAtInSec, ScheduledAtInSec int64
		err := rows.Scan(&uuid, &name, &namespace, &pipelineID, &CreatedAtInSec, &ScheduledAtInSec, &conditions, &workflow)
		if err != nil {
			return jobs, nil
		}
		jobs = append(jobs, model.JobDetail{Job: model.Job{
			UUID:             uuid,
			Name:             name,
			Namespace:        namespace,
			PipelineID:       pipelineID,
			CreatedAtInSec:   CreatedAtInSec,
			ScheduledAtInSec: ScheduledAtInSec,
			Conditions:       conditions},
			Workflow: workflow})
	}
	return jobs, nil
}

func (s *JobStore) createJob(
	ownerUID string,
	name string,
	namespace string,
	workflowUID string,
	createdAtInSec int64,
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
			CreatedAtInSec:   createdAtInSec,
			ScheduledAtInSec: scheduledAtInSec,
			Conditions:       condition,
		},
		Workflow: marshalled,
	}

	stmt, err := s.db.Prepare(
		`INSERT INTO job_details(UUID,Name,Namespace,PipelineID,CreatedAtInSec,ScheduledAtInSec,Conditions,Workflow) 
		VALUES(?,?,?,?,?,?,?,?)`)
	if err != nil {
		return util.NewInternalServerError(err, "Error while creating job using workflow: %v, %+v",
			err, workflow.Workflow)
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		job.UUID,
		job.Name,
		job.Namespace,
		job.PipelineID,
		job.CreatedAtInSec,
		job.ScheduledAtInSec,
		job.Conditions,
		job.Workflow)
	if err != nil {
		return util.NewInternalServerError(err, "Error while creating job for workflow: '%v/%v",
			namespace, name)
	}

	return nil
}

func (s *JobStore) CreateOrUpdateJob(workflow *util.Workflow) (err error) {
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

	// Attempting to create the job in the DB.

	createError := s.createJob(
		ownerUID,
		workflow.Name,
		workflow.Namespace,
		string(workflow.UID),
		workflow.CreationTimestamp.Unix(),
		scheduledAtInSec,
		condition,
		string(marshalled),
		workflow)

	if createError == nil {
		return nil
	}

	// If creating the job did not work, attempting to update the job in the DB.

	stmt, err := s.db.Prepare(`UPDATE job_details SET 
		Name = ?,
		Namespace = ?,
		PipelineID = ?,
		CreatedAtInSec = ?,
		ScheduledAtInSec = ?,
		Conditions = ?,
		Workflow = ?
		WHERE UUID = ?`)
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while creating or updating job for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			workflow.Namespace, workflow.Name, createError.Error(), err.Error())
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		workflow.Name,
		workflow.Namespace,
		ownerUID,
		workflow.CreationTimestamp.Unix(),
		scheduledAtInSec,
		condition,
		string(marshalled),
		string(workflow.UID))

	if err != nil {
		return util.NewInternalServerError(err,
			"Error while creating or updating job for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			workflow.Namespace, workflow.Name, createError.Error(), err.Error())
	}

	return nil
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
func NewJobStore(db *sql.DB, time util.TimeInterface) *JobStore {
	return &JobStore{db: db, time: time}
}
